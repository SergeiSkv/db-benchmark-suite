# Database Benchmark Suite

Comprehensive benchmark suite comparing PostgreSQL, MongoDB, Cassandra, and ClickHouse for event analytics workloads.

**Author:** Serge Skoredin (https://skoredin.pro)

## What is this?

This project reproduces the results from the article [SQL vs NoSQL vs Columnar: Choosing the Right Database](https://skoredin.pro/blog/golang/sql-nosql-columnar-comparison).

Run the benchmarks yourself and verify the performance numbers:
- **Insert speed**: events/sec throughput for each database
- **Query performance**: analytics query latency (1 hour, 1 day, 1 week, 1 month)
- **Storage efficiency**: data size, index size, compression ratio

## Quick Start

### Requirements

- Docker & Docker Compose
- Go 1.22+
- At least 4GB free RAM (8GB+ total recommended)
- 50GB free disk space

### 3-step launch

```bash
# 1. Clone the repository
git clone https://github.com/skoredin/db-benchmark-suite.git
cd db-benchmark-suite

# 2. Run everything (databases start sequentially to save memory)
./run.sh

# 3. Quick test (10K events, ~30 seconds)
./run.sh --quick
```

The script starts each database one at a time, runs the benchmark, then stops it before moving to the next. This keeps memory usage under ~2GB at any point.

### Quick test (10K events)

```bash
make quick-test
```

## What gets tested

### Insert Performance
- Batch inserts with configurable batch size
- Parallel workers (defaults to CPU count)
- Throughput measurement (events/sec)

### Query Performance
Analytics queries with aggregation:
- **1 hour range**: last hour of events
- **1 day range**: last 24 hours
- **1 week range**: last 7 days
- **1 month range**: last 30 days

Metrics per query:
- Average, Min, Max latency
- P50, P95, P99 percentiles
- Error count

### Storage Statistics
- Total size (data + indexes)
- Index size
- Compression ratio
- Row count

## Usage

### Benchmark all databases

```bash
make benchmark-all
```

### Benchmark a specific database

```bash
make benchmark-postgres
make benchmark-mongodb
make benchmark-cassandra
make benchmark-clickhouse
```

### Custom configuration

```bash
# 10M events, batch 20K, 16 workers
./bin/benchmark \
  -db all \
  -events 10000000 \
  -batch 20000 \
  -workers 16 \
  -queries 100 \
  -output markdown
```

### CLI flags

```
-db string
    Database type: postgres, mongodb, cassandra, clickhouse, all (default "all")

-events int
    Number of events to generate (default 1000000)

-batch int
    Batch size for inserts (default 10000)

-workers int
    Number of concurrent workers (default: CPU count)

-queries int
    Number of query iterations (default 100)

-output string
    Output format: table, json, markdown (default "table")

-skip-insert
    Skip insert benchmark

-skip-query
    Skip query benchmark

-cleanup
    Cleanup data after benchmark
```

## Output Formats

### Table (default)

```bash
make benchmark-all
```

Console output with aligned columns.

### Markdown

```bash
./bin/benchmark -db all -output markdown > results.md
```

Ready-to-use markdown for articles or documentation.

### JSON

```bash
./bin/benchmark -db all -output json > results.json
```

Machine-readable format for further processing.

## Database Schemas

### PostgreSQL
```sql
CREATE TABLE events (
    id BIGSERIAL,
    event_id VARCHAR(255) NOT NULL,
    user_id BIGINT NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    payload TEXT,
    created_at TIMESTAMP NOT NULL
) PARTITION BY RANGE (created_at);
```

Indexes:
- `created_at` (BRIN, pages_per_range=32) - for time-range queries
- `(event_type, created_at)` - for analytics
- `user_id` - for user lookups
- `(event_id, created_at)` UNIQUE - deduplication

### MongoDB
```javascript
{
  event_id: String,    // unique index
  user_id: Number,     // index
  event_type: String,
  payload: String,
  created_at: Date     // index
}
```

### Cassandra
```cql
CREATE TABLE events (
    date_bucket text,      -- partition key (YYYYMMDD)
    created_at timestamp,  -- clustering key
    event_id text,         -- clustering key
    user_id bigint,
    event_type text,
    payload text,
    PRIMARY KEY ((date_bucket), event_type, created_at, event_id)
) WITH CLUSTERING ORDER BY (event_type ASC, created_at DESC);
```

### ClickHouse
```sql
CREATE TABLE events (
    event_id String,
    user_id UInt64,
    event_type LowCardinality(String),
    payload String,
    created_at DateTime
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (event_type, created_at, user_id);
```

## Configuration

### Environment Variables

Override connection settings via env vars:

```bash
# PostgreSQL
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=benchmark
export POSTGRES_PASSWORD=benchmark123
export POSTGRES_DB=events

# MongoDB
export MONGODB_URI=mongodb://benchmark:benchmark123@localhost:27017
export MONGODB_DB=events

# Cassandra
export CASSANDRA_HOST=localhost
export CASSANDRA_KEYSPACE=events

# ClickHouse
export CLICKHOUSE_HOST=localhost
export CLICKHOUSE_PORT=9000
export CLICKHOUSE_USER=benchmark
export CLICKHOUSE_PASSWORD=benchmark123
export CLICKHOUSE_DB=events
```

### Docker Resources

Each database is limited to ~1GB RAM by default. To adjust, edit `docker-compose.yml`:

```yaml
services:
  postgres:
    deploy:
      resources:
        limits:
          memory: 2G
```

## Testing

```bash
# Unit tests
make test

# Coverage
make coverage

# Lint
make lint
```

## Monitoring

Prometheus and Grafana are available as optional services:

```bash
docker-compose --profile monitoring up -d

# Grafana: http://localhost:3000 (admin / admin)
# Prometheus: http://localhost:9090
```

## Troubleshooting

### Cassandra won't start

Cassandra needs more time to initialize:

```bash
# Check logs
docker-compose logs cassandra

# Wait for "Starting listening for CQL clients"
# Usually takes 30-60 seconds
```

### MongoDB authentication error

```bash
# Recreate the container
docker-compose down -v
docker-compose up -d mongodb
```

### Out of Memory

Reduce workers and batch size:

```bash
./bin/benchmark -workers 2 -batch 5000 -events 100000
```

### Check connectivity

```bash
make check-db
```

## Contributing

Pull requests welcome! Areas of interest:
- Database schema optimizations
- New query types
- Support for other databases (ScyllaDB, TimescaleDB, DuckDB)
- Benchmark methodology improvements

## License

MIT License - free to use in your projects.

## Links

- **Blog**: https://skoredin.pro
- **Article**: https://skoredin.pro/blog/golang/sql-nosql-columnar-comparison
- **Author**: [@skoredin](https://github.com/sergeiSkv)

## Credits

Inspired by real-world pain of choosing the wrong database for analytics workloads.

---

**Questions?** Open an issue or reach out at [skoredin.pro](https://skoredin.pro)
