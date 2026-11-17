# Local development

How to run Friendfold on a laptop. Production deploy is later (`docs/deployment/deployment-prod.md` is not written yet).

Compose runs **Postgres, Redis, Mailpit, and MinIO**. The Go API and Next.js app run **on the host**. Never point this workflow at a production database. The API default object store is a **local directory** (`MEDIA_DRIVER=fs`); MinIO is there so `MEDIA_DRIVER=s3` can be exercised without AWS.

From the repo root, `make help` lists targets. This page uses those targets.

## Prerequisites

| Tool | Version used here | Notes |
| --- | --- | --- |
| Git | any recent | |
| Docker Engine + Compose v2 | `docker compose` (plugin) | Postgres 16, Redis 7, Mailpit, MinIO images |
| Go | **1.22+** module (`backend/go.mod`); CI uses **1.24.x** | goose v3.24.3 and sqlc v1.29.0 auto-select a newer compiler via `GOTOOLCHAIN=auto` |
| Node.js **22** + pnpm **12.18.0** | CI Node 22; Corepack pin in `frontend/package.json` | `frontend/pnpm-lock.yaml`. `corepack enable` then `corepack prepare pnpm@12.18.0 --activate`. Do not use npm/yarn in `frontend/`. |
| GNU make | macOS 3.81 or Linux 4.x | `make help` is the default target |

Optional: copy `.env.example` to `.env` only when you need to override defaults. The Compose Postgres password is **local-only**.

## Quick start

```bash
make up
make migrate
make dev
```

`make dev` starts Compose (if needed), applies migrations, then runs the API and Next.js together. Use two terminals instead if you want separate logs:

```bash
make up
make migrate
make api        # terminal 1 — http://localhost:8080
make frontend   # terminal 2 — http://localhost:3000
```

| Check | Expected |
| --- | --- |
| http://localhost:8080/v1/health | `{"status":"ok"}` |
| http://localhost:8080/v1/ready | `{"status":"ok"}` when Compose is up |
| http://localhost:3000 | Friendfold / Into the fold of friends. |
| http://localhost:3000/health | `OK` when the API ready check succeeds |
| http://localhost:8025 | Mailpit UI |
| http://localhost:9001 | MinIO console (local user `friendfold`) |

Stop Compose with `make down` (volumes stay). Ctrl-C stops `make api` / `make frontend` / `make dev`.

## Ports

| Service | Host | Container | Why |
| --- | --- | --- | --- |
| Postgres | **5435** | 5432 | Host 5432 is often a local Postgres |
| Redis | **6377** | 6379 | Host 6379 is often a local Redis |
| Mailpit SMTP | 1025 | 1025 | API default `SMTP_HOST` / `SMTP_PORT` |
| Mailpit UI | 8025 | 8025 | Browser: http://localhost:8025 |
| MinIO S3 API | 9000 | 9000 | `MEDIA_S3_ENDPOINT` when `MEDIA_DRIVER=s3` |
| MinIO console | 9001 | 9001 | Browser: http://localhost:9001 |
| Go API | 8080 | — (host process) | `API_ADDR` default `:8080` |
| Next.js | 3000 | — (host process) | `PORT` default `3000` |

Container-internal ports are unchanged. Host processes must use **5435** and **6377**, not 5432/6379.

If an old `.env` still has `127.0.0.1:5432` or `:6379`, the API will miss Compose. Delete those lines or update them to 5435/6377.

## Docker Compose

`docker-compose.yml` at the repo root. Project name: `friendfold`.

| Service | Image | Role |
| --- | --- | --- |
| `postgres` | `postgres:16` | System of record. User `friendfold`, password `friendfold`, db `friendfold`. Volume `postgres_data`. Healthcheck `pg_isready`. |
| `redis` | `redis:7` | Sessions, cache, rate limits (ready check is `PING`). Healthcheck `redis-cli ping`. |
| `mailpit` | `axllent/mailpit` | Dev SMTP catcher. No auth. Not for production. |
| `minio` | `minio/minio` | S3-compatible object storage. User `friendfold`, password `friendfold` (local-only). Volume `minio_data`. Default API driver is still `fs`. |

