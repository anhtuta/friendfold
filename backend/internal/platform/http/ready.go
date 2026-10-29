package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	readyOKBody   = `{"status":"ok"}`
	readyFailBody = `{"status":"unavailable"}`
	readyTimeout  = 2 * time.Second
)

// Pinger is a Postgres pool or Redis client that can be pinged.
type Pinger interface {
	Ping(ctx context.Context) error
}

// ReadyDeps are the downstreams GET /v1/ready must reach.
type ReadyDeps struct {
	Write Pinger
	Read  Pinger
	Redis Pinger
}

// Ready is GET /v1/ready: Postgres (write + read) and Redis reachability.
func Ready(logger *slog.Logger, deps ReadyDeps) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := pingAll(ctx, deps); err != nil {
			logger.Warn("ready check failed",
				slog.String("request_id", RequestIDFrom(r.Context())),
				slog.String("error", err.Error()),
			)
			writeJSON(w, http.StatusServiceUnavailable, readyFailBody)
			return
		}
		writeJSON(w, http.StatusOK, readyOKBody)
	}
}

func pingAll(ctx context.Context, deps ReadyDeps) error {
	if deps.Write == nil {
		return fmt.Errorf("postgres write: not configured")
	}
	if err := deps.Write.Ping(ctx); err != nil {
		return fmt.Errorf("postgres write: %w", err)
	}
	if deps.Read == nil {
		return fmt.Errorf("postgres read: not configured")
	}
	if err := deps.Read.Ping(ctx); err != nil {
		return fmt.Errorf("postgres read: %w", err)
	}
	if deps.Redis == nil {
		return fmt.Errorf("redis: not configured")
	}
	if err := deps.Redis.Ping(ctx); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
