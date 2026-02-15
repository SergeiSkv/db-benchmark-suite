package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/skoredin/db-benchmark-suite/internal/benchmark"
	"github.com/skoredin/db-benchmark-suite/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleResults() map[string]*benchmark.Results {
	return map[string]*benchmark.Results{
		"postgres": {
			Database:  "postgres",
			Timestamp: time.Now(),
			Insert: &benchmark.InsertResult{
				TotalEvents: 1000,
				Duration:    5 * time.Second,
				Throughput:  200.0,
				ErrorCount:  0,
				BatchSize:   100,
				WorkerCount: 4,
			},
			Queries: map[string]*benchmark.QueryResult{
				"1_hour": {
					QueryName:   "1_hour",
					Iterations:  10,
					AvgDuration: 50 * time.Millisecond,
					MinDuration: 30 * time.Millisecond,
					MaxDuration: 80 * time.Millisecond,
					P50Duration: 45 * time.Millisecond,
					P95Duration: 75 * time.Millisecond,
					P99Duration: 79 * time.Millisecond,
					ErrorCount:  0,
				},
			},
			Storage: &repository.StorageStats{
				TotalSize:      1024 * 1024 * 1024,
				IndexSize:      256 * 1024 * 1024,
				CompressionPct: 42.5,
				RowCount:       1000,
			},
		},
	}
}

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer

	rep := New("table", &buf)
	rep.PrintResults(sampleResults())

	output := buf.String()

	assert.Contains(t, output, "INSERT BENCHMARK")
	assert.Contains(t, output, "STORAGE STATISTICS")
	assert.Contains(t, output, "postgres")
	assert.Contains(t, output, "200/sec")
	assert.Contains(t, output, "1.00 GB")
	assert.Contains(t, output, "256.00 MB")
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer

	rep := New("json", &buf)
	rep.PrintResults(sampleResults())

	output := buf.Bytes()

	var parsed map[string]any

	err := json.Unmarshal(output, &parsed)
	require.NoError(t, err, "output should be valid JSON")

	pg, ok := parsed["postgres"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "postgres", pg["database"])
}

func TestPrintMarkdown(t *testing.T) {
	var buf bytes.Buffer

	rep := New("markdown", &buf)
	rep.PrintResults(sampleResults())

	output := buf.String()

	assert.Contains(t, output, "## Insert Performance")
	assert.Contains(t, output, "## Storage Statistics")
	assert.Contains(t, output, "postgres")
	// Markdown tables use pipes
	assert.True(t, strings.Contains(output, "| postgres"))
}

func TestPrintHeader(t *testing.T) {
	var buf bytes.Buffer

	rep := New("table", &buf)
	rep.PrintHeader()

	output := buf.String()
	assert.Contains(t, output, "Database Benchmark Suite")
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{50 * 1024 * 1024, "50.00 MB"},
		{256 * 1024 * 1024, "256.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{3 * 1024 * 1024 * 1024, "3.00 GB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatBytes(tt.bytes), "formatBytes(%d)", tt.bytes)
	}
}
