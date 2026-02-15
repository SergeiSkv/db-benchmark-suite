package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/skoredin/db-benchmark-suite/internal/config"
	"github.com/skoredin/db-benchmark-suite/internal/generator"
)

type ClickHouseRepo struct {
	conn driver.Conn
}

func NewClickHouseRepo(ctx context.Context, cfg *config.ClickHouseConfig) (*ClickHouseRepo, error) {
	if err := createClickHouseDB(ctx, cfg); err != nil {
		return nil, err
	}

	return connectClickHouse(ctx, cfg)
}

func createClickHouseDB(ctx context.Context, cfg *config.ClickHouseConfig) error {
	initConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: cfg.User,
			Password: cfg.Password,
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	if err := initConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", cfg.Database)); err != nil {
		_ = initConn.Close()

		return fmt.Errorf("failed to create database: %w", err)
	}

	return initConn.Close()
}

func connectClickHouse(ctx context.Context, cfg *config.ClickHouseConfig) (*ClickHouseRepo, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:      5 * time.Second,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return &ClickHouseRepo{conn: conn}, nil
}

func (r *ClickHouseRepo) InitSchema(ctx context.Context) error {
	if err := r.conn.Exec(ctx, "DROP TABLE IF EXISTS events"); err != nil {
		return err
	}

	schema := `
		CREATE TABLE IF NOT EXISTS events (
			event_id String,
			user_id UInt64,
			event_type LowCardinality(String),
			payload String,
			created_at DateTime
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(created_at)
		ORDER BY (event_type, created_at, user_id)
		SETTINGS index_granularity = 8192
	`

	return r.conn.Exec(ctx, schema)
}

func (r *ClickHouseRepo) InsertBatch(ctx context.Context, events []generator.Event) error {
	batch, err := r.conn.PrepareBatch(ctx, "INSERT INTO events")
	if err != nil {
		return err
	}

	for _, event := range events {
		err := batch.Append(
			event.ID,
			safeInt64ToUint64(event.UserID),
			event.EventType,
			event.Payload,
			event.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

func (r *ClickHouseRepo) GetEventStats(ctx context.Context, start, end time.Time) ([]EventStats, error) {
	query := `
		SELECT
			toStartOfHour(created_at) as hour,
			event_type,
			count() as cnt,
			uniq(user_id) as unique_users
		FROM events
		WHERE created_at BETWEEN ? AND ?
		GROUP BY hour, event_type
		ORDER BY hour DESC
	`

	rows, err := r.conn.Query(ctx, query, start, end)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	var stats []EventStats

	for rows.Next() {
		var (
			s                EventStats
			cnt, uniqueUsers uint64
		)

		if err := rows.Scan(&s.Hour, &s.EventType, &cnt, &uniqueUsers); err != nil {
			return nil, err
		}

		s.Count = safeUint64ToInt64(cnt)
		s.UniqueUsers = safeUint64ToInt64(uniqueUsers)
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

func (r *ClickHouseRepo) GetStorageStats(ctx context.Context) *StorageStats {
	var stats StorageStats

	query := `
		SELECT
			sum(bytes) as total_bytes,
			sum(rows) as total_rows,
			sum(bytes) / sum(data_uncompressed_bytes) as compression_ratio
		FROM system.parts
		WHERE database = currentDatabase()
		AND table = 'events'
		AND active = 1
	`

	row := r.conn.QueryRow(ctx, query)

	var totalBytes, totalRows uint64

	var compressionRatio float64

	err := row.Scan(&totalBytes, &totalRows, &compressionRatio)
	if err != nil {
		return &stats
	}

	stats.TotalSize = safeUint64ToInt64(totalBytes)
	stats.RowCount = safeUint64ToInt64(totalRows)
	stats.CompressionPct = (1 - compressionRatio) * 100
	stats.IndexSize = 0

	return &stats
}

func (r *ClickHouseRepo) Cleanup(ctx context.Context) error {
	return r.conn.Exec(ctx, "TRUNCATE TABLE events")
}

func (r *ClickHouseRepo) Close() error {
	return r.conn.Close()
}

func safeInt64ToUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}

	return uint64(v)
}

func safeUint64ToInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(v)
}
