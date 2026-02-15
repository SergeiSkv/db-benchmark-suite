package benchmark

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/skoredin/db-benchmark-suite/internal/generator"
)

// Runner executes insert and query benchmarks.
type Runner struct {
	EventCount       int
	BatchSize        int
	Workers          int
	QueryIterations  int
	WarmupIterations int
	PreloadCount     int
}

// Preload inserts seed data without measuring performance.
func (r *Runner) Preload(ctx context.Context, repo Repository) error {
	if r.PreloadCount <= 0 {
		return nil
	}

	inserted, errors := r.parallelInsert(ctx, repo, r.PreloadCount, int64(r.BatchSize)*50)
	log.Printf("Preload complete: %d events inserted, %d errors", inserted, errors)

	if errors > 0 && inserted == 0 {
		return fmt.Errorf("preload failed: all %d batches errored", errors)
	}

	return nil
}

// RunInsert benchmarks batch inserts into the given repository.
func (r *Runner) RunInsert(ctx context.Context, repo Repository) *InsertResult {
	start := time.Now()
	inserted, errors := r.parallelInsert(ctx, repo, r.EventCount, int64(r.BatchSize)*10)
	duration := time.Since(start)

	return &InsertResult{
		TotalEvents: r.EventCount,
		Duration:    duration,
		Throughput:  float64(inserted) / duration.Seconds(),
		ErrorCount:  errors,
		BatchSize:   r.BatchSize,
		WorkerCount: r.Workers,
	}
}

func (r *Runner) parallelInsert(ctx context.Context, repo Repository, count int, logInterval int64) (inserted, errors int64) {
	gen := generator.New(count, r.BatchSize)

	var totalInserted, totalErrors int64

	batches := make(chan []generator.Event, r.Workers*2)

	var wg sync.WaitGroup

	for i := 0; i < r.Workers; i++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			r.consumeBatches(ctx, repo, batches, &totalInserted, &totalErrors, count, logInterval, workerID)
		}(i)
	}

	go pumpBatches(gen.Generate(), batches)

	wg.Wait()

	return atomic.LoadInt64(&totalInserted), atomic.LoadInt64(&totalErrors)
}

func (r *Runner) consumeBatches(
	ctx context.Context, repo Repository, batches <-chan []generator.Event,
	totalInserted, totalErrors *int64, total int, logInterval int64, workerID int,
) {
	for batch := range batches {
		if err := repo.InsertBatch(ctx, batch); err != nil {
			if workerID >= 0 {
				log.Printf("Worker %d insert error: %v", workerID, err)
			}

			atomic.AddInt64(totalErrors, 1)

			continue
		}

		prev := atomic.LoadInt64(totalInserted)
		inserted := atomic.AddInt64(totalInserted, int64(len(batch)))

		if logInterval > 0 && prev/logInterval != inserted/logInterval {
			log.Printf("Insert progress: %d / %d events", inserted, total)
		}
	}
}

func pumpBatches(src <-chan []generator.Event, dst chan<- []generator.Event) {
	for batch := range src {
		dst <- batch
	}

	close(dst)
}

// RunQueries benchmarks all query scenarios against the given repository.
func (r *Runner) RunQueries(ctx context.Context, repo Repository) map[string]*QueryResult {
	results := make(map[string]*QueryResult)
	now := time.Now()

	scenarios := []struct {
		name  string
		start time.Time
	}{
		{"1_hour", now.Add(-1 * time.Hour)},
		{"1_day", now.Add(-24 * time.Hour)},
		{"1_week", now.Add(-7 * 24 * time.Hour)},
		{"1_month", now.Add(-30 * 24 * time.Hour)},
	}

	for _, s := range scenarios {
		results[s.name] = r.runQuery(ctx, repo, s.name, s.start, now)
	}

	return results
}

func (r *Runner) runQuery(ctx context.Context, repo Repository, name string, start, end time.Time) *QueryResult {
	for i := 0; i < r.WarmupIterations; i++ {
		_, _ = repo.GetEventStats(ctx, start, end)
	}

	durations, errors := r.measureQuery(ctx, repo, start, end)

	if len(durations) == 0 {
		return &QueryResult{QueryName: name, ErrorCount: errors}
	}

	return &QueryResult{
		QueryName:   name,
		Iterations:  len(durations),
		AvgDuration: AvgDuration(durations),
		MinDuration: MinDuration(durations),
		MaxDuration: MaxDuration(durations),
		P50Duration: Percentile(durations, 0.50),
		P95Duration: Percentile(durations, 0.95),
		P99Duration: Percentile(durations, 0.99),
		ErrorCount:  errors,
		DateRange:   fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")),
	}
}

func (r *Runner) measureQuery(ctx context.Context, repo Repository, start, end time.Time) (durations []time.Duration, errors int64) {
	for i := 0; i < r.QueryIterations; i++ {
		queryStart := time.Now()
		_, err := repo.GetEventStats(ctx, start, end)
		d := time.Since(queryStart)

		if err != nil {
			errors++

			log.Printf("Query error: %v", err)

			continue
		}

		durations = append(durations, d)
	}

	return
}
