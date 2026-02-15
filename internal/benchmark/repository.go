package benchmark

import (
	"context"
	"time"

	"github.com/skoredin/db-benchmark-suite/internal/generator"
	"github.com/skoredin/db-benchmark-suite/internal/repository"
)

// Repository defines common interface for all database implementations.
type Repository interface {
	InitSchema(ctx context.Context) error
	InsertBatch(ctx context.Context, events []generator.Event) error
	GetEventStats(ctx context.Context, start, end time.Time) ([]repository.EventStats, error)
	GetStorageStats(ctx context.Context) *repository.StorageStats
	Cleanup(ctx context.Context) error
	Close() error
}
