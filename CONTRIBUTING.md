# Contributing to Database Benchmark Suite

Thanks for your interest in contributing!

## How to contribute

### Reporting Issues

If you found a bug or want to suggest an improvement:

1. Check if a similar issue already exists
2. Create a new issue with a detailed description
3. For bugs, include:
   - Go version
   - Database versions (from docker-compose.yml)
   - Full error output
   - Steps to reproduce

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

The project follows standard Go conventions:

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test
```

### Tests

All new features must be covered by tests:

**Unit Tests:**
```go
func TestNewFeature(t *testing.T) {
    // Arrange
    // Act
    // Assert
    assert.Equal(t, expected, actual)
}
```

**Benchmark Tests:**
```go
func BenchmarkNewFeature(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Benchmark code
    }
}
```

**Fuzz Tests (where applicable):**
```go
func FuzzNewFeature(f *testing.F) {
    f.Add(input)
    f.Fuzz(func(t *testing.T, data []byte) {
        // Fuzz test code
    })
}
```

### Test requirements

- **Unit tests**: required for all packages
- **Testify**: use `github.com/stretchr/testify`
- **Gomock**: use `go.uber.org/mock` for mocks

### Running tests

```bash
# All tests
go test -v -race ./...

# With coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Unit tests only
go test -v -short ./...

# Benchmarks
go test -bench=. -benchmem ./...

# Fuzz tests
go test -fuzz=. -fuzztime=30s ./...
```

## Adding new databases

Want to add support for ScyllaDB, TimescaleDB, or DuckDB?

1. Create a new file in `internal/repository/`:
   ```go
   // internal/repository/scylladb.go
   package repository

   type ScyllaDBRepo struct {
       // ...
   }

   func NewScyllaDBRepo(ctx context.Context, cfg *config.ScyllaDBConfig) (*ScyllaDBRepo, error) {
       // Implementation
   }

   // Implement the Repository interface
   ```

2. Add configuration to `internal/config/config.go`

3. Add a service to `docker-compose.yml`

4. Update `internal/repository/repository.go`:
   ```go
   case "scylladb":
       return NewScyllaDBRepo(ctx, cfg.ScyllaDB)
   ```

5. Add tests and benchmarks

6. Update documentation

## Improving benchmarks

### New query types

Want to add tests for JOINs or full-text search?

1. Add a method to the `Repository` interface:
   ```go
   FullTextSearch(ctx context.Context, query string) ([]Result, error)
   ```

2. Implement for all databases

3. Add to `Runner.RunQueries()` in `internal/benchmark/runner.go`

### New metrics

Want to measure memory usage or CPU utilization?

1. Extend the `benchmark.Results` struct:
   ```go
   type Results struct {
       // ...
       MemoryStats *MemoryStats
       CPUStats    *CPUStats
   }
   ```

2. Collect metrics during the benchmark

3. Add output to `reporter`

## Code Review Guidelines

When reviewing PRs we look for:

**Code Quality:**
- Following Go best practices
- Proper error handling
- Context propagation
- Resource cleanup (defer)

**Performance:**
- No unnecessary allocations
- Efficient algorithms
- Proper use of batching

**Testing:**
- Edge cases covered
- Benchmarks for hot paths

**Documentation:**
- Godoc comments
- README updated
- Usage examples

## Project Structure

```
db-benchmark-suite/
├── cmd/
│   └── benchmark/          # Main application
├── internal/
│   ├── benchmark/          # Benchmark logic (runner, stats, results)
│   ├── config/             # Configuration
│   ├── generator/          # Event generator
│   ├── reporter/           # Results reporter
│   └── repository/         # Database implementations
├── docker-compose.yml      # Infrastructure
├── Makefile                # Build automation
└── README.md
```

## Questions?

- Open an issue with the `question` label
- Reach out at skoredin.pro

## License

All contributions are licensed under the MIT License.

---

**Thanks for contributing!**
