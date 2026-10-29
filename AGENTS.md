# Friendfold agent rules

This file is the source of truth for every coding agent working in this repository. Cursor project rules in `.cursor/rules/` restate the same policies with scoping. If a rule here and a `.mdc` file ever disagree, treat this file as canonical and fix the `.mdc`.

Friendfold is a general-audience social network ("Into the fold of friends."). It is a real product for real users, not a demo. Design for correctness and safety first, then for a path to ~100M users. Do not invent extra product surface (groups, stories, marketplace, messaging, a mobile app) unless a plan doc in `docs/` says to.

**Docs only until a feature plan says otherwise.** Do not add application code, configs, Dockerfiles, `go.mod`, `package.json`, or scaffolding unless you are implementing a numbered phase of a feature plan.

Do not modify root `README.md` unless the task explicitly says to; other work may already be changing it.

## (a) Plan first

1. Every feature lives in `docs/features/NN-name.md`, written from `docs/features/_TEMPLATE.md`.
2. Do not write implementation code for a feature until that plan exists and the phase you are building is specified.
3. Implement phases **in the order listed in the plan**. Do not skip ahead, combine phases, or start the next feature while the current one's required phases are open.
4. **Each pull request does exactly one concern.** Prefer several small, independently reviewable PRs over one large PR. If a phase contains more than one distinct concern (for example Docker Compose services vs. migration tooling vs. config and a ready check), split it into stacked PRs and state the merge order in each description. Tests for the behavior a PR adds ship in that PR.
5. When a phase ships, update that phase's status in the plan (and the doc's overall **Status** if the feature is fully done). Link the PR, or every PR if the phase was split.
6. If you discover the plan is wrong, update the plan in the same PR as the code, or in a docs-only PR before coding further. Record cross-cutting choices as ADRs in `docs/decisions/`.

New work starts by reading:

- [docs/00-project-overview.md](docs/00-project-overview.md)
- The relevant feature plan under [docs/features/](docs/features/)
- Accepted ADRs in [docs/decisions/](docs/decisions/)

## (b) Repo structure

Monorepo. Do not create `frontend/` or `backend/` until feature 01 (project foundation) is being implemented.

```
frontend/                     # Next.js + React + TypeScript (web only)
backend/
  cmd/api/                    # HTTP process entrypoint
  internal/<feature>/         # one module per product feature
  internal/platform/          # shared kernel: config, DB, Redis, HTTP, IDs, log
  api/openapi.yaml            # API contract (source of truth)
  db/migrations/              # goose SQL migrations
  db/queries/                 # sqlc query files
docs/                         # plans, ADRs, overview — already present
```

Backend feature modules (create when that feature is implemented):

| Module | Feature plan |
| --- | --- |
| `internal/platform` | 01 foundation |
| `internal/auth` | 02 accounts |
| `internal/profile` | 03 profiles |
| `internal/friend` | 04 friends |
| `internal/post` | 05 posts |
| `internal/feed` | 06 news feed |
| `internal/reaction` | 07 likes |
| `internal/comment` | 07 comments |
| `internal/notification` | 08 notifications |
| `internal/safety` | 09 safety |

A module owns its HTTP handlers, service layer, sqlc queries, and tests. It may import `internal/platform` and call other modules through small Go interfaces, not by reaching into another module's database tables. When a module is later split into its own service, those interfaces become gRPC.

Do not add a mobile app. Do not introduce new top-level packages (`services/`, `apps/`, `packages/`) without an ADR.

## (c) Architecture Decision Records

Record lasting technical choices in `docs/decisions/NNNN-title.md`.

Write an ADR when you choose a language, datastore, ID scheme, module boundary, feed strategy, auth mechanism, or anything expensive to reverse. Do not write ADRs for local refactors.

Accepted ADRs already in the repo:

- [0001](docs/decisions/0001-go-backend.md) — Go backend
- [0002](docs/decisions/0002-nextjs-frontend.md) — Next.js frontend
- [0003](docs/decisions/0003-postgres-redis.md) — Postgres + Redis
- [0004](docs/decisions/0004-modular-monolith.md) — modular monolith
- [0005](docs/decisions/0005-monorepo-layout.md) — monorepo layout
- [0006](docs/decisions/0006-snowflake-ids.md) — 64-bit Snowflake IDs
- [0007](docs/decisions/0007-tailwind-css.md) — Tailwind CSS
- [0008](docs/decisions/0008-dark-theme.md) — dark theme as the default look

