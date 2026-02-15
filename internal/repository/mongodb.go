package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/skoredin/db-benchmark-suite/internal/config"
	"github.com/skoredin/db-benchmark-suite/internal/generator"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoDBRepo struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewMongoDBRepo(ctx context.Context, cfg config.MongoDBConfig) (*MongoDBRepo, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)

		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	collection := client.Database(cfg.Database).Collection("events")

	return &MongoDBRepo{
		client:     client,
		collection: collection,
	}, nil
}

func (r *MongoDBRepo) InitSchema(ctx context.Context) error {
	_ = r.collection.Drop(ctx)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "created_at", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "event_type", Value: 1},
				{Key: "created_at", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)

	return err
}

func (r *MongoDBRepo) InsertBatch(ctx context.Context, events []generator.Event) error {
	docs := make([]bson.M, len(events))
	for i, event := range events {
		docs[i] = bson.M{
			"event_id":   event.ID,
			"user_id":    event.UserID,
			"event_type": event.EventType,
			"payload":    event.Payload,
			"created_at": event.CreatedAt,
		}
	}

	opts := options.InsertMany().SetOrdered(false)

	_, err := r.collection.InsertMany(ctx, docs, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}

		return err
	}

	return nil
}

func (r *MongoDBRepo) GetEventStats(ctx context.Context, start, end time.Time) ([]EventStats, error) {
	pipeline := eventStatsPipeline(start, end)

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	defer func() { _ = cursor.Close(ctx) }()

	return decodeEventStats(ctx, cursor)
}

func eventStatsPipeline(start, end time.Time) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "created_at", Value: bson.D{
				{Key: "$gte", Value: start},
				{Key: "$lte", Value: end},
			}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "hour", Value: bson.D{
					{Key: "$dateTrunc", Value: bson.D{
						{Key: "date", Value: "$created_at"},
						{Key: "unit", Value: "hour"},
					}},
				}},
				{Key: "type", Value: "$event_type"},
			}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "unique_users", Value: bson.D{{Key: "$addToSet", Value: "$user_id"}}},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "hour", Value: "$_id.hour"},
			{Key: "event_type", Value: "$_id.type"},
			{Key: "count", Value: 1},
			{Key: "unique_users", Value: bson.D{{Key: "$size", Value: "$unique_users"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "hour", Value: -1}}}},
	}
}

func decodeEventStats(ctx context.Context, cursor *mongo.Cursor) ([]EventStats, error) {
	var stats []EventStats

	for cursor.Next(ctx) {
		var result struct {
			Hour        time.Time `bson:"hour"`
			EventType   string    `bson:"event_type"`
			Count       int64     `bson:"count"`
			UniqueUsers int64     `bson:"unique_users"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}

		stats = append(stats, EventStats{
			Hour:        result.Hour,
			EventType:   result.EventType,
			Count:       result.Count,
			UniqueUsers: result.UniqueUsers,
		})
	}

	return stats, cursor.Err()
}

func (r *MongoDBRepo) GetStorageStats(ctx context.Context) *StorageStats {
	var result bson.M

	err := r.collection.Database().RunCommand(ctx, bson.D{
		{Key: "collStats", Value: "events"},
	}).Decode(&result)
	if err != nil {
		return &StorageStats{}
	}

	count, _ := r.collection.CountDocuments(ctx, bson.D{})

	stats := &StorageStats{
		RowCount: count,
	}

	stats.TotalSize = bsonToInt64(result, "size")
	stats.IndexSize = bsonToInt64(result, "totalIndexSize")

	storageSize := bsonToInt64(result, "storageSize")
	if stats.TotalSize > 0 {
		stats.CompressionPct = (1 - float64(storageSize)/float64(stats.TotalSize)) * 100
	}

	return stats
}

func bsonToInt64(m bson.M, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}

	switch n := v.(type) {
	case int32:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func (r *MongoDBRepo) Cleanup(ctx context.Context) error {
	return r.collection.Drop(ctx)
}

func (r *MongoDBRepo) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return r.client.Disconnect(ctx)
}
