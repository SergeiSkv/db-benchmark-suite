package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/skoredin/db-benchmark-suite/internal/benchmark"
	"github.com/skoredin/db-benchmark-suite/internal/config"
	"github.com/skoredin/db-benchmark-suite/internal/reporter"
	"github.com/skoredin/db-benchmark-suite/internal/repository"
)

var (
	dbType          = flag.String("db", "all", "Database type: postgres, mongodb, cassandra, clickhouse, all")
	eventCount      = flag.Int("events", 1000000, "Number of events to generate")
	batchSize       = flag.Int("batch", 10000, "Batch size for inserts")
	workers         = flag.Int("workers", runtime.NumCPU(), "Number of concurrent workers")
	queryIterations = flag.Int("queries", 100, "Number of query iterations")
	outputFormat    = flag.String("output", "table", "Output format: table, json, markdown")
	skipInsert      = flag.Bool("skip-insert", false, "Skip insert benchmark")
	skipQuery       = flag.Bool("skip-query", false, "Skip query benchmark")
	preloadCount    = flag.Int("preload", 0, "Pre-load database with N events before benchmarking (0 = skip)")
	cleanupFlag     = flag.Bool("cleanup", false, "Cleanup data after benchmark")
	managed         = flag.Bool("managed", false, "Manage Docker containers automatically (start/stop per database)")
)

func main() {
	flag.Parse()
	validateFlags()

	if *managed {
		runManaged()
		return
	}

	runDirect()
}

func validateFlags() {
	if *eventCount <= 0 {
		log.Fatal("--events must be positive")
	}

	if *batchSize <= 0 {
		log.Fatal("--batch must be positive")
	}

	if *workers <= 0 {
		log.Fatal("--workers must be positive")
	}

	if *queryIterations <= 0 {
		log.Fatal("--queries must be positive")
	}
}

func runDirect() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	rep := reporter.New(*outputFormat, os.Stdout)
	rep.PrintHeader()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databases := getDatabases(*dbType)
	runner := newRunner()

	results := runAllBenchmarks(ctx, cfg, runner, databases)
	rep.PrintResults(results)

	if *cleanupFlag {
		cleanupDatabases(ctx, cfg, databases)
	}
}

func runAllBenchmarks(ctx context.Context, cfg *config.Config, runner *benchmark.Runner, databases []string) map[string]*benchmark.Results {
	results := make(map[string]*benchmark.Results)

	var mu sync.Mutex

	var wg sync.WaitGroup

	for _, db := range databases {
		wg.Add(1)

		go func(dbName string) {
			defer wg.Done()

			log.Printf("Starting benchmark for %s...", dbName)

			result := runBenchmark(ctx, cfg, runner, dbName)

			mu.Lock()

			results[dbName] = result

			mu.Unlock()

			log.Printf("Completed benchmark for %s", dbName)
		}(db)
	}

	wg.Wait()

	return results
}

func newRunner() *benchmark.Runner {
	batch := *batchSize
	maxEvents := *eventCount

	if *preloadCount > maxEvents {
		maxEvents = *preloadCount
	}

	if batch > maxEvents {
		batch = maxEvents
	}

	totalBatches := (maxEvents + batch - 1) / batch
	w := *workers

	if w > totalBatches {
		w = totalBatches
	}

	return &benchmark.Runner{
		EventCount:       *eventCount,
		BatchSize:        batch,
		Workers:          w,
		QueryIterations:  *queryIterations,
		WarmupIterations: 5,
		PreloadCount:     *preloadCount,
	}
}

func getDatabases(dbType string) []string {
	if dbType == "all" {
		return []string{"postgres", "mongodb", "clickhouse", "cassandra"}
	}

	return []string{dbType}
}

func runBenchmark(ctx context.Context, cfg *config.Config, runner *benchmark.Runner, dbName string) *benchmark.Results {
	repo, err := newRepo(ctx, dbName, cfg)
	if err != nil {
		log.Printf("Failed to initialize %s: %v", dbName, err)
		return &benchmark.Results{Error: err}
	}

	defer func() {
		if err := repo.Close(); err != nil {
			log.Printf("Failed to close %s: %v", dbName, err)
		}
	}()

	if err := repo.InitSchema(ctx); err != nil {
		log.Printf("Failed to initialize %s schema: %v", dbName, err)
		return &benchmark.Results{Error: err}
	}

	if err := preloadIfNeeded(ctx, runner, repo, dbName); err != nil {
		return &benchmark.Results{Error: err}
	}

	return executeBenchmark(ctx, runner, repo, dbName)
}

func preloadIfNeeded(ctx context.Context, runner *benchmark.Runner, repo benchmark.Repository, dbName string) error {
	if runner.PreloadCount <= 0 {
		return nil
	}

	log.Printf("Pre-loading %s with %d events...", dbName, runner.PreloadCount)

	if err := runner.Preload(ctx, repo); err != nil {
		log.Printf("Failed to preload %s: %v", dbName, err)
		return err
	}

	return nil
}

func executeBenchmark(ctx context.Context, runner *benchmark.Runner, repo benchmark.Repository, dbName string) *benchmark.Results {
	res := &benchmark.Results{Database: dbName, Timestamp: time.Now()}

	if !*skipInsert {
		log.Printf("Benchmarking inserts for %s (%d events)...", dbName, runner.EventCount)
		res.Insert = runner.RunInsert(ctx, repo)
		log.Printf("Insert benchmark done for %s: %.0f/sec", dbName, res.Insert.Throughput)
	}

	if !*skipQuery {
		log.Printf("Benchmarking queries for %s...", dbName)

		res.Queries = runner.RunQueries(ctx, repo)

		log.Printf("Query benchmark done for %s", dbName)
	}

	if s := repo.GetStorageStats(ctx); s != nil {
		res.Storage = s
	}

	return res
}

func newRepo(ctx context.Context, dbType string, cfg *config.Config) (benchmark.Repository, error) {
	switch dbType {
	case "postgres":
		return repository.NewPostgresRepo(ctx, &cfg.Postgres)
	case "mongodb":
		return repository.NewMongoDBRepo(ctx, cfg.MongoDB)
	case "cassandra":
		return repository.NewCassandraRepo(ctx, cfg.Cassandra)
	case "clickhouse":
		return repository.NewClickHouseRepo(ctx, &cfg.ClickHouse)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

func cleanupDatabases(ctx context.Context, cfg *config.Config, databases []string) {
	log.Println("Cleaning up databases...")

	for _, dbName := range databases {
		repo, err := newRepo(ctx, dbName, cfg)
		if err != nil {
			log.Printf("Failed to connect to %s for cleanup: %v", dbName, err)
			continue
		}

		if err := repo.Cleanup(ctx); err != nil {
			log.Printf("Failed to cleanup %s: %v", dbName, err)
		} else {
			log.Printf("Cleaned up %s", dbName)
		}

		if err := repo.Close(); err != nil {
			log.Printf("Failed to close %s: %v", dbName, err)
		}
	}
}