```bash
make up      # docker compose up -d
make logs    # docker compose logs -f
make down    # docker compose down  (keeps postgres_data)
```

## Migrations (goose)

Pin: **goose v3.24.3** (`go run github.com/pressly/goose/v3/cmd/goose@v3.24.3`). Files in `backend/db/migrations/`. Version table: `goose_db_version`.

```bash
make migrate        # goose up against 127.0.0.1:5435
make migrate-down   # goose down (local/dev only)
```

Default DSN (override with `DATABASE_URL`):

```text
postgres://friendfold:friendfold@127.0.0.1:5435/friendfold?sslmode=disable
```

Never edit a migration that already landed on `main`; add a new file.

## Backend (Go API)

Module `github.com/anhtuta/friendfold/backend`. Process: `backend/cmd/api`. Health: `GET /v1/health` (no DB). Ready: `GET /v1/ready` (Postgres + Redis ping).

```bash
make api            # CGO_ENABLED=0 go run ./cmd/api  (alias: make backend)
make test-backend   # CGO_ENABLED=0 go test ./...
make lint-backend   # golangci-lint v2.4.0
make sqlc           # sqlc v1.29.0 generate
make sqlc-diff      # sqlc diff (same check as CI)
```

`DATABASE_URL`, `DATABASE_READ_URL`, and `REDIS_URL` default to the Compose host ports so `make api` works without a filled `.env`. `DATABASE_READ_URL` falls back to `DATABASE_URL`.

## Frontend (Next.js)

App Router in `frontend/`. Server Components by default. `API_BASE_URL` is server-only (health page). Browser calls use `NEXT_PUBLIC_API_BASE_URL` and CORS `WEB_ORIGIN`. Package manager is **pnpm** (Corepack `packageManager` `pnpm@12.18.0`). `npm install` / `yarn` in `frontend/` are rejected.

```bash
make frontend       # pnpm install && pnpm dev
make test-frontend  # vitest
make lint-frontend  # eslint
make typecheck      # next typegen && tsc --noEmit
```

## Mailpit

Local SMTP: `127.0.0.1:1025`, no username/password, `SMTP_STARTTLS=false`. From-address default: `Friendfold <friendfold@localhost>`.

- Sign-up and password-reset mail appears in http://localhost:8025.
- Optional test-only `MAILPIT_URL=http://127.0.0.1:8025` (see `.env.example`). The Mailpit round-trip test **skips** when SMTP `:1025` is unreachable.

Production SES/SMTP is a later env; do not put those credentials in git.

## Environment defaults

Names live in `.env.example`. Empty values fall back to these (see `backend/internal/platform/config` and `frontend/lib/apiBaseUrl.ts`).

