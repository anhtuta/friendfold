# ADR 0005: Monorepo layout

- **Status:** Accepted
- **Date:** 2026-10-29
- **Tags:** repo, layout

## Context

The OpenAPI spec, Go API, Next.js app, and docs need to version together. Feature work almost always touches a plan doc, a spec, a migration, and both sides of an endpoint.

## Decision

One Git repository (`anhtuta/friendfold`) with:

```
frontend/     # Next.js app
backend/      # Go API, OpenAPI, goose, sqlc
docs/         # overview, ADRs, feature plans
AGENTS.md     # agent rules
.cursor/rules/
```

- One Go module under `backend/`.
- One Node package under `frontend/` (no npm workspaces until a second JS package exists, which needs its own ADR).
- CI in `.github/workflows/` at the root, with path filters so docs-only PRs do not build the world once those workflows exist.
- Docker Compose at the repo root for Postgres, Redis, Mailpit.

`frontend/` and `backend/` are created in [feature 01](../features/01-project-foundation.md), not before.

## Consequences

- Atomic PRs can change spec + API + UI + docs.
- Clone is simple for agents and humans.
- If a module is extracted to its own service much later, it may leave this repo (new ADR) or stay as another Go module here. Default is **stay in the monorepo** until proven otherwise.

## Alternatives considered

- **Polyrepo** (api, web, docs). Better independent versioning; worse for a small team and for contract-first work.
- **Turborepo / pnpm workspaces with many JS packages.** Unnecessary while there is one web app.
- **Backend as the repo root** (`cmd/` at top level). Rejected so the web app is a first-class sibling, not an afterthought.
