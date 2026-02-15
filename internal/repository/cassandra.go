package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/skoredin/db-benchmark-suite/internal/config"
	"github.com/skoredin/db-benchmark-suite/internal/generator"
)

// cqlQuoteIdentifier escapes double quotes inside a CQL identifier.
func cqlQuoteIdentifier(s string) string {
	return strings.ReplaceAll(s, `"`, `""`)
}

type CassandraRepo struct {
	session *gocql.Session
}

func NewCassandraRepo(_ context.Context, cfg config.CassandraConfig) (*CassandraRepo, error) {
	cluster := newCassandraCluster(cfg)

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create cassandra session: %w", err)
	}

	if err := createKeyspace(session, cfg.Keyspace); err != nil {
		session.Close()
		return nil, err
	}

	session.Close()

	cluster.Keyspace = cfg.Keyspace

	session, err = cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect to keyspace: %w", err)
	}

	return &CassandraRepo{session: session}, nil
}

func newCassandraCluster(cfg config.CassandraConfig) *gocql.ClusterConfig {
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Keyspace = "system"
	cluster.Consistency = gocql.LocalOne
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = 10 * time.Second
	cluster.Timeout = 30 * time.Second
	cluster.NumConns = 2
	cluster.DisableInitialHostLookup = true
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{NumRetries: 3, Min: 500 * time.Millisecond, Max: 5 * time.Second}

	return cluster
}

func createKeyspace(session *gocql.Session, keyspace string) error {
	keyspaceQuery := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS "%s"
		WITH replication = {
			'class': 'SimpleStrategy',
			'replication_factor': 1
		}
	`, cqlQuoteIdentifier(keyspace))

	if err := session.Query(keyspaceQuery).Exec(); err != nil {
		return fmt.Errorf("failed to create keyspace: %w", err)
	}

	return nil
}

func (r *CassandraRepo) InitSchema(ctx context.Context) error {
	_ = r.session.Query("DROP TABLE IF EXISTS events").WithContext(ctx).Exec()

	schema := `
		CREATE TABLE IF NOT EXISTS events (
			date_bucket text,
			created_at timestamp,
			event_id text,
			user_id bigint,
			event_type text,
			payload text,
			PRIMARY KEY ((date_bucket), event_type, created_at, event_id)
		) WITH CLUSTERING ORDER BY (event_type ASC, created_at DESC)
		AND compaction = {
			'class': 'TimeWindowCompactionStrategy',
			'compaction_window_size': 1,
			'compaction_window_unit': 'DAYS'
		}
	`

	return r.session.Query(schema).WithContext(ctx).Exec()
}

func (r *CassandraRepo) InsertBatch(ctx context.Context, events []generator.Event) error {
	for _, event := range events {
		bucket := event.CreatedAt.Format("20060102")
		if err := r.session.Query(`
			INSERT INTO events (date_bucket, created_at, event_id, user_id, event_type, payload)
			VALUES (?, ?, ?, ?, ?, ?)`,
			bucket, event.CreatedAt, event.ID, event.UserID, event.EventType, event.Payload,
		).WithContext(ctx).Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (r *CassandraRepo) GetEventStats(ctx context.Context, start, end time.Time) ([]EventStats, error) {
	var stats []EventStats

	current := start
	for current.Before(end) || current.Equal(end) {
		bucket := current.Format("20060102")

		query := `
			SELECT date_bucket, event_type, COUNT(*)
			FROM events
			WHERE date_bucket = ?
			GROUP BY date_bucket, event_type
		`

		iter := r.session.Query(query, bucket).WithContext(ctx).Iter()

		var (
			dateBucket string
			eventType  string
			cnt        int64
		)

		for iter.Scan(&dateBucket, &eventType, &cnt) {
			stats = append(stats, EventStats{
				Hour:        current.Truncate(24 * time.Hour),
				EventType:   eventType,
				Count:       cnt,
				UniqueUsers: 0,
			})
		}

		if err := iter.Close(); err != nil {
			return nil, err
		}

		current = current.AddDate(0, 0, 1)
	}

	return stats, nil
}

func (r *CassandraRepo) GetStorageStats(ctx context.Context) *StorageStats {
	var stats StorageStats

	if err := r.session.Query("SELECT COUNT(*) FROM events").WithContext(ctx).Scan(&stats.RowCount); err != nil {
		return &stats
	}

	sizeQuery := `
		SELECT mean_partition_size, partitions_count
		FROM system.size_estimates
		WHERE keyspace_name = 'events'
		AND table_name = 'events'
	`
	iter := r.session.Query(sizeQuery).Iter()

	var meanSize, partCount int64

	var totalSize int64

	for iter.Scan(&meanSize, &partCount) {
		totalSize += meanSize * partCount
	}

	if err := iter.Close(); err != nil {
		return &stats
	}

	if totalSize > 0 {
		stats.TotalSize = totalSize
	} else {
		stats.TotalSize = stats.RowCount * 200
	}

	return &stats
}

func (r *CassandraRepo) Cleanup(ctx context.Context) error {
	return r.session.Query("TRUNCATE TABLE events").WithContext(ctx).Exec()
}

func (r *CassandraRepo) Close() error {
	r.session.Close()
	return nil
}
