package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/anhtuta/friendfold/backend/internal/auth"
	"github.com/anhtuta/friendfold/backend/internal/platform/config"
	"github.com/anhtuta/friendfold/backend/internal/platform/db"
	httpx "github.com/anhtuta/friendfold/backend/internal/platform/http"
	"github.com/anhtuta/friendfold/backend/internal/platform/id"
	platformlog "github.com/anhtuta/friendfold/backend/internal/platform/log"
	"github.com/anhtuta/friendfold/backend/internal/platform/ratelimit"
	redisx "github.com/anhtuta/friendfold/backend/internal/platform/redis"
	"github.com/anhtuta/friendfold/backend/internal/platform/session"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}

	logger := platformlog.New(cfg.LogLevel)

	gen, err := id.NewGenerator(cfg.SnowflakeWorkerID)
	if err != nil {
		logger.Error("snowflake generator", slog.String("error", err.Error()))
		return 1
	}

	hasher, err := auth.NewHasher(runtime.GOMAXPROCS(0))
	if err != nil {
		logger.Error("password hasher", slog.String("error", err.Error()))
		return 1
	}

	pools, err := db.New(context.Background(), cfg)
	if err != nil {
		logger.Error("db pools", slog.String("error", err.Error()))
		return 1
	}
	defer pools.Close()

	rdb, err := redisx.New(cfg.RedisURL)
	if err != nil {
		logger.Error("redis client", slog.String("error", err.Error()))
		return 1
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			logger.Error("redis close", slog.String("error", err.Error()))
		}
	}()

	mux := httpx.NewRouter(cfg, logger, httpx.ReadyDeps{
		Write: pools.Write,
		Read:  pools.Read,
		Redis: rdb,
	})
	auth.NewHandler(
		auth.NewService(
			auth.NewStore(pools.Write),
			session.NewStore(rdb.Cmdable()),
			hasher,
			gen,
			ratelimit.New(rdb.Cmdable()),
			logger,
		),
		cfg.CookieSecure,
		cfg.WebOrigin,
	).Mount(mux)

	srv := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", slog.String("addr", cfg.APIAddr))
		errCh <- srv.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", slog.String("error", err.Error()))
			return 1
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown", slog.String("error", err.Error()))
			return 1
		}
	}
	return 0
}
