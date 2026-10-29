package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anhtuta/friendfold/backend/internal/platform/config"
)

func readyRouter(logger *slog.Logger, deps ReadyDeps) http.Handler {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	}
	return NewRouter(config.Config{
		WebOrigin: "http://localhost:3000",
	}, logger, deps)
}

func TestReadyOK(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ready", nil)
	readyRouter(nil, okDeps()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
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

func TestReadyWritePingFails(t *testing.T) {
	t.Parallel()
	deps := ReadyDeps{
		Write: pingStub{err: errors.New("write down")},
		Read:  pingStub{},
		Redis: pingStub{},
	}
	assertReadyUnavailable(t, deps, "postgres write")
}

func TestReadyReadPingFails(t *testing.T) {
	t.Parallel()
	deps := ReadyDeps{
		Write: pingStub{},
		Read:  pingStub{err: errors.New("read down")},
		Redis: pingStub{},
	}
	assertReadyUnavailable(t, deps, "postgres read")
}

func TestReadyRedisPingFails(t *testing.T) {
	t.Parallel()
	deps := ReadyDeps{
		Write: pingStub{},
		Read:  pingStub{},
		Redis: pingStub{err: errors.New("redis down")},
	}
	assertReadyUnavailable(t, deps, "redis")
}

func TestReadyMissingDepIsUnavailable(t *testing.T) {
	t.Parallel()
	assertReadyUnavailable(t, ReadyDeps{}, "not configured")
}

func TestHealthOKWhenReadyWouldFail(t *testing.T) {
	t.Parallel()
	deps := ReadyDeps{
		Write: pingStub{err: errors.New("write down")},
		Read:  pingStub{},
		Redis: pingStub{},
	}
	h := readyRouter(nil, deps)

	readyRec := httptest.NewRecorder()
	h.ServeHTTP(readyRec, httptest.NewRequest(http.MethodGet, "/v1/ready", nil))
	if readyRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status = %d, want 503", readyRec.Code)
	}

	healthRec := httptest.NewRecorder()
	h.ServeHTTP(healthRec, httptest.NewRequest(http.MethodGet, "/v1/health", nil))
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", healthRec.Code)
	}
}

func TestReadyUnavailableDoesNotLeakError(t *testing.T) {
	t.Parallel()
	deps := ReadyDeps{
		Write: pingStub{err: errors.New("postgres://secret@db/friendfold")},
		Read:  pingStub{},
		Redis: pingStub{},
	}
	rec := httptest.NewRecorder()
	readyRouter(nil, deps).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "postgres://") {
		t.Fatalf("body leaked dependency detail: %s", rec.Body.String())
	}
}

func TestReadyLogsFailure(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	deps := ReadyDeps{
		Write: pingStub{},
		Read:  pingStub{},
		Redis: pingStub{err: errors.New("redis down")},
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/ready", nil)
	req.Header.Set("X-Request-Id", "ready-log-id")
	readyRouter(logger, deps).ServeHTTP(httptest.NewRecorder(), req)

	line := buf.String()
	if !strings.Contains(line, `"request_id":"ready-log-id"`) {
		t.Fatalf("log missing request_id: %s", line)
	}
	if !strings.Contains(line, "ready check failed") {
		t.Fatalf("log missing ready failure: %s", line)
	}
}

func assertReadyUnavailable(t *testing.T, deps ReadyDeps, logNeedle string) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ready", nil)
	readyRouter(logger, deps).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
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
	if body.Status != "unavailable" {
		t.Fatalf("status field = %q, want unavailable", body.Status)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("missing X-Request-Id on 503")
	}
	if logNeedle != "" && !strings.Contains(buf.String(), logNeedle) {
		t.Fatalf("log missing %q: %s", logNeedle, buf.String())
	}
}
