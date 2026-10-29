// Package httpx is the HTTP kernel: chi router, request IDs, CORS, and health.
package httpx

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/anhtuta/friendfold/backend/internal/platform/config"
)

// NewRouter builds the chi mux for the API process.
func NewRouter(cfg config.Config, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(RequestID)
	r.Use(AccessLog(logger))
	r.Use(CORS(cfg.WebOrigin))

	r.Get("/v1/health", Health)
	return r
}
