package generator

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_Generate(t *testing.T) {
	totalEvents := 100
	batchSize := 10
	gen := New(totalEvents, batchSize)

	eventCount := 0
	batchCount := 0

	for batch := range gen.Generate() {
		batchCount++
		eventCount += len(batch)

		// Check batch size
		if eventCount < totalEvents {
			assert.LessOrEqual(t, len(batch), batchSize)
		}

		// Validate each event
		for _, event := range batch {
			assert.NotEmpty(t, event.ID)
			assert.NotZero(t, event.UserID)
			assert.NotEmpty(t, event.EventType)
			assert.NotEmpty(t, event.Payload)
			assert.False(t, event.CreatedAt.IsZero())
		}
	}

	assert.Equal(t, totalEvents, eventCount, "Should generate exact number of events")
	assert.Equal(t, totalEvents/batchSize, batchCount, "Should generate correct number of batches")
}

func TestGenerator_EventTypes(t *testing.T) {
	gen := New(1000, 100)
	seenTypes := make(map[string]bool)

	for batch := range gen.Generate() {
		for _, event := range batch {
			seenTypes[event.EventType] = true
		}
	}

	// Should have seen multiple event types
	assert.Greater(t, len(seenTypes), 3, "Should generate diverse event types")
}

func TestGenerator_UniqueEventIDs(t *testing.T) {
	gen := New(100, 10)
	seenIDs := make(map[string]bool)

	for batch := range gen.Generate() {
		for _, event := range batch {
			assert.False(t, seenIDs[event.ID], "Event IDs should be unique")
			seenIDs[event.ID] = true
		}
	}

	assert.Equal(t, 100, len(seenIDs), "Should have 100 unique event IDs")
}

func TestGenerator_PayloadGeneration(t *testing.T) {
	gen := New(10, 10)

	for batch := range gen.Generate() {
		for _, event := range batch {
			payload := event.Payload
			assert.NotEmpty(t, payload, "Payload should not be empty")
			assert.Greater(t, len(payload), 10, "Payload should have reasonable length")
		}
	}
}

func TestGenerator_TimeDistribution(t *testing.T) {
	gen := New(100, 10)

	for batch := range gen.Generate() {
		for _, event := range batch {
			// Events should be within last 90 days
			daysDiff := time.Since(event.CreatedAt).Hours() / 24
			assert.LessOrEqual(t, daysDiff, 90.0, "Events should be within 90 days")
			assert.GreaterOrEqual(t, daysDiff, 0.0, "Events should not be in future")
		}
	}
}

func TestGenerator_EmptyGeneration(t *testing.T) {
	gen := New(0, 10)

	batchCount := 0
	for range gen.Generate() {
		batchCount++
	}

	assert.Equal(t, 0, batchCount, "Should not generate batches for 0 events")
}

func TestRandomString(t *testing.T) {
	gen := New(1, 1)
	tests := []int{0, 10, 32, 64}

	for _, length := range tests {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			result := gen.randomString(length)
			assert.Equal(t, length, len(result))

			// Check all characters are valid
			for _, ch := range result {
				assert.True(t,
					(ch >= 'a' && ch <= 'z') ||
						(ch >= 'A' && ch <= 'Z') ||
						(ch >= '0' && ch <= '9'),
					"Should only contain alphanumeric characters",
				)
			}
		})
	}
}

func BenchmarkGenerator_Generate(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gen := New(1000, 100)
		for batch := range gen.Generate() {
			_ = batch
		}
	}
}

func BenchmarkGenerator_GenerateEvent(b *testing.B) {
	gen := New(1000000, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = gen.generateEvent()
	}
}

func BenchmarkGenerator_GeneratePayload(b *testing.B) {
	gen := New(1000000, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = gen.generatePayload()
	}
}

func TestGenerator_UserIDDistribution(t *testing.T) {
	gen := New(10000, 1000)
	userIDs := make(map[int64]int)

	for batch := range gen.Generate() {
		for _, event := range batch {
			userIDs[event.UserID]++
		}
	}

	// Should have good distribution of users
	assert.Greater(t, len(userIDs), 100, "Should have diverse user IDs")
	assert.LessOrEqual(t, len(userIDs), 1000000, "User IDs should be within range")
}

// Fuzz test for generator
func FuzzGenerator(f *testing.F) {
	f.Add(100, 10)
	f.Add(1000, 100)
	f.Add(10000, 1000)

	f.Fuzz(func(t *testing.T, totalEvents, batchSize int) {
		if totalEvents < 0 || totalEvents > 100000 {
			t.Skip()
		}

		if batchSize < 1 || batchSize > 10000 {
			t.Skip()
		}

		gen := New(totalEvents, batchSize)
		eventCount := 0

		for batch := range gen.Generate() {
			require.LessOrEqual(t, len(batch), batchSize)
			eventCount += len(batch)

			for _, event := range batch {
				require.NotEmpty(t, event.ID)
				require.NotEmpty(t, event.EventType)
			}
		}

		require.Equal(t, totalEvents, eventCount)
	})
}
