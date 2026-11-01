package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// t.Setenv cannot be used with t.Parallel.
	t.Setenv("API_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SNOWFLAKE_WORKER_ID", "")
	t.Setenv("WEB_ORIGIN", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_READ_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("PGX_MAX_CONNS", "")
	t.Setenv("COOKIE_SECURE", "")
	// Empty values fall back to defaults. Clearing avoids leakage from the
	// process environment.

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIAddr != defaultAPIAddr {
		t.Errorf("APIAddr = %q, want %q", cfg.APIAddr, defaultAPIAddr)
	}
	if cfg.LogLevel != defaultLogLevel {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, defaultLogLevel)
	}
	if cfg.SnowflakeWorkerID != defaultWorkerID {
		t.Errorf("SnowflakeWorkerID = %d, want %d", cfg.SnowflakeWorkerID, defaultWorkerID)
	}
	if cfg.WebOrigin != defaultWebOrigin {
		t.Errorf("WebOrigin = %q, want %q", cfg.WebOrigin, defaultWebOrigin)
	}
	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}
	if cfg.DatabaseReadURL != defaultDatabaseURL {
		t.Errorf("DatabaseReadURL = %q, want write URL default %q", cfg.DatabaseReadURL, defaultDatabaseURL)
	}
	if cfg.RedisURL != defaultRedisURL {
		t.Errorf("RedisURL = %q, want %q", cfg.RedisURL, defaultRedisURL)
	}
	if cfg.PGXMaxConns != defaultPGXMaxConns {
		t.Errorf("PGXMaxConns = %d, want %d", cfg.PGXMaxConns, defaultPGXMaxConns)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure = true, want false when WEB_ORIGIN is http")
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("API_ADDR", "127.0.0.1:9090")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("SNOWFLAKE_WORKER_ID", "42")
	t.Setenv("WEB_ORIGIN", "http://localhost:3001")
	t.Setenv("DATABASE_URL", "postgres://w@db/write")
	t.Setenv("DATABASE_READ_URL", "postgres://r@db/read")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6380/1")
	t.Setenv("PGX_MAX_CONNS", "8")
	t.Setenv("COOKIE_SECURE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIAddr != "127.0.0.1:9090" {
		t.Errorf("APIAddr = %q", cfg.APIAddr)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.SnowflakeWorkerID != 42 {
		t.Errorf("SnowflakeWorkerID = %d, want 42", cfg.SnowflakeWorkerID)
	}
	if cfg.WebOrigin != "http://localhost:3001" {
		t.Errorf("WebOrigin = %q", cfg.WebOrigin)
	}
	if cfg.DatabaseURL != "postgres://w@db/write" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.DatabaseReadURL != "postgres://r@db/read" {
		t.Errorf("DatabaseReadURL = %q", cfg.DatabaseReadURL)
	}
	if cfg.RedisURL != "redis://127.0.0.1:6380/1" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.PGXMaxConns != 8 {
		t.Errorf("PGXMaxConns = %d, want 8", cfg.PGXMaxConns)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure = false, want true from COOKIE_SECURE")
	}
}

func TestCookieSecureFollowsHTTPSOrigin(t *testing.T) {
	t.Setenv("WEB_ORIGIN", "https://friendfold.example")
	t.Setenv("COOKIE_SECURE", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure = false, want true for https WEB_ORIGIN")
	}
}

func TestCookieSecureRejectsGarbage(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "sometimes")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid COOKIE_SECURE")
	}
}

func TestLoadReadURLDefaultsToWriteURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://w@db/write")
	t.Setenv("DATABASE_READ_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseReadURL != "postgres://w@db/write" {
		t.Errorf("DatabaseReadURL = %q, want write URL", cfg.DatabaseReadURL)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestLoadRejectsWorkerIDTooLarge(t *testing.T) {
	t.Setenv("SNOWFLAKE_WORKER_ID", "1024")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for worker ID 1024")
	}
}

func TestLoadAcceptsWorkerIDZero(t *testing.T) {
	t.Setenv("SNOWFLAKE_WORKER_ID", "0")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SnowflakeWorkerID != 0 {
		t.Errorf("SnowflakeWorkerID = %d, want 0", cfg.SnowflakeWorkerID)
	}
}

func TestLoadRejectsPGXMaxConnsOutOfRange(t *testing.T) {
	t.Setenv("PGX_MAX_CONNS", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for PGX_MAX_CONNS 0")
	}
	t.Setenv("PGX_MAX_CONNS", "101")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for PGX_MAX_CONNS 101")
	}
}
