package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type requestLog struct {
	userID int64
}

type requestLogKey struct{}

// AccessLog writes one JSON line per request with request_id, route, status,
// and latency_ms. When a handler calls SetLogUserID, user_id is included.
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rl := &requestLog{}
			ctx := context.WithValue(r.Context(), requestLogKey{}, rl)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r.WithContext(ctx))
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			attrs := []any{
				slog.String("request_id", RequestIDFrom(ctx)),
				slog.String("method", r.Method),
				slog.String("route", r.URL.Path),
				slog.Int("status", status),
				slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			}
			if rl.userID != 0 {
				attrs = append(attrs, slog.Int64("user_id", rl.userID))
			}
			logger.Info("request", attrs...)
		})
	}
}

// SetLogUserID records user_id on the access log for this request. No-op if
// AccessLog is not in the chain. Never pass a session id.
func SetLogUserID(ctx context.Context, userID int64) {
	if userID == 0 {
		return
	}
	rl, ok := ctx.Value(requestLogKey{}).(*requestLog)
	if !ok || rl == nil {
		return
	}
	rl.userID = userID
}
