# Feature 01: Project foundation

**Status:** Shipped  
**Owner:** —  
**Plan version:** 1

## Goal

Stand up the skeleton every later feature depends on: a compilable Go API, a runnable Next.js app, Postgres/Redis/Mailpit in Docker Compose, goose + sqlc, an OpenAPI spec, structured logging, Snowflake ID plumbing, and GitHub Actions CI. Nothing here is user-facing except a health page / health JSON.

## Scope

### In scope

- Repo directories `frontend/` and `backend/` as in [ADR 0005](../decisions/0005-monorepo-layout.md).
- Go module `github.com/anhtuta/friendfold/backend` with `cmd/api` and `internal/platform` (config, pgx pools, Redis, chi middleware, slog, Snowflake).
- `GET /v1/health` (liveness) and `GET /v1/ready` (Postgres + Redis ping).
- goose migrations, first migration that enables `pgcrypto` only if needed — prefer **no extension** until a feature requires it. `schema_migrations` is goose's.
- sqlc config and a placeholder query (health can skip SQL).
- `backend/api/openapi.yaml` with the health operations and the shared error envelope.
- Next.js App Router app: root layout, English `<html lang="en">`, a `/` page that says the product name and tagline, a `/health` page that calls the API (or shows "API unreachable").
- Docker Compose: Postgres 16, Redis 7, Mailpit. Documented env vars in `.env.example` (names only).
- GitHub Actions: Go test + lint, sqlc diff (once queries exist), `next build` / `tsc`, OpenAPI syntax check. Path filters so docs-only PRs skip heavy jobs if easy.
- `.gitignore` for binaries, `node_modules`, `.env`, Next/Go build artifacts.
- One CSS/styling choice documented in an ADR **or** in this plan's open questions, then used on the placeholder page. **Decision:** Tailwind CSS ([ADR 0007](../decisions/0007-tailwind-css.md)).

### Out of scope

- Any product feature (auth, profiles, …).
- Kubernetes, Terraform, Cloudflare, Kafka, OpenSearch, real S3.
- Production-grade observability (OTel collector); a `request_id` middleware is enough.
- Choosing a component library beyond "we need one consistent styling method."

## User stories

- As a **developer**, I want `docker compose up` plus two commands to run API and web, so that I can work on feature 02 without inventing tooling.
- As a **developer**, I want CI to fail on a broken build, so that main stays green.
- As an **operator**, I want `/v1/health` and `/v1/ready`, so that an orchestrator can restart a bad process later.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-01-01 | `go test ./...` and `go build ./cmd/api` succeed from `backend/`. |
| F-01-02 | API listens on a configurable `API_ADDR` (default `:8080`), loads config from env, and logs JSON to stdout. |
| F-01-03 | `GET /v1/health` returns `200 {"status":"ok"}` without touching Postgres. |
| F-01-04 | `GET /v1/ready` returns `200` when Postgres and Redis ping succeed, `503` otherwise. |
| F-01-05 | goose can apply migrations against Compose Postgres; a `platform` ping is enough (no product tables yet). |
| F-01-06 | Next.js starts on a configurable port (default `3000`), TypeScript `strict` true. |
| F-01-07 | Compose publishes Postgres `5432`, Redis `6379`, Mailpit SMTP `1025` and UI `8025`. |
| F-01-08 | OpenAPI 3.1 spec validates and includes health, ready, and the error schema. |
| F-01-09 | CI runs on PRs and on `main`. |
| F-01-10 | `internal/platform/id` (or equivalent) generates Snowflake IDs; covered by a unit test (monotonic in-process, worker ID honored). |
| F-01-11 | Every request gets `X-Request-Id` (generated or echoed) and that id is in logs. |

## Data model

No product tables. Goose's version table only.

