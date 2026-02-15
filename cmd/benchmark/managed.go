package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skoredin/db-benchmark-suite/internal/benchmark"
	"github.com/skoredin/db-benchmark-suite/internal/config"
	"github.com/skoredin/db-benchmark-suite/internal/orchestrator"
	"github.com/skoredin/db-benchmark-suite/internal/reporter"
)

const (
	cGreen  = "\033[0;32m"
	cBlue   = "\033[0;34m"
	cYellow = "\033[1;33m"
	cRed    = "\033[0;31m"
	cReset  = "\033[0m"
)

func colorLogf(color, format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, color+format+cReset+"\n", args...)
}

// runManaged starts each database container sequentially, runs the benchmark,
// stops the container, then prints a combined summary at the end.
func runManaged() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runner := newRunner()
	databases := getDatabases(*dbType)

	printManagedHeader(runner, databases)

	allResults := runManagedBenchmarks(ctx, cfg, runner, databases)

	printManagedResults(ctx, allResults)
}

func runManagedBenchmarks(ctx context.Context, cfg *config.Config, runner *benchmark.Runner, databases []string) map[string]*benchmark.Results {
	allResults := make(map[string]*benchmark.Results)
	for _, dbName := range databases {
		allResults[dbName] = runManagedDB(ctx, cfg, runner, dbName)
	}

	return allResults
}

func printManagedResults(ctx context.Context, allResults map[string]*benchmark.Results) {
	rep := reporter.New(*outputFormat, os.Stderr)
	rep.PrintHeader()
	rep.PrintResults(allResults)

	if *cleanupFlag {
		if err := orchestrator.Cleanup(ctx); err != nil {
			log.Printf("Failed to cleanup orchestrator: %v", err)
		}
	}

	colorLogf(cGreen, "All benchmarks complete!")
}

func printManagedHeader(runner *benchmark.Runner, databases []string) {
	colorLogf(cBlue, "Managed mode: testing %d database(s) sequentially", len(databases))

	if *preloadCount > 0 {
		colorLogf(cYellow, "Preload: %d | Events: %d | Batch: %d | Workers: %d", runner.PreloadCount, runner.EventCount, runner.BatchSize, runner.Workers)
	} else {
		colorLogf(cYellow, "Events: %d | Batch: %d | Workers: %d", runner.EventCount, runner.BatchSize, runner.Workers)
	}

	_, _ = fmt.Fprintln(os.Stderr)
}

func runManagedDB(ctx context.Context, cfg *config.Config, runner *benchmark.Runner, dbName string) *benchmark.Results {
	svc, ok := orchestrator.ServiceByName(dbName)
	if !ok {
		colorLogf(cRed, "Unknown database: %s, skipping", dbName)
		return &benchmark.Results{Database: dbName, Error: fmt.Errorf("unknown database: %s", dbName)}
	}

	colorLogf(cBlue, "================================================")
	colorLogf(cBlue, "  %s", dbName)
	colorLogf(cBlue, "================================================")

	result := runManagedBenchmark(ctx, cfg, runner, svc)

	if result.Error != nil {
		colorLogf(cRed, "✗ %s failed: %v", dbName, result.Error)
	} else {
		colorLogf(cGreen, "✓ %s benchmark complete", dbName)
	}

	_, _ = fmt.Fprintln(os.Stderr)

	return result
}

func runManagedBenchmark(ctx context.Context, cfg *config.Config, runner *benchmark.Runner, svc orchestrator.DBService) *benchmark.Results {
	if err := orchestrator.StartService(ctx, svc.Service); err != nil {
		return &benchmark.Results{Database: svc.Name, Error: err}
	}

	if err := orchestrator.WaitReady(ctx, svc); err != nil {
		if err := orchestrator.StopService(ctx, svc.Service); err != nil {
			log.Printf("Failed to stop orchestrator: %v", err)
		}

		return &benchmark.Results{Database: svc.Name, Error: err}
	}

	colorLogf(cGreen, "Running benchmark for %s...", svc.Name)
	result := runBenchmark(ctx, cfg, runner, svc.Name)
	result.Database = svc.Name
	result.Timestamp = time.Now()

	if err := orchestrator.StopService(ctx, svc.Service); err != nil {
		log.Printf("Failed to stop orchestrator: %v", err)
	}

	return result
}
