package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anhtuta/friendfold/backend/internal/platform/config"
)

func testRouter(logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	}
	return NewRouter(config.Config{
		WebOrigin: "http://localhost:3000",
	}, logger)
}

func TestHealthOK(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	testRouter(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status field = %q, want ok", body.Status)
	}
}

func TestHealthGeneratesRequestID(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	testRouter(nil).ServeHTTP(rec, req)

	rid := rec.Header().Get("X-Request-Id")
	if rid == "" {
		t.Fatal("missing X-Request-Id")
	}
	if len(rid) != 32 {
		t.Fatalf("generated request id length = %d, want 32 hex chars", len(rid))
	}
}

func TestHealthEchoesRequestID(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("X-Request-Id", "client-req-123")
	testRouter(nil).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got != "client-req-123" {
		t.Fatalf("X-Request-Id = %q, want client-req-123", got)
	}
}

func TestHealthRejectsInvalidRequestID(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("X-Request-Id", "has space")
	testRouter(nil).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got == "has space" || got == "" {
		t.Fatalf("expected generated id, got %q", got)
	}
}

func TestAccessLogIncludesRequestID(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("X-Request-Id", "log-check-id")
	testRouter(logger).ServeHTTP(rec, req)

	line := buf.String()
	if !strings.Contains(line, `"request_id":"log-check-id"`) {
		t.Fatalf("log missing request_id: %s", line)
	}
	if !strings.Contains(line, `"route":"/v1/health"`) {
		t.Fatalf("log missing route: %s", line)
	}
	if !strings.Contains(line, `"status":200`) {
		t.Fatalf("log missing status: %s", line)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	testRouter(nil).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Allow-Origin = %q", got)
	}
}

func TestCORSDeniesOtherOrigin(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	testRouter(nil).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Allow-Origin %q", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (request still served)", rec.Code)
	}
}
