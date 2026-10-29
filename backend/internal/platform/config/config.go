package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds process-level settings loaded from the environment.
// Database and Redis URLs are added in feature 01 phase 2.
type Config struct {
	APIAddr           string
	LogLevel          string
	SnowflakeWorkerID uint16
	WebOrigin         string
}

const (
	defaultAPIAddr   = ":8080"
	defaultLogLevel  = "info"
	defaultWorkerID  = uint16(1)
	defaultWebOrigin = "http://localhost:3000"

	maxSnowflakeWorkerID = 1023 // 10 bits in the Snowflake layout
)

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		APIAddr:   envOr("API_ADDR", defaultAPIAddr),
		LogLevel:  strings.ToLower(envOr("LOG_LEVEL", defaultLogLevel)),
		WebOrigin: envOr("WEB_ORIGIN", defaultWebOrigin),
	}

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