Redis: none required except the ready-check `PING`.

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/v1/health` | public | Process liveness |
| `GET` | `/v1/ready` | public | Postgres + Redis reachability |

CORS: allow the Next.js origin from env (`WEB_ORIGIN`). Other origins denied.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `/` | Product name, tagline, link to sign-in once feature 02 exists (plain text until then) |
| `/health` | Dev-only status of the API ready check |
| `app/layout.tsx` | Root HTML, `lang="en"`, Inter, dark theme tokens on `html`/`body` |

## Non-functional and scaling considerations

- Config must include **two** database URLs (`DATABASE_URL`, `DATABASE_READ_URL`) even if they are identical locally ([ADR 0003](../decisions/0003-postgres-redis.md)).
- pgx pool max connections are configurable; default small (e.g. 10) so local Docker is not exhausted.
- Health must not depend on downstreams; ready may.
- Binary should be `CGO_ENABLED=0` so later containers stay tiny.
- Do not bake secrets into images; Compose reads `.env`.

## Security and privacy

- No auth yet; health is public and must not return versions of libraries, env dumps, or connection strings.
- `.env` is gitignored; `.env.example` lists `DATABASE_URL`, `DATABASE_READ_URL`, `REDIS_URL`, `API_ADDR`, `WEB_ORIGIN`, `API_BASE_URL`, `SNOWFLAKE_WORKER_ID`, `LOG_LEVEL`, `PGX_MAX_CONNS`.
- Default Compose Postgres password is for **local only** and documented as such.

## Dependencies on other features

- Requires: nothing (first feature).
- Enables: 02–09.

## Implementation phases

### Phase 1 — Go API skeleton

**Status:** Shipped  
**PR:** https://github.com/anhtuta/friendfold/pull/3

Work:

- Create `backend/` module, `cmd/api`, chi router, env config, slog JSON, request ID middleware, `GET /v1/health`, Snowflake helper with tests.
- `internal/platform` packages: `config`, `http`, `log`, `id`.

Acceptance:

- [x] `go test ./...` and `go build ./cmd/api` pass.
- [x] Hitting health returns 200 JSON.
- [x] Logs include `request_id`.
- [x] OpenAPI file exists with at least the health path (may be completed in phase 4, but prefer to add health now).

Local run (phase 1 only; Compose and the web app come in later phases):

```bash
cd backend
go test ./...
CGO_ENABLED=0 go build -o /tmp/friendfold-api ./cmd/api
API_ADDR=:8080 go run ./cmd/api
# GET http://localhost:8080/v1/health  →  {"status":"ok"}
```

Env vars used now: `API_ADDR` (default `:8080`), `LOG_LEVEL` (`debug`/`info`/`warn`/`error`, default `info`), `SNOWFLAKE_WORKER_ID` (0–1023, default `1`), `WEB_ORIGIN` (default `http://localhost:3000`). Database and Redis URLs wait for phase 2.

Decisions made while implementing (plan was slightly underspecified):

- Snowflake IDs are generated by a small internal Twitter-style helper (41-bit ms since 2026-01-01 UTC, 10-bit worker, 12-bit sequence), not sonyflake. Worker ID is taken from config. If the wall clock jumps backwards by more than 5ms versus the last issued ID, `Next` returns an error; smaller skew waits.
- Package directory `internal/platform/http` uses Go package name `httpx` so it can import `net/http`.
- CORS is enforced on the chi router now (allow `WEB_ORIGIN` only; methods GET/HEAD/OPTIONS). The Next.js app is still phase 3; this matches the feature-level origin policy so health is callable from a browser later without a router rewrite.
- OpenAPI 3.1 includes `GET /v1/health` and the shared error envelope. `GET /v1/ready` and stub/client codegen wait for later phases (ready in phase 2, spec/CI finish in phase 4). The health handler is handwritten to match the spec.

### Phase 2 — Compose, Postgres, Redis, goose, ready check

**Status:** Shipped  
**PR:** split into three stacked PRs (one concern each):

1. https://github.com/anhtuta/friendfold/pull/5 — Compose (Postgres 16, Redis 7, Mailpit), `.env.example`, `.gitignore`
2. https://github.com/anhtuta/friendfold/pull/6 — goose bootstrap migration + sqlc config
3. https://github.com/anhtuta/friendfold/pull/7 — pgx/Redis pools + `GET /v1/ready`

