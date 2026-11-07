// Command api is the Friendfold HTTP process: one binary that loads config,
// opens Postgres and Redis, mounts feature modules on the platform mux, and
// serves /v1 until SIGINT or SIGTERM.
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
	"github.com/anhtuta/friendfold/backend/internal/platform/mail"
	"github.com/anhtuta/friendfold/backend/internal/platform/ratelimit"
	redisx "github.com/anhtuta/friendfold/backend/internal/platform/redis"
	"github.com/anhtuta/friendfold/backend/internal/platform/session"
)

func main() {
	// run owns defers (pool/Redis Close). os.Exit from main would skip those.
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		// Logger is not up yet (and LOG_LEVEL may be the thing that failed).
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}

	logger := platformlog.New(cfg.LogLevel)

	// IDs are minted in-process (ADR 0006). A worker ID outside 0–1023 cannot
	// generate; fail here instead of returning 500 on every register.
	gen, err := id.NewGenerator(cfg.SnowflakeWorkerID)
	if err != nil {
		logger.Error("snowflake generator", slog.String("error", err.Error()))
		return 1
	}

	// Argon2id is memory-hard (~19 MiB per job). Cap in-flight hashes at
	// GOMAXPROCS(0) (query, not set) so a login flood cannot OOM the process.
	// NewHasher also precomputes the dummy hash used to pad unknown-email
	// logins; if that fails, CompareDummy cannot run and login would leak
	// whether an email exists.
	hasher, err := auth.NewHasher(runtime.GOMAXPROCS(0))
	if err != nil {
		logger.Error("password hasher", slog.String("error", err.Error()))
		return 1
	}

	// Pools are opened without a ping so a down database still lets /v1/health
	// answer (liveness). Readiness is GET /v1/ready via ReadyDeps below.
	pools, err := db.New(context.Background(), cfg)
	if err != nil {
		logger.Error("db pools", slog.String("error", err.Error()))
		return 1
	}
	defer pools.Close()

	// Sessions and auth rate limits live in Redis. Like Postgres, New does not
	// ping so liveness stays independent of Redis.
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

	// ReadyDeps are the three backends GET /v1/ready must ping (write Postgres,
	// read Postgres, Redis). Orchestrators use that to withhold traffic until
	// the process can actually serve; /v1/health stays a no-dependency liveness
	// check. NewRouter also applies CORS for cfg.WebOrigin.
	mux := httpx.NewRouter(cfg, logger, httpx.ReadyDeps{
		Write: pools.Write,
		Read:  pools.Read,
		Redis: rdb,
	})

	mailCtx, mailCancel := context.WithCancel(context.Background())
	defer mailCancel()
	mailer := mail.NewQueue(&mail.SMTP{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		StartTLS: cfg.SMTPStartTLS,
	}, logger, 8, mailCtx)

	// Mount auth before ListenAndServe so the first accepted connection already
	// has /v1/auth. CookieSecure is the session cookie Secure flag: off for
	// local HTTP (COOKIE_SECURE=false or http WEB_ORIGIN), on for HTTPS so
	// browsers will not attach friendfold_session to cleartext requests.
	// WebOrigin is the Next.js origin: mutating /v1/auth routes RequireOrigin
	// against it (CSRF) in addition to mux-level CORS.
	auth.NewHandler(
		auth.NewService(
			auth.NewStore(pools.Write),
			session.NewStore(rdb.Cmdable()),
			hasher,
			gen,
			ratelimit.New(rdb.Cmdable()),
			mailer,
			cfg.WebOrigin,
			logger,
		),
		cfg.CookieSecure,
		cfg.WebOrigin,
	).Mount(mux)

	// ReadHeaderTimeout bounds Slowloris-style header reads.
	srv := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Buffered so ListenAndServe can still send after Shutdown takes the
	// signal path and run() is already leaving the select.
	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", slog.String("addr", cfg.APIAddr))
		errCh <- srv.ListenAndServe()
	}()

	// SIGINT/SIGTERM cancel ctx so we Shutdown instead of dying mid-request.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", slog.String("error", err.Error()))
			return 1
		}
	case <-ctx.Done():
		// Drain in-flight requests instead of closing sockets immediately.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown", slog.String("error", err.Error()))
			mailCancel()
			_ = mailer.Shutdown(shutdownCtx)
			return 1
		}
		mailCancel()
		if err := mailer.Shutdown(shutdownCtx); err != nil {
			logger.Error("mail shutdown", slog.String("error", err.Error()))
			return 1
		}
	}
	return 0
}
