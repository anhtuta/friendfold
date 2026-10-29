# ADR 0004: Modular monolith

- **Status:** Accepted
- **Date:** 2026-10-29
- **Tags:** architecture, modules, grpc

## Context

A social network splits naturally into accounts, profiles, graph, posts, feed, engagement, notifications, and safety. Starting with a microservice fleet would multiply ops work before we have users. Starting with a ball of shared SQL would make a later split a rewrite.

## Decision

Ship **one Go API process** whose internal packages are feature modules:

```
backend/internal/<feature>/   # handlers, service, tests
backend/internal/platform/    # config, db, redis, http middleware, Snowflake, log
backend/cmd/api/              # main, chi mux, constructors, adapters
```

Rules:

1. A module does not import another module's store or SQL. It depends on a **small interface** it owns (e.g. `feed` needs `PostByIDs(ctx, ids)` defined in `feed`, implemented by `post` and wired in `cmd/api`).
2. A module's tables are its own. Other modules may store **IDs** of foreign aggregates, not duplicate their rows.
3. When a module becomes a service, the interface is implemented over **gRPC**. HTTP stays at the edge (BFF or gateway); internal traffic is not JSON-over-HTTP.
4. Shared kernel code lives in `internal/platform`, not in a grab-bag `internal/common`.

Feature → package mapping is in [AGENTS.md](../../AGENTS.md) and [feature 01](../features/01-project-foundation.md).

## Consequences

- We can iterate as a monolith (one binary, one deploy, one transaction when we truly need it).
- Cross-module transactions are allowed only inside a single process and only when a use case is atomic (e.g. accept friend request + write both friendship rows). After a split, those become outbox events — so keep them rare.
- Code review should reject `internal/feed` importing `internal/post/store.go`.
- gRPC stubs are not generated until the first split; do not add protobuf "just in case."

## Alternatives considered

- **Microservices from day one.** Ops-heavy, distributed transactions, worse local dev.
- **Layered monolith (`handlers/`, `services/`, `repos/` for the whole app).** Fast to start, almost impossible to extract a service cleanly.
- **Modular monolith with in-process events only.** Useful later (and with Kafka); not a substitute for module interfaces on synchronous reads.
