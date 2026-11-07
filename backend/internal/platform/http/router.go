// Package httpx is the HTTP kernel: chi router, request IDs, CORS, health, and ready.
package httpx

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/anhtuta/friendfold/backend/internal/platform/config"
)

// NewRouter builds the chi mux for the API process. Feature modules mount
// additional routes on the returned mux in cmd/api.
func NewRouter(cfg config.Config, logger *slog.Logger, ready ReadyDeps) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(RequestID)
	r.Use(AccessLog(logger))
	r.Use(CORS(cfg.WebOrigin))

	r.Get("/v1/health", Health)
	r.Get("/v1/ready", Ready(logger, ready))
	return r
}
