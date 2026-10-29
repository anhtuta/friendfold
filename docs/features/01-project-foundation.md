# Feature 01: Project foundation

**Status:** Not started  
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
- One CSS/styling choice documented in an ADR **or** in this plan's open questions, then used on the placeholder page.

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
| `app/layout.tsx` | Root HTML, `lang="en"`, placeholder font |

## Non-functional and scaling considerations

- Config must include **two** database URLs (`DATABASE_URL`, `DATABASE_READ_URL`) even if they are identical locally ([ADR 0003](../decisions/0003-postgres-redis.md)).
- pgx pool max connections are configurable; default small (e.g. 10) so local Docker is not exhausted.
- Health must not depend on downstreams; ready may.
- Binary should be `CGO_ENABLED=0` so later containers stay tiny.
- Do not bake secrets into images; Compose reads `.env`.

## Security and privacy

- No auth yet; health is public and must not return versions of libraries, env dumps, or connection strings.
- `.env` is gitignored; `.env.example` lists `DATABASE_URL`, `DATABASE_READ_URL`, `REDIS_URL`, `API_ADDR`, `WEB_ORIGIN`, `SNOWFLAKE_WORKER_ID`, `LOG_LEVEL`.
- Default Compose Postgres password is for **local only** and documented as such.

## Dependencies on other features

- Requires: nothing (first feature).
- Enables: 02–09.

## Implementation phases

### Phase 1 — Go API skeleton

**Status:** Not started  
**PR:** —

Work:

- Create `backend/` module, `cmd/api`, chi router, env config, slog JSON, request ID middleware, `GET /v1/health`, Snowflake helper with tests.
- `internal/platform` packages: `config`, `http`, `log`, `id`.

Acceptance:

- [ ] `go test ./...` and `go build ./cmd/api` pass.
- [ ] Hitting health returns 200 JSON.
- [ ] Logs include `request_id`.
- [ ] OpenAPI file exists with at least the health path (may be completed in phase 4, but prefer to add health now).

### Phase 2 — Compose, Postgres, Redis, goose, ready check

**Status:** Not started  
**PR:** —

Work:

- Root `docker-compose.yml`: Postgres 16, Redis 7, Mailpit.
- `backend/db/migrations` + goose; empty or comment-only first migration is acceptable if goose needs a bootstrap.
- pgx pools (write + read), Redis client, `GET /v1/ready`.
- `.env.example`, `.gitignore`.
- sqlc.yaml pointing at `db/queries` (may contain a no-op or `SELECT 1`).

Acceptance:

- [ ] `docker compose up -d` then ready returns 200.
- [ ] Stopping Postgres makes ready return 503; API process stays up.
- [ ] goose up/down works against Compose.

### Phase 3 — Next.js app

**Status:** Not started  
**PR:** —

Work:

- `frontend/` with App Router, TypeScript strict, root layout, `/` and `/health`.
- Document `WEB_ORIGIN` and `NEXT_PUBLIC_API_BASE_URL` (or server-only `API_BASE_URL`).
- Pick and document the styling approach in this phase's PR description and in this doc (update this file).

Acceptance:

- [ ] `npx tsc --noEmit` and production build succeed.
- [ ] `/` renders the name and tagline.
- [ ] `/health` shows OK when the API is ready.

### Phase 4 — OpenAPI completion and CI

**Status:** Not started  
**PR:** —

Work:

- Finish `backend/api/openapi.yaml` (error envelope, health, ready).
- GitHub Actions: backend tests + golangci-lint (or `go vet` + `gofmt` check if lint config is too much for this PR — prefer golangci-lint), frontend typecheck + build, optional Spectral/Redocly on the spec.
- README **must not** be rewritten here (owned elsewhere); put run instructions in `docs/features/01-project-foundation.md` (a "Local run" subsection added in this phase) or a `docs/dev-setup.md` if the PR wants a dedicated page — prefer a **Local run** section at the bottom of this file to avoid extra docs sprawl.

Acceptance:

- [ ] CI green on the PR.
- [ ] Spec validates.
- [ ] This plan's Status becomes **Shipped** when this phase merges; add Local run commands.

## Testing plan

- Unit: Snowflake generator (unique, worker bits).
- Handler: health 200; ready 200/503 with fake ping or testcontainers/Compose.
- CI job exists and is required.
- No frontend component tests required beyond build; `/health` can be a smoke test later.

## Open questions

- **CSS approach:** Tailwind vs CSS modules. Default if we must proceed: **Tailwind** (common with Next.js, easy for agents). Confirm in phase 3 and ADR 0007 if we want it recorded.
- **golangci-lint vs a smaller set:** Default: golangci-lint with a committed `.golangci.yml` in phase 4.
- **API proxy:** Browser calls `localhost:8080` directly in dev (CORS) vs Next.js rewrite. Default: **CORS + direct**, simpler; a rewrite can be added if cookie SameSite becomes an issue (it should not on http://localhost).
- **i18n library now vs later:** Default: English strings in phase 3; add `next-intl` when Vietnamese is scheduled, not in 01.

## Local run

Filled in when phase 4 ships. Expected shape:

```bash
docker compose up -d
# backend: go run ./cmd/api
# frontend: npm run dev
# mailpit UI: http://localhost:8025
```
