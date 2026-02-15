package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageStats_TotalSizeGB(t *testing.T) {
	tests := []struct {
		name      string
		totalSize int64
		expected  float64
	}{
		{
			name:      "1GB",
			totalSize: 1024 * 1024 * 1024,
			expected:  1.0,
		},
		{
			name:      "2.5GB",
			totalSize: int64(2.5 * 1024 * 1024 * 1024),
			expected:  2.5,
		},
		{
			name:      "500MB",
			totalSize: 500 * 1024 * 1024,
			expected:  500.0 / 1024.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &StorageStats{
				TotalSize: tt.totalSize,
			}

			assert.InDelta(t, tt.expected, stats.TotalSizeGB(), 0.001)
		})
	}
}

func TestStorageStats_IndexSizeGB(t *testing.T) {
	stats := &StorageStats{
		IndexSize: 1024 * 1024 * 1024,
	}

	assert.Equal(t, 1.0, stats.IndexSizeGB())
}

func BenchmarkStorageStats_TotalSizeGB(b *testing.B) {
	stats := &StorageStats{
		TotalSize: 1024 * 1024 * 1024 * 100,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = stats.TotalSizeGB()
	}
}
