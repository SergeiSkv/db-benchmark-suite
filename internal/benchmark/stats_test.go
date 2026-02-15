package benchmark

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAvgDuration(t *testing.T) {
	t.Run("multiple elements", func(t *testing.T) {
		durations := []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
			300 * time.Millisecond,
		}
		assert.Equal(t, 200*time.Millisecond, AvgDuration(durations))
	})

	t.Run("single element", func(t *testing.T) {
		durations := []time.Duration{42 * time.Millisecond}
		assert.Equal(t, 42*time.Millisecond, AvgDuration(durations))
	})
}

func TestMinDuration(t *testing.T) {
	durations := []time.Duration{
		300 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}
	assert.Equal(t, 100*time.Millisecond, MinDuration(durations))
}

func TestMaxDuration(t *testing.T) {
	durations := []time.Duration{
		300 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}
	assert.Equal(t, 300*time.Millisecond, MaxDuration(durations))
}

func TestPercentile(t *testing.T) {
	// 100 values: 1ms, 2ms, ..., 100ms
	durations := make([]time.Duration, 100)
	for i := range durations {
		durations[i] = time.Duration(i+1) * time.Millisecond
	}

	t.Run("P50", func(t *testing.T) {
		// index = int(100 * 0.50) = 50 → value at [50] = 51ms
		assert.Equal(t, 51*time.Millisecond, Percentile(durations, 0.50))
	})

	t.Run("P95", func(t *testing.T) {
		// index = int(100 * 0.95) = 95 → value at [95] = 96ms
		assert.Equal(t, 96*time.Millisecond, Percentile(durations, 0.95))
	})

	t.Run("P99", func(t *testing.T) {
		// index = int(100 * 0.99) = 99 → value at [99] = 100ms
		assert.Equal(t, 100*time.Millisecond, Percentile(durations, 0.99))
	})
}

func TestPercentileSingleElement(t *testing.T) {
	durations := []time.Duration{42 * time.Millisecond}
	assert.Equal(t, 42*time.Millisecond, Percentile(durations, 0.50))
	assert.Equal(t, 42*time.Millisecond, Percentile(durations, 0.99))
}

func TestPercentileDoesNotMutateInput(t *testing.T) {
	durations := []time.Duration{
		300 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}
	_ = Percentile(durations, 0.50)
	// First element should still be 300ms (unsorted)
	assert.Equal(t, 300*time.Millisecond, durations[0])
}

func TestInsertResult_Throughput(t *testing.T) {
	result := &InsertResult{
		TotalEvents: 1000000,
		Duration:    10 * time.Second,
		Throughput:  100000.0,
	}

	assert.Equal(t, 1000000, result.TotalEvents)
	assert.Equal(t, 10*time.Second, result.Duration)
	assert.Equal(t, 100000.0, result.Throughput)
}

func TestQueryResult_AllFields(t *testing.T) {
	result := &QueryResult{
		QueryName:   "test_query",
		Iterations:  100,
		AvgDuration: 100 * time.Millisecond,
		MinDuration: 50 * time.Millisecond,
		MaxDuration: 200 * time.Millisecond,
		P50Duration: 95 * time.Millisecond,
		P95Duration: 180 * time.Millisecond,
		P99Duration: 195 * time.Millisecond,
		ErrorCount:  5,
		DateRange:   "2024-01-01 to 2024-01-31",
	}

	assert.Equal(t, "test_query", result.QueryName)
	assert.Equal(t, 100, result.Iterations)
	assert.Equal(t, int64(5), result.ErrorCount)
	assert.Equal(t, "2024-01-01 to 2024-01-31", result.DateRange)
}

func TestResults_ErrorHandling(t *testing.T) {
	result := &Results{
		Database:  "test_db",
		Timestamp: time.Now(),
		Error:     assert.AnError,
	}

	assert.NotNil(t, result.Error)
	assert.Equal(t, "test_db", result.Database)
}
