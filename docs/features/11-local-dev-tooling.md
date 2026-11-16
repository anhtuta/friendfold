# Feature 11: Local dev Makefile and deployment docs

**Status:** In progress (phase 1)  
**Owner:** —  
**Plan version:** 1

> Developer tooling, not an MVP product feature. Independent of 02–09 and of [10](10-implement-task-skill.md). Does not add user-facing surface.

## Goal

A developer can start the local stack from documented `make` targets, on host ports that do not collide with a typical local Postgres or Redis, and follow one local deployment guide.

## Scope

### In scope

- `docs/deployment/deployment-local.md` (full local runbook).
- Extract the **Local run** section from [01](01-project-foundation.md); leave a short link.
- Index `docs/deployment/` in [docs/README.md](../README.md). Production runbook is later (`deployment-prod.md` is not in this feature).
- Compose **host** ports: Postgres `5435→5432`, Redis `6377→6379`. Container ports stay `5432` / `6379`.
- Update every default and reference that used the old host ports.
- Root `Makefile` with `.PHONY` targets that wrap commands the repo already supports (Compose, goose v3.24.3, sqlc v1.29.0, Go API, Next.js, tests, lint). macOS and Linux.

### Out of scope

- Production deploy, Kubernetes, Terraform, Cloudflare.
- Creating `docs/deployment/deployment-prod.md`.
- Running the API or Next.js inside Compose.
- Changing Mailpit host ports (`1025` / `8025`), API `:8080`, or Next.js `3000`.
- Root `README.md` (owned separately; it has no port map today).

## User stories

- As a **developer**, I want `make help` and a single local guide, so I do not copy stale commands out of a shipped feature plan.
- As a **developer**, I want Compose Postgres/Redis on non-default host ports, so they do not fight with databases already on my machine.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-11-01 | `docs/deployment/deployment-local.md` covers prereqs, Compose services, migrations, backend, frontend, Mailpit, ports, env defaults, troubleshooting. Commands use `make` targets. |
| F-11-02 | Feature 01 **Local run** is a short link to that doc. |
| F-11-03 | `docs/README.md` lists `docs/deployment/` and notes that `deployment-prod.md` comes later. |
| F-11-04 | Compose publishes Postgres host `5435` and Redis host `6377`. Container ports remain `5432` and `6379`. |
| F-11-05 | Config defaults, `.env.example` comments, docs, tests, and goose/sqlc command examples that pointed at host `5432`/`6379` use `5435`/`6377`. CI is unchanged unless it used those host ports. |
| F-11-06 | Root `Makefile`: `help` is the default target; `up`/`down`/`logs`; `migrate` (goose v3.24.3, DSN on `5435`); `api`/`backend`; `frontend`; `dev`; `test`; `lint`; `sqlc`; `.PHONY`; works with GNU make on macOS and Linux. |

## Data model

N/A.

## API endpoints

N/A.

## Frontend pages / components

N/A.

## Non-functional and scaling considerations

Host port remap is local-only. Production bind addresses stay in env, not in Compose. Makefile does not replace CI.

## Security and privacy

- Makefile DSN uses the **local-only** Compose password already in `docker-compose.yml`.
- `.env.example` stays names-plus-comments; no production secrets.

## Dependencies on other features

- Requires: [01](01-project-foundation.md) (Compose, goose, sqlc, API, Next.js, CI pins).
- Enables: faster local work on 02–09. Does not block them.

## Implementation phases

### Phase 1 — Guide, host ports, Makefile

**Status:** In progress  
**PR:** https://github.com/anhtuta/friendfold/pull/39

Work:

- Add `docs/deployment/deployment-local.md` and index it.
- Move feature 01 local-run commands; update F-01-07 to the new host ports.
- Remap Compose host ports; update `internal/platform/config` defaults and other references.
- Add root `Makefile`.

Acceptance:

- [x] `make help` lists targets; dry-run of non-interactive targets matches the guide.
- [x] `go test ./...` (backend) and frontend checks that match CI still pass.
- [x] Relative markdown links in the new/changed docs resolve.
- [x] CI does not start using host `5432`/`6379` (it did not before).

## Testing plan

- Makefile: `make help`; `make -n` on `up`, `down`, `migrate`, `test`, `lint`, `sqlc`.
- Go: existing config default tests; URL fixtures that used host `5432`/`6379`.
- Frontend: existing unit tests (API origin is still `:8080`).
- Docs: relative links exist on disk.
- No UI screenshots (no user-visible UI).

## Open questions

- **`make sqlc` generate vs diff:** Default **generate** (local query workflow). `sqlc-diff` matches the CI `sqlc diff` job.
- **`make lint-backend` binary vs `go run`:** Default **`go run` golangci-lint@v2.4.0** so a clean machine matches CI without a global install.