Next number is `0009`. Status is `Proposed`, `Accepted`, `Superseded by NNNN`, or `Deprecated`.

## (d) API contract first

`backend/api/openapi.yaml` is the single source of truth for HTTP APIs.

1. Update the OpenAPI spec (paths, schemas, error responses, auth) **before** implementing an endpoint.
2. Generate server stubs / client types from the spec. Do not hand-maintain a parallel type layer that can drift.
3. JSON field names are `camelCase`. 64-bit IDs are **strings** in JSON (JavaScript cannot safely hold Snowflake IDs as numbers).
4. Version all HTTP APIs under `/v1`. Breaking changes get `/v2`; do not silently break `/v1`.
5. Document error bodies uniformly: `{ "error": { "code": "...", "message": "...", "details": {} } }`.

## (e) Schema changes are migrations

- Every table, column, index, constraint, or enum change is a goose migration in `backend/db/migrations/`.
- Never edit a migration that has been merged to `main`. Add a new one.
- Migrations are forwards-compatible: expand (add nullable column / new table) → deploy code → backfill → constrain. Do not drop or rename in the same PR that deploys readers of the old shape.
- sqlc queries go in `backend/db/queries/`. Handlers do not embed SQL.
- Indexes listed in the feature plan's data model must exist before that feature is marked shipped.

## (f) Quality

- Tests ship in the same PR as the change: Go table-driven unit tests around the service layer, HTTP handler tests with a test database or tightly scoped fakes, and frontend tests for user-visible behavior.
- **UI screenshots in the PR description.** Every pull request that changes user-visible UI (pages, layout, styling, or components) must embed screenshots of the affected pages and states in the PR description so they render on GitHub (Markdown images or HTML `<img>` tags). When changing existing UI, include **before and after**. Cover the important states you touched (happy path plus error/empty when those are visible). Docs-only, API-only, and non-visual changes are exempt.
- Lint and typecheck must be green locally; CI on GitHub Actions is the gate.
- PRs are small and focused — **one concern**, independently reviewable. A phase may be multiple PRs; do not combine unrelated concerns. If a PR is hard to review, split it.
- Do not merge with skipped tests, `as any`, or `//nolint` without a one-line justification.

## (g) Security

- No secrets in git. Configuration comes from environment variables. `.env.example` lists names only.
- Validate and bound every input (length, encoding, content type, UUID/Snowflake shape, pagination limits).
- Authorization on **every** endpoint: unauthenticated vs authenticated vs resource owner vs friend vs admin. Check block lists before returning another user's content.
- Passwords are hashed with **argon2id** (never bcrypt, never SHA). Compare in constant time.
- Sessions live in Redis; session cookies are `HttpOnly`, `Secure`, `SameSite=Lax`.
- Default-deny on privacy: if a post is `friends` or `only_me`, strangers get 404, not 403 (do not leak existence).
- Uploads: allowlisted MIME types, size caps, scan/transcode server-side, store outside the web root.

## (h) Commits and style

**Conventional Commits:** `type(scope): summary`

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`. Scopes match modules (`auth`, `profile`, `friend`, `post`, `feed`, `reaction`, `comment`, `notification`, `safety`, `platform`) or `docs`, `web`.

**Git LFS:** Binary files (images, video, fonts, and other non-text assets) must be stored with Git LFS. Track them in `.gitattributes`; do not commit raw binaries as regular git blobs.

**Go:** `gofmt` (and `goimports`). `context.Context` is the first argument. Wrap errors with `%w`. No package-level `init()` that connects to the network. Handlers decode/encode only; business rules live in the module's service. Log with structured fields (`request_id`, `user_id`, `error`), never with `fmt.Println`.

**TypeScript:** `strict` true, no `any`, no unused locals. App Router; Server Components by default; `"use client"` only when needed. Named exports. One component per file, filename matches the export.

## (i) Keep docs current

If you change behavior, endpoints, schema, or a decision, update the matching feature plan, OpenAPI spec, and ADR in the same PR. Do not leave "TODO: update docs" as follow-up. Stale docs are a defect.

## Local development (once foundation exists)

Docker Compose runs Postgres, Redis, and Mailpit. The Go API and Next.js app run on the host (or in Compose once that is wired). Never point a development workflow at a production database.