Work:

- Root `docker-compose.yml`: Postgres 16, Redis 7, Mailpit.
- `backend/db/migrations` + goose; empty or comment-only first migration is acceptable if goose needs a bootstrap.
- pgx pools (write + read), Redis client, `GET /v1/ready`.
- `.env.example`, `.gitignore`.
- sqlc.yaml pointing at `db/queries` (may contain a no-op or `SELECT 1`).

Acceptance:

- [x] `docker compose up -d` then ready returns 200.
- [x] Stopping Postgres makes ready return 503; API process stays up.
- [x] goose up/down works against Compose.

Local run (through phase 2; Next.js is phase 3):

```bash
docker compose up -d
cd backend
go run github.com/pressly/goose/v3/cmd/goose@v3.24.3 -dir db/migrations postgres \
  "postgres://friendfold:friendfold@127.0.0.1:5432/friendfold?sslmode=disable" up
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go build -o /tmp/friendfold-api ./cmd/api
API_ADDR=:8080 go run ./cmd/api
# GET http://localhost:8080/v1/health  →  {"status":"ok"}
# GET http://localhost:8080/v1/ready   →  {"status":"ok"} when Compose is up
# mailpit UI: http://localhost:8025
```

`DATABASE_URL`, `DATABASE_READ_URL`, and `REDIS_URL` default to the local Compose endpoints so `go run ./cmd/api` works without a filled `.env`. Copy `.env.example` to `.env` when you need to override them. The Compose Postgres password is local-only.

Decisions / deviations:

- Phase 2 is three PRs, not one, matching the one-concern-per-PR rule.
- Goose's version table is `goose_db_version` (tool default), not `schema_migrations`.
- `DATABASE_READ_URL` falls back to `DATABASE_URL` when unset. `PGX_MAX_CONNS` is configurable (default 10, range 1–100) and listed in `.env.example`.
- Ready 503 body is `{"status":"unavailable"}` with no dependency names or URLs (those go in logs with `request_id`). Ping timeout is 2s.
- sqlc Go for `SELECT 1` is generated under `internal/platform/db/sqlc` (pgx/v5). Ready uses pool/client `Ping`, not that query.
- goose v3.24.3 and sqlc v1.29.0 are run via `go run github.com/...@version` (they want Go ≥ 1.23; the Go 1.22 module toolchain auto-selects a newer compiler for those tools only).
- Redis helper lives in `internal/platform/redis` with package name `redisx` so it can wrap `github.com/redis/go-redis/v9`.

### Phase 3 — Next.js app

**Status:** Shipped  
**PR:** split into four stacked PRs (one concern each); merge in this order:

1. https://github.com/anhtuta/friendfold/pull/8 — Tailwind CSS ([ADR 0007](../decisions/0007-tailwind-css.md))
2. https://github.com/anhtuta/friendfold/pull/9 — Next.js App Router scaffold
3. https://github.com/anhtuta/friendfold/pull/10 — home page (`/` name + tagline)
4. https://github.com/anhtuta/friendfold/pull/11 — `/health` page + `API_BASE_URL` (this PR)

Work:

- `frontend/` with App Router, TypeScript strict, root layout, `/` and `/health`.
- Document `WEB_ORIGIN` and `NEXT_PUBLIC_API_BASE_URL` (or server-only `API_BASE_URL`).
- Pick and document the styling approach in this phase's PR description and in this doc (update this file).

Acceptance:

- [x] `npx tsc --noEmit` and production build succeed.
- [x] `/` renders the name and tagline.
- [x] `/health` shows OK when the API is ready.

Local run (through phase 3; CI is phase 4):

```bash
docker compose up -d
cd backend
go run github.com/pressly/goose/v3/cmd/goose@v3.24.3 -dir db/migrations postgres \
  "postgres://friendfold:friendfold@127.0.0.1:5432/friendfold?sslmode=disable" up
API_ADDR=:8080 go run ./cmd/api
# other terminal:
cd frontend
npm install
npm run dev
# http://localhost:3000       → Friendfold / Into the fold of friends.
# http://localhost:3000/health → OK when the API ready check succeeds
```

