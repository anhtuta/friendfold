# ADR 0001: Go backend

- **Status:** Accepted
- **Date:** 2026-10-29
- **Tags:** backend, language, http, sql

## Context

Friendfold needs a server that owns accounts, the social graph, posts, feeds, and enforcement of privacy. It must start as a single deployable and later split into services without a rewrite. Expected long-term load is high (path to ~100M users): many small RPCs, lots of concurrent connections, and a lot of Postgres and Redis I/O.

Candidates: Go, Java/Kotlin, Node.js (TypeScript), Python, Rust.

## Decision

Implement the API in **Go** as a modular monolith:

- HTTP: [chi](https://github.com/go-chi/chi)
- Postgres: [pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev)
- Migrations: [goose](https://github.com/pressly/goose)
- Logging: structured `log/slog`
- Later inter-service: **gRPC** (not HTTP) once a module is split out

The Go module path is `github.com/anhtuta/friendfold/backend`. Process entrypoint: `backend/cmd/api`.

## Consequences

- Strong fit for long-lived servers, goroutine concurrency, and small static binaries in Kubernetes.
- sqlc gives type-safe SQL without an ORM, which keeps queries explicit and shard-friendly.
- Hiring and agent/codegen quality for Go + chi + pgx is good.
- We accept that CPU-heavy image transcoding should be a separate worker (later), not in-request Go code beyond a thin enqueue.
- We will not introduce a second backend language. Workers, when added, are also Go.

## Alternatives considered

- **TypeScript / Node.** One language with the frontend, but weaker isolation around CPU and a less natural path to gRPC microservices at this scale.
- **Java/Kotlin.** Excellent at this scale; heavier local toolchain and slower iteration for a small team.
- **Rust.** Great runtime, slower product iteration and a smaller hiring pool.
- **Python.** Fine for early prototypes; not the long-term API runtime we want for this load shape.
