package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds process-level settings loaded from the environment.
type Config struct {
	APIAddr           string
	LogLevel          string
	SnowflakeWorkerID uint16
	WebOrigin         string
	DatabaseURL       string
	DatabaseReadURL   string
	RedisURL          string
	PGXMaxConns       int32
}

const (
	defaultAPIAddr   = ":8080"
	defaultLogLevel  = "info"
	defaultWorkerID  = uint16(1)
	defaultWebOrigin = "http://localhost:3000"
	// Local Compose credentials (docker-compose.yml). Not for production.
	defaultDatabaseURL = "postgres://friendfold:friendfold@127.0.0.1:5432/friendfold?sslmode=disable"
	defaultRedisURL    = "redis://127.0.0.1:6379/0"
	defaultPGXMaxConns = int32(10)

	maxSnowflakeWorkerID = 1023 // 10 bits in the Snowflake layout
	minPGXMaxConns       = 1
	maxPGXMaxConns       = 100
)

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		APIAddr:     envOr("API_ADDR", defaultAPIAddr),
		LogLevel:    strings.ToLower(envOr("LOG_LEVEL", defaultLogLevel)),
		WebOrigin:   envOr("WEB_ORIGIN", defaultWebOrigin),
		DatabaseURL: envOr("DATABASE_URL", defaultDatabaseURL),
		RedisURL:    envOr("REDIS_URL", defaultRedisURL),
	}

	cfg.DatabaseReadURL = envOr("DATABASE_READ_URL", cfg.DatabaseURL)

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("config: invalid LOG_LEVEL %q (want debug, info, warn, or error)", cfg.LogLevel)
	}

	rawWorker := envOr("SNOWFLAKE_WORKER_ID", strconv.Itoa(int(defaultWorkerID)))
	worker, err := strconv.ParseUint(rawWorker, 10, 16)
	if err != nil {
		return Config{}, fmt.Errorf("config: parse SNOWFLAKE_WORKER_ID: %w", err)
	}
	if worker > maxSnowflakeWorkerID {
		return Config{}, fmt.Errorf("config: SNOWFLAKE_WORKER_ID %d exceeds max %d", worker, maxSnowflakeWorkerID)
	}
	cfg.SnowflakeWorkerID = uint16(worker)

	rawMaxConns := envOr("PGX_MAX_CONNS", strconv.Itoa(int(defaultPGXMaxConns)))
	maxConns, err := strconv.ParseInt(rawMaxConns, 10, 32)
	if err != nil {
		return Config{}, fmt.Errorf("config: parse PGX_MAX_CONNS: %w", err)
	}
	if maxConns < minPGXMaxConns || maxConns > maxPGXMaxConns {
		return Config{}, fmt.Errorf("config: PGX_MAX_CONNS %d out of range %d–%d", maxConns, minPGXMaxConns, maxPGXMaxConns)
	}
	cfg.PGXMaxConns = int32(maxConns)

	return cfg, nil
}

func envOr(key, fallback string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}
