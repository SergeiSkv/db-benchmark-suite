package config

import (
	"fmt"
	"os"
)

type Config struct {
	Postgres   PostgresConfig
	MongoDB    MongoDBConfig
	Cassandra  CassandraConfig
	ClickHouse ClickHouseConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

type MongoDBConfig struct {
	URI      string
	Database string
}

type CassandraConfig struct {
	Hosts    []string
	Keyspace string
}

type ClickHouseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func Load() (*Config, error) {
	return &Config{
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "benchmark"),
			Password: getEnv("POSTGRES_PASSWORD", "benchmark123"),
			Database: getEnv("POSTGRES_DB", "events"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGODB_URI", "mongodb://benchmark:benchmark123@localhost:27017"),
			Database: getEnv("MONGODB_DB", "events"),
		},
		Cassandra: CassandraConfig{
			Hosts:    []string{getEnv("CASSANDRA_HOST", "127.0.0.1")},
			Keyspace: getEnv("CASSANDRA_KEYSPACE", "events"),
		},
		ClickHouse: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
			Port:     getEnv("CLICKHOUSE_PORT", "9000"),
			User:     getEnv("CLICKHOUSE_USER", "benchmark"),
			Password: getEnv("CLICKHOUSE_PASSWORD", "benchmark123"),
			Database: getEnv("CLICKHOUSE_DB", "events"),
		},
	}, nil
}

func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}