| Variable | Default | Used by |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://friendfold:friendfold@127.0.0.1:5435/friendfold?sslmode=disable` | API, `make migrate` |
| `DATABASE_READ_URL` | same as `DATABASE_URL` | API read pool |
| `REDIS_URL` | `redis://127.0.0.1:6377/0` | API |
| `API_ADDR` | `:8080` | API listen |
| `WEB_ORIGIN` | `http://localhost:3000` | API CORS allowlist; email links |
| `API_BASE_URL` | `http://localhost:8080` | Next.js server → API |
| `NEXT_PUBLIC_API_BASE_URL` | `http://localhost:8080` | Browser → API |
| `SNOWFLAKE_WORKER_ID` | `1` (0–1023) | API |
| `LOG_LEVEL` | `info` | API (`debug`/`info`/`warn`/`error`) |
| `PGX_MAX_CONNS` | `10` (1–100) | API |
| `COOKIE_SECURE` | unset → false on `http` `WEB_ORIGIN`, true on `https` | API session cookie |
| `SMTP_HOST` | `127.0.0.1` | API |
| `SMTP_PORT` | `1025` | API |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | empty | Mailpit has no auth |
| `SMTP_FROM` | `Friendfold <friendfold@localhost>` | API |
| `SMTP_STARTTLS` | `false` | API |
| `MEDIA_DRIVER` | `fs` | `fs` or `s3` ([ADR 0009](../decisions/0009-object-storage.md)) |
| `MEDIA_DIR` | `data/media` | `fs` driver directory (gitignored) |
| `MEDIA_PUBLIC_BASE_URL` | `http://localhost:8080/v1/media` | Prefix for `avatarUrl` / `coverUrl` |
| `MEDIA_S3_ENDPOINT` | `http://127.0.0.1:9000` | MinIO host port |
| `MEDIA_S3_REGION` | `us-east-1` | SDK region; unused by MinIO |
| `MEDIA_S3_BUCKET` | `friendfold-media` | Created on first `s3` driver start if missing |
| `MEDIA_S3_ACCESS_KEY` / `MEDIA_S3_SECRET_KEY` | `friendfold` | Local MinIO root user. Not for production. |
| `MEDIA_S3_USE_SSL` | `false` | |
| `MEDIA_S3_PATH_STYLE` | `true` | Required for MinIO |
| `PORT` | `3000` | Next.js |

## Other make targets

| Target | What it runs |
| --- | --- |
| `make help` | This list (default) |
| `make test` | Backend + frontend tests |
| `make lint` | golangci-lint, ESLint, OpenAPI (Redocly CLI 1.34.5) |
| `make openapi` | `npx --yes @redocly/cli@1.34.5 lint --config redocly.yaml` |
| `make typecheck` | Frontend `pnpm typecheck` |

CI (GitHub Actions) runs the same backend test/lint/sqlc-diff, frontend typecheck/test/lint/build, and OpenAPI lint. CI does **not** start Compose or bind host 5435/6377.

## Troubleshooting

**`connection refused` on 5432 or Redis on 6379.** Host Compose ports are **5435** and **6377**. Update `.env` / `DATABASE_URL` / `REDIS_URL`. Config defaults already use the new ports if those vars are unset.

**`bind: address already in use` on 5435 or 6377.** Another process (or an old Compose project) owns the port. `docker compose ps`, `make down`, or stop the other process.

**`GET /v1/ready` → 503 / `/health` page says API is not ready.** Compose is down, Postgres/Redis not healthy yet, or the API cannot reach 5435/6377. `make up` and `make logs`. Health (`/v1/health`) staying 200 while ready is 503 is expected: health does not touch Postgres.

**goose `connection refused` / auth failed.** `make up` first. DSN user/password/db must match Compose (`friendfold` / `friendfold` / `friendfold`) on host port **5435**.

**Mailpit UI is empty.** API `SMTP_HOST`/`SMTP_PORT` must be `127.0.0.1:1025`. Sign-up still succeeds if SMTP fails (mail is best-effort).

**CORS / browser fetch fails.** `WEB_ORIGIN` must be exactly the Next.js origin (`http://localhost:3000` by default).

**`sqlc-diff` fails.** Regenerated code is stale. `make sqlc` and commit `backend/internal/**/sqlc`.

**Frontend `/health` → API unreachable.** API not running, or `API_BASE_URL` not `http://localhost:8080`.

**`npm install` in `frontend/` fails / “must be installed with pnpm”.** Use Corepack pnpm 12.18.0 (`pnpm install`). Do not commit `package-lock.json`.

**MinIO console login fails / `MEDIA_DRIVER=s3` Put errors.** Host port **9000**. User/password are the local-only Compose values `friendfold` / `friendfold`. Default `MEDIA_DRIVER=fs` does not need MinIO; `make api` still works if MinIO is down.

**`make lint-backend` downloads a lot on first run.** It uses `go run` golangci-lint **v2.4.0** (same pin as CI). Later runs are cached in the module cache.
