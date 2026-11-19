package config

import (
	"strings"
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
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("SMTP_FROM", "")
	t.Setenv("SMTP_STARTTLS", "")
	t.Setenv("MEDIA_DRIVER", "")
	t.Setenv("MEDIA_DIR", "")
	t.Setenv("MEDIA_PUBLIC_BASE_URL", "")
	t.Setenv("MEDIA_S3_ENDPOINT", "")
	t.Setenv("MEDIA_S3_REGION", "")
	t.Setenv("MEDIA_S3_BUCKET", "")
	t.Setenv("MEDIA_S3_ACCESS_KEY", "")
	t.Setenv("MEDIA_S3_SECRET_KEY", "")
	t.Setenv("MEDIA_S3_USE_SSL", "")
	t.Setenv("MEDIA_S3_PATH_STYLE", "")
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
	if cfg.SMTPHost != defaultSMTPHost {
		t.Errorf("SMTPHost = %q, want %q", cfg.SMTPHost, defaultSMTPHost)
	}
	if cfg.SMTPPort != defaultSMTPPort {
		t.Errorf("SMTPPort = %d, want %d", cfg.SMTPPort, defaultSMTPPort)
	}
	if cfg.SMTPFrom != defaultSMTPFrom {
		t.Errorf("SMTPFrom = %q, want %q", cfg.SMTPFrom, defaultSMTPFrom)
	}
	if cfg.SMTPStartTLS {
		t.Error("SMTPStartTLS = true, want false")
	}
	if cfg.SMTPUsername != "" || cfg.SMTPPassword != "" {
		t.Error("SMTP auth should be empty by default")
	}
	if cfg.MediaDriver != defaultMediaDriver {
		t.Errorf("MediaDriver = %q, want %q", cfg.MediaDriver, defaultMediaDriver)
	}
	if cfg.MediaDir != defaultMediaDir {
		t.Errorf("MediaDir = %q, want %q", cfg.MediaDir, defaultMediaDir)
	}
	if cfg.MediaPublicBaseURL != defaultMediaPublicBaseURL {
		t.Errorf("MediaPublicBaseURL = %q, want %q", cfg.MediaPublicBaseURL, defaultMediaPublicBaseURL)
	}
	if cfg.MediaS3Endpoint != defaultMediaS3Endpoint {
		t.Errorf("MediaS3Endpoint = %q", cfg.MediaS3Endpoint)
	}
	if cfg.MediaS3AccessKey != defaultMediaS3AccessKey || cfg.MediaS3SecretKey != defaultMediaS3SecretKey {
		t.Error("S3 keys should default to local RustFS friendfold/friendfold")
	}
	if cfg.MediaS3UseSSL {
		t.Error("MediaS3UseSSL = true, want false")
	}
	if !cfg.MediaS3PathStyle {
		t.Error("MediaS3PathStyle = false, want true")
	}
}

func TestLoadDefaultsUseComposeHostPorts(t *testing.T) {
	t.Parallel()
	if !strings.Contains(defaultDatabaseURL, "127.0.0.1:5435/") {
		t.Errorf("defaultDatabaseURL = %q, want host port 5435", defaultDatabaseURL)
	}
	if !strings.Contains(defaultRedisURL, "127.0.0.1:6377/") {
		t.Errorf("defaultRedisURL = %q, want host port 6377", defaultRedisURL)
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
	t.Setenv("SMTP_HOST", "127.0.0.1")
	t.Setenv("SMTP_PORT", "1025")
	t.Setenv("SMTP_FROM", "Friendfold <dev@localhost>")
	t.Setenv("SMTP_STARTTLS", "false")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("MEDIA_DRIVER", "s3")
	t.Setenv("MEDIA_DIR", "/tmp/media")
	t.Setenv("MEDIA_PUBLIC_BASE_URL", "https://cdn.example.com/")
	t.Setenv("MEDIA_S3_ENDPOINT", "http://127.0.0.1:9000")
	t.Setenv("MEDIA_S3_BUCKET", "friendfold-media")
	t.Setenv("MEDIA_S3_USE_SSL", "true")
	t.Setenv("MEDIA_S3_PATH_STYLE", "false")

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
	if cfg.SMTPHost != "127.0.0.1" || cfg.SMTPPort != 1025 {
		t.Errorf("SMTP = %s:%d", cfg.SMTPHost, cfg.SMTPPort)
	}
	if cfg.SMTPFrom != "Friendfold <dev@localhost>" {
		t.Errorf("SMTPFrom = %q", cfg.SMTPFrom)
	}
	if cfg.SMTPStartTLS {
		t.Error("SMTPStartTLS = true, want false")
	}
	if cfg.MediaDriver != "s3" {
		t.Errorf("MediaDriver = %q, want s3", cfg.MediaDriver)
	}
	if cfg.MediaDir != "/tmp/media" {
		t.Errorf("MediaDir = %q", cfg.MediaDir)
	}
	if cfg.MediaPublicBaseURL != "https://cdn.example.com" {
		t.Errorf("MediaPublicBaseURL = %q (trailing slash should be stripped)", cfg.MediaPublicBaseURL)
	}
	if !cfg.MediaS3UseSSL {
		t.Error("MediaS3UseSSL = false, want true")
	}
	if cfg.MediaS3PathStyle {
		t.Error("MediaS3PathStyle = true, want false")
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

func TestLoadRejectsSMTPPortOutOfRange(t *testing.T) {
	t.Setenv("SMTP_PORT", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for SMTP_PORT 0")
	}
	t.Setenv("SMTP_PORT", "65536")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for SMTP_PORT 65536")
	}
}

func TestLoadSMTPStartTLS(t *testing.T) {
	t.Setenv("SMTP_STARTTLS", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.SMTPStartTLS {
		t.Error("SMTPStartTLS = false, want true")
	}
}

func TestLoadRejectsInvalidMediaDriver(t *testing.T) {
	t.Setenv("MEDIA_DRIVER", "gcs")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid MEDIA_DRIVER")
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
