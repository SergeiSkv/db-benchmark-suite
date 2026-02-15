package benchmark

import (
	"encoding/json"
	"time"

	"github.com/skoredin/db-benchmark-suite/internal/repository"
)

// Results contains all benchmark results for a database
type Results struct {
	Database  string                   `json:"database"`
	Timestamp time.Time                `json:"timestamp"`
	Insert    *InsertResult            `json:"insert,omitempty"`
	Queries   map[string]*QueryResult  `json:"queries,omitempty"`
	Storage   *repository.StorageStats `json:"storage,omitempty"`
	Error     error                    `json:"-"`
	ErrorText string                   `json:"error,omitempty"`
}

// MarshalJSON implements json.Marshaler to serialize the Error field as a string.
func (r *Results) MarshalJSON() ([]byte, error) {
	type Alias Results

	a := (*Alias)(r)
	if a.Error != nil && a.ErrorText == "" {
		a.ErrorText = a.Error.Error()
	}

	return json.Marshal(a)
}

// InsertResult contains insert benchmark metrics
type InsertResult struct {
	TotalEvents int           `json:"total_events"`
	Duration    time.Duration `json:"duration"`
	Throughput  float64       `json:"throughput"`
	ErrorCount  int64         `json:"error_count"`
	BatchSize   int           `json:"batch_size"`
	WorkerCount int           `json:"worker_count"`
}

// QueryResult contains query benchmark metrics
type QueryResult struct {
	QueryName   string        `json:"query_name"`
	Iterations  int           `json:"iterations"`
	AvgDuration time.Duration `json:"avg_duration"`
	MinDuration time.Duration `json:"min_duration"`
	MaxDuration time.Duration `json:"max_duration"`
	P50Duration time.Duration `json:"p50_duration"`
	P95Duration time.Duration `json:"p95_duration"`
	P99Duration time.Duration `json:"p99_duration"`
	ErrorCount  int64         `json:"error_count"`
	DateRange   string        `json:"date_range"`
}
