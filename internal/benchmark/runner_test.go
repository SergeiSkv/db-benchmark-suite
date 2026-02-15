package benchmark

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skoredin/db-benchmark-suite/internal/generator"
	"github.com/skoredin/db-benchmark-suite/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository implements Repository for testing.
type mockRepository struct {
	insertBatchFunc   func(ctx context.Context, events []generator.Event) error
	getEventStatsFunc func(ctx context.Context, start, end time.Time) ([]repository.EventStats, error)
	callCount         int64
}

func (m *mockRepository) InitSchema(context.Context) error { return nil }

func (m *mockRepository) InsertBatch(ctx context.Context, events []generator.Event) error {
	if m.insertBatchFunc != nil {
		return m.insertBatchFunc(ctx, events)
	}

	return nil
}

func (m *mockRepository) GetEventStats(ctx context.Context, start, end time.Time) ([]repository.EventStats, error) {
	atomic.AddInt64(&m.callCount, 1)

	if m.getEventStatsFunc != nil {
		return m.getEventStatsFunc(ctx, start, end)
	}

	return nil, nil
}

func (m *mockRepository) GetStorageStats(context.Context) *repository.StorageStats {
	return nil
}

func (m *mockRepository) Cleanup(context.Context) error { return nil }
func (m *mockRepository) Close() error                  { return nil }

func TestRunInsert(t *testing.T) {
	mock := &mockRepository{}

	runner := &Runner{
		EventCount: 100,
		BatchSize:  10,
		Workers:    2,
	}

	result := runner.RunInsert(context.Background(), mock)

	require.NotNil(t, result)
	assert.Equal(t, 100, result.TotalEvents)
	assert.Equal(t, int64(0), result.ErrorCount)
	assert.Greater(t, result.Throughput, 0.0)
	assert.Greater(t, result.Duration, time.Duration(0))
	// Throughput should be based on actually inserted events
	expectedThroughput := float64(100) / result.Duration.Seconds()
	assert.InDelta(t, expectedThroughput, result.Throughput, 1.0)
}

func TestRunInsertWithErrors(t *testing.T) {
	var callNum int64

	mock := &mockRepository{
		insertBatchFunc: func(_ context.Context, _ []generator.Event) error {
			n := atomic.AddInt64(&callNum, 1)
			if n%2 == 0 {
				return fmt.Errorf("simulated error")
			}

			return nil
		},
	}

	runner := &Runner{
		EventCount: 100,
		BatchSize:  10,
		Workers:    1,
	}

	result := runner.RunInsert(context.Background(), mock)

	require.NotNil(t, result)
	assert.Greater(t, result.ErrorCount, int64(0))
	// Throughput should reflect only actually inserted events, not all requested
	// With half the batches failing, inserted should be ~50
	inserted := int64(result.Throughput * result.Duration.Seconds())
	assert.Less(t, inserted, int64(result.TotalEvents))
}

func TestRunQueries(t *testing.T) {
	mock := &mockRepository{}

	runner := &Runner{
		QueryIterations:  5,
		WarmupIterations: 1,
	}

	results := runner.RunQueries(context.Background(), mock)

	require.Len(t, results, 4)

	for _, name := range []string{"1_hour", "1_day", "1_week", "1_month"} {
		qr, ok := results[name]
		require.True(t, ok, "missing query result for %s", name)
		assert.Equal(t, name, qr.QueryName)
		assert.Equal(t, 5, qr.Iterations)
		assert.Equal(t, int64(0), qr.ErrorCount)
	}
}

func TestRunQueryWarmup(t *testing.T) {
	mock := &mockRepository{}

	runner := &Runner{
		QueryIterations:  10,
		WarmupIterations: 3,
	}

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()

	_ = runner.runQuery(context.Background(), mock, "test", start, end)

	// Total calls = warmup (3) + iterations (10)
	assert.Equal(t, int64(13), atomic.LoadInt64(&mock.callCount))
}
