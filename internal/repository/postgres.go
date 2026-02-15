package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/skoredin/db-benchmark-suite/internal/config"
	"github.com/skoredin/db-benchmark-suite/internal/generator"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(ctx context.Context, cfg *config.PostgresConfig) (*PostgresRepo, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresRepo{db: db}, nil
}

func (r *PostgresRepo) InitSchema(ctx context.Context) error {
	schema := `
		DROP TABLE IF EXISTS events CASCADE;

		CREATE TABLE events (
			id BIGSERIAL,
			event_id VARCHAR(255) NOT NULL,
			user_id BIGINT NOT NULL,
			event_type VARCHAR(50) NOT NULL,
			payload TEXT,
			created_at TIMESTAMP NOT NULL
		) PARTITION BY RANGE (created_at);
	`

	if _, err := r.db.ExecContext(ctx, schema); err != nil {
		return err
	}

	if err := r.createPartitions(ctx); err != nil {
		return err
	}

	// Create indexes on the partitioned table
	indexes := `
		CREATE INDEX idx_events_created_at ON events USING brin(created_at) WITH (pages_per_range = 32);
		CREATE INDEX idx_events_type_time ON events(event_type, created_at);
		CREATE INDEX idx_events_user_id ON events(user_id);
		CREATE UNIQUE INDEX idx_events_event_id ON events(event_id, created_at);
	`

	_, err := r.db.ExecContext(ctx, indexes)

	return err
}

func (r *PostgresRepo) createPartitions(ctx context.Context) error {
	now := time.Now()
	for i := -4; i <= 0; i++ {
		start := time.Date(now.Year(), now.Month()+time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)

		partName := "events_" + start.Format("200601")
		if err := r.createPartition(ctx, partName, start, end); err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresRepo) createPartition(ctx context.Context, name string, start, end time.Time) error {
	query := "CREATE TABLE IF NOT EXISTS " + pq.QuoteIdentifier(name) +
		" PARTITION OF events FOR VALUES FROM (" + pq.QuoteLiteral(start.Format("2006-01-02")) +
		") TO (" + pq.QuoteLiteral(end.Format("2006-01-02")) + ")"

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create partition %s: %w", name, err)
	}

	return nil
}

func (r *PostgresRepo) InsertBatch(ctx context.Context, events []generator.Event) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO events (event_id, user_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id, created_at) DO NOTHING
	`)
	if err != nil {
		return err
	}

	defer func() { _ = stmt.Close() }()

	for _, event := range events {
		_, err := stmt.ExecContext(ctx,
			event.ID,
			event.UserID,
			event.EventType,
			event.Payload,
			event.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepo) GetEventStats(ctx context.Context, start, end time.Time) ([]EventStats, error) {
	query := `
		SELECT 
			date_trunc('hour', created_at) as hour,
			event_type,
			COUNT(*) as count,
			COUNT(DISTINCT user_id) as unique_users
		FROM events
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY hour, event_type
		ORDER BY hour DESC
	`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	var stats []EventStats

	for rows.Next() {
		var s EventStats
		if err := rows.Scan(&s.Hour, &s.EventType, &s.Count, &s.UniqueUsers); err != nil {
			return nil, err
		}

		stats = append(stats, s)
	}

	return stats, rows.Err()
}

func (r *PostgresRepo) GetStorageStats(ctx context.Context) *StorageStats {
	var stats StorageStats

	// Sum sizes across all partitions (pg_total_relation_size on a
	// partitioned parent may return 0 in some PostgreSQL versions).
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(pg_total_relation_size(inhrelid::regclass)), 0),
			COALESCE(SUM(pg_indexes_size(inhrelid::regclass)), 0)
		FROM pg_inherits
		WHERE inhparent = 'events'::regclass
	`).Scan(&stats.TotalSize, &stats.IndexSize)
	if err != nil {
		return &StorageStats{}
	}

	// Row count separately to avoid mixing aggregate with system functions
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&stats.RowCount)

	return &stats
}

func (r *PostgresRepo) Cleanup(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "TRUNCATE TABLE events")
	return err
}

func (r *PostgresRepo) Close() error {
	return r.db.Close()
}
