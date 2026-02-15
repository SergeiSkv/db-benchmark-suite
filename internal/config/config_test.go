package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, "5432", cfg.Postgres.Port)
	assert.Equal(t, "benchmark", cfg.Postgres.User)
	assert.Equal(t, "benchmark123", cfg.Postgres.Password)
	assert.Equal(t, "events", cfg.Postgres.Database)
	assert.Equal(t, "disable", cfg.Postgres.SSLMode)

	assert.Equal(t, "mongodb://benchmark:benchmark123@localhost:27017", cfg.MongoDB.URI)
	assert.Equal(t, "events", cfg.MongoDB.Database)

	assert.Equal(t, []string{"127.0.0.1"}, cfg.Cassandra.Hosts)
	assert.Equal(t, "events", cfg.Cassandra.Keyspace)

	assert.Equal(t, "localhost", cfg.ClickHouse.Host)
	assert.Equal(t, "9000", cfg.ClickHouse.Port)
	assert.Equal(t, "benchmark", cfg.ClickHouse.User)
	assert.Equal(t, "events", cfg.ClickHouse.Database)
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "pghost")
	t.Setenv("POSTGRES_PORT", "5555")
	t.Setenv("MONGODB_URI", "mongodb://custom:27017")
	t.Setenv("CLICKHOUSE_HOST", "chhost")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "pghost", cfg.Postgres.Host)
	assert.Equal(t, "5555", cfg.Postgres.Port)
	assert.Equal(t, "mongodb://custom:27017", cfg.MongoDB.URI)
	assert.Equal(t, "chhost", cfg.ClickHouse.Host)
}

func TestPostgresConfigDSN(t *testing.T) {
	cfg := PostgresConfig{
		Host:     "myhost",
		Port:     "5432",
		User:     "myuser",
		Password: "mypass",
		Database: "mydb",
		SSLMode:  "require",
	}

	dsn := cfg.DSN()
	assert.Equal(t, "host=myhost port=5432 user=myuser password=mypass dbname=mydb sslmode=require", dsn)
}