`WEB_ORIGIN` (API CORS allowlist) defaults to `http://localhost:3000`. `API_BASE_URL` (Next.js server → Go API) defaults to `http://localhost:8080`. Set either in `.env` / `frontend/.env.local` only to override. `PORT` is Next.js's usual process env (default `3000`).

Decisions / deviations:

- Phase 3 is four PRs, not one, matching the one-concern-per-PR rule.
- Styling is Tailwind CSS v4 ([ADR 0007](../decisions/0007-tailwind-css.md)). The default look is a dark theme via `@theme` tokens in `frontend/app/globals.css` ([ADR 0008](../decisions/0008-dark-theme.md)).
- Next.js **16.4** (current `create-next-app`) with App Router. The empty template turns on Cache Components; we left it **off** so placeholder routes do not need `"use cache"` / extra Suspense.
- Root layout uses Inter (`next/font/google`) as the placeholder font. `app/layout.tsx` and `app/**/page.tsx` use Next's required default export; other modules use named exports.
- English copy lives in `frontend/lib/copy.ts`. `next-intl` waits until Vietnamese is scheduled.
- `/health` is a Server Component that calls `GET {API_BASE_URL}/v1/ready` (server-only `API_BASE_URL`, not `NEXT_PUBLIC_API_BASE_URL`). Browser CORS + direct API calls remain the default for later client components (`WEB_ORIGIN`).
- Labels: ready 200 → `OK`; 503 → `API is not ready`; network/other → `API unreachable`. No connection strings in the page.
- Vitest + Testing Library cover the home page and the three `/health` states even though the original testing plan allowed build-only.

### Phase 4 — OpenAPI completion and CI

**Status:** Shipped  
**PR:** split into five stacked PRs (one concern each); merge in this order:

1. https://github.com/anhtuta/friendfold/pull/16 — backend GitHub Actions (`go test` / `go build`, golangci-lint, sqlc diff)
2. https://github.com/anhtuta/friendfold/pull/14 — frontend GitHub Actions (`tsc`, Vitest, ESLint, `next build`)
3. https://github.com/anhtuta/friendfold/pull/17 — OpenAPI lint in GitHub Actions (Redocly)
4. https://github.com/anhtuta/friendfold/pull/15 — complete `backend/api/openapi.yaml` (health, ready, error envelope)
5. https://github.com/anhtuta/friendfold/pull/18 — this PR (Local run + mark feature shipped)

Work:

- Finish `backend/api/openapi.yaml` (error envelope, health, ready).
- GitHub Actions: backend tests + golangci-lint (or `go vet` + `gofmt` check if lint config is too much for this PR — prefer golangci-lint), frontend typecheck + build, optional Spectral/Redocly on the spec.
- README **must not** be rewritten here (owned elsewhere); put run instructions in `docs/features/01-project-foundation.md` (a "Local run" subsection added in this phase) or a `docs/dev-setup.md` if the PR wants a dedicated page — prefer a **Local run** section at the bottom of this file to avoid extra docs sprawl.

Acceptance:

- [x] CI green on the PR.
- [x] Spec validates.
- [x] This plan's Status becomes **Shipped** when this phase merges; add Local run commands.

Local run (whole stack, including CI checks): see **Local run** at the bottom of this file.

Decisions / deviations:

