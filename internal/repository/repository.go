package repository

import "time"

// EventStats represents aggregated event statistics
type EventStats struct {
	Hour        time.Time
	EventType   string
	Count       int64
	UniqueUsers int64
}

// StorageStats represents storage metrics
type StorageStats struct {
	TotalSize      int64   `json:"total_size"`
	IndexSize      int64   `json:"index_size"`
	CompressionPct float64 `json:"compression_pct"`
	RowCount       int64   `json:"row_count"`
}

// TotalSizeGB returns total size in gigabytes.
func (s *StorageStats) TotalSizeGB() float64 {
	return float64(s.TotalSize) / (1024 * 1024 * 1024)
}

// IndexSizeGB returns index size in gigabytes.
func (s *StorageStats) IndexSizeGB() float64 {
	return float64(s.IndexSize) / (1024 * 1024 * 1024)
}
