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
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("API_ADDR", "127.0.0.1:9090")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("SNOWFLAKE_WORKER_ID", "42")
	t.Setenv("WEB_ORIGIN", "http://localhost:3001")

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