- Phase 4 is five PRs, not one, matching the one-concern-per-PR rule.
- golangci-lint **v2.4.0** with committed `backend/.golangci.yml` (standard set plus `errorlint` and `contextcheck`; gofmt/goimports as formatters). CI uses Go **1.24.x** even though `go.mod` is 1.22, because sqlc 1.29.0 and golangci-lint v2 need a newer compiler.
- OpenAPI lint is **Redocly CLI 1.34.5**, not Spectral (better OpenAPI 3.1 support). `redocly.yaml` turns off `info-license`, `no-server-example.com`, `operation-4xx-response`, and `no-unused-components`: there is no product license yet, the only server is localhost, health/ready have no 4xx, and `ErrorEnvelope` / `cookieAuth` are declared for later features.
- `cookieAuth` is the `friendfold_session` cookie from [02](02-accounts.md). Ready 503 stays `{"status":"unavailable"}`, not the error envelope.
- Frontend CI also runs Vitest and ESLint (scripts from phase 3). `npm run typecheck` is `next typegen && tsc --noEmit` so App Router types exist on a clean checkout.
- Path filters were omitted. GitHub required checks stay pending when a path-filtered workflow never runs; these jobs are short enough to run on every PR, including docs-only.
- Generated OpenAPI client/server stubs are **not** in this phase. Health/ready bodies are tiny literals that already match the spec; codegen waits until a product endpoint needs shared models. AGENTS.md still requires generating types before those endpoints.
- No Git LFS checkout in CI: no job reads binary assets. PR 13 owns LFS tracking; this phase does not duplicate it.
- Root `README.md` was not edited.

## Testing plan

- Unit: Snowflake generator (unique, worker bits).
- Handler: health 200; ready 200/503 with fake ping or testcontainers/Compose.
- CI jobs exist on pull requests and on `main`: backend (`test`, `lint`, `sqlc`), frontend (`typecheck-test-build`), OpenAPI (`lint`). Mark them required in branch protection when that is configured.
- Frontend (phase 3): Vitest + Testing Library for `/` (name, tagline, sign-in is not a link) and `/health` (`OK` / not ready / unreachable). Build + `tsc --noEmit` still required.

## Open questions

- **CSS approach:** Confirmed **Tailwind CSS** in phase 3 ([ADR 0007](../decisions/0007-tailwind-css.md)). CSS modules and CSS-in-JS are out unless a later ADR supersedes this.
- **golangci-lint vs a smaller set:** Confirmed **golangci-lint** v2 with committed `backend/.golangci.yml` in phase 4.
- **API proxy:** Confirmed **CORS + direct** for browser calls later (`WEB_ORIGIN`). The phase 3 `/health` page uses a server-side fetch with `API_BASE_URL` because it is a Server Component; no Next.js rewrite. A rewrite can be added if cookie SameSite becomes an issue (it should not on http://localhost).
- **i18n library now vs later:** Confirmed English strings in `frontend/lib/copy.ts` in phase 3; add `next-intl` when Vietnamese is scheduled, not in 01.

## Local run

Copy `.env.example` to `.env` only when you need to override defaults. Compose Postgres password is local-only.

```bash
docker compose up -d

cd backend
go run github.com/pressly/goose/v3/cmd/goose@v3.24.3 -dir db/migrations postgres \
  "postgres://friendfold:friendfold@127.0.0.1:5432/friendfold?sslmode=disable" up
CGO_ENABLED=0 go test ./...
golangci-lint run --config .golangci.yml ./...
GOTOOLCHAIN=auto go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 diff
CGO_ENABLED=0 go build -o /tmp/friendfold-api ./cmd/api
API_ADDR=:8080 go run ./cmd/api
# GET http://localhost:8080/v1/health  →  {"status":"ok"}
# GET http://localhost:8080/v1/ready   →  {"status":"ok"} when Compose is up
# mailpit UI: http://localhost:8025

# other terminal:
cd frontend
npm install
npm run typecheck
npm test
npm run lint
npm run build
npm run dev
# http://localhost:3000       → Friendfold / Into the fold of friends.
# http://localhost:3000/health → OK when the API ready check succeeds

# from repo root:
npx --yes @redocly/cli@1.34.5 lint --config redocly.yaml
```

`WEB_ORIGIN` (API CORS allowlist) defaults to `http://localhost:3000`. `API_BASE_URL` (Next.js server → Go API) defaults to `http://localhost:8080`. `PORT` is Next.js's usual process env (default `3000`). `DATABASE_URL`, `DATABASE_READ_URL`, and `REDIS_URL` default to the local Compose endpoints.
