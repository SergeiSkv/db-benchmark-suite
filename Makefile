.PHONY: help build run test clean docker-up docker-down benchmark-all benchmark-postgres benchmark-mongodb benchmark-cassandra benchmark-clickhouse

# Default target
help:
	@echo "Database Benchmark Suite - Available commands:"
	@echo ""
	@echo "  Infrastructure:"
	@echo "  make docker-up              - Start all databases"
	@echo "  make docker-down            - Stop all databases"
	@echo "  make clean                  - Stop databases and remove volumes"
	@echo ""
	@echo "  Benchmarks:"
	@echo "  make benchmark-all          - Run benchmarks on all databases"
	@echo "  make benchmark-postgres     - Run PostgreSQL benchmark only"
	@echo "  make benchmark-mongodb      - Run MongoDB benchmark only"
	@echo "  make benchmark-cassandra    - Run Cassandra benchmark only"
	@echo "  make benchmark-clickhouse   - Run ClickHouse benchmark only"
	@echo ""
	@echo "  Quick:"
	@echo "  make quick-test             - Quick test with 10K events"
	@echo "  make full-test              - Full test with 1M events"
	@echo ""
	@echo "  Development:"
	@echo "  make build                  - Build benchmark binary"
	@echo "  make test                   - Run unit tests"
	@echo "  make coverage               - Run tests with coverage report"
	@echo "  make lint                   - Run linter"
	@echo ""

# Build the benchmark binary
build:
	@echo "Building benchmark binary..."
	go build -o bin/benchmark ./cmd/benchmark

# Start all databases
docker-up:
	@echo "Starting databases..."
	docker-compose up -d postgres mongodb clickhouse cassandra
	@echo "Waiting for databases to be ready..."
	@sleep 30
	@echo "Databases are ready!"

# Stop all databases
docker-down:
	@echo "Stopping databases..."
	docker-compose down

# Clean everything including volumes
clean:
	@echo "Cleaning up..."
	docker-compose down -v
	rm -rf bin/
	@echo "Cleanup complete!"

# Run all benchmarks with default settings
benchmark-all: build
	@echo "Running benchmarks on all databases..."
	./bin/benchmark -db all -events 100000 -batch 5000 -workers 4 -output table

# Individual database benchmarks
benchmark-postgres: build
	@echo "Running PostgreSQL benchmark..."
	./bin/benchmark -db postgres -events 100000 -batch 5000 -workers 4 -output table

benchmark-mongodb: build
	@echo "Running MongoDB benchmark..."
	./bin/benchmark -db mongodb -events 100000 -batch 5000 -workers 4 -output table

benchmark-cassandra: build
	@echo "Running Cassandra benchmark..."
	./bin/benchmark -db cassandra -events 100000 -batch 5000 -workers 4 -output table

benchmark-clickhouse: build
	@echo "Running ClickHouse benchmark..."
	./bin/benchmark -db clickhouse -events 100000 -batch 5000 -workers 4 -output table

# Quick test with smaller dataset
quick-test: build
	@echo "Running quick test (10K events)..."
	./bin/benchmark -db all -events 10000 -batch 1000 -workers 4 -queries 10 -output table

# Full test with large dataset
full-test: build
	@echo "Running full test (1M events) - This will take a while..."
	./bin/benchmark -db all -events 1000000 -batch 10000 -workers 8 -queries 100 -output markdown > results.md
	@echo "Results saved to results.md"

# Generate markdown report
report: build
	./bin/benchmark -db all -events 100000 -batch 5000 -workers 4 -output markdown > benchmark-results.md
	@echo "Report saved to benchmark-results.md"

# Generate JSON report
report-json: build
	./bin/benchmark -db all -events 100000 -batch 5000 -workers 4 -output json > benchmark-results.json
	@echo "Report saved to benchmark-results.json"

# Run unit tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Show coverage
coverage: test
	go tool cover -html=coverage.out

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Lint code (requires golangci-lint)
lint:
	golangci-lint run ./...

# Run benchmarks and cleanup after
benchmark-cleanup: benchmark-all
	./bin/benchmark -db all -cleanup

# Check database connectivity
check-db:
	@echo "Checking database connectivity..."
	@docker-compose ps
	@echo ""
	@echo "Testing PostgreSQL..."
	@docker exec benchmark-postgres pg_isready -U benchmark || echo "PostgreSQL not ready"
	@echo ""
	@echo "Testing MongoDB..."
	@docker exec benchmark-mongodb mongosh --eval "db.adminCommand('ping')" || echo "MongoDB not ready"
	@echo ""
	@echo "Testing ClickHouse..."
	@docker exec benchmark-clickhouse clickhouse-client --query "SELECT 1" || echo "ClickHouse not ready"
	@echo ""
	@echo "Testing Cassandra..."
	@docker exec benchmark-cassandra cqlsh -e "DESCRIBE KEYSPACES" || echo "Cassandra not ready"

# View logs
logs:
	docker-compose logs -f

logs-postgres:
	docker-compose logs -f postgres

logs-mongodb:
	docker-compose logs -f mongodb

logs-cassandra:
	docker-compose logs -f cassandra

logs-clickhouse:
	docker-compose logs -f clickhouse
