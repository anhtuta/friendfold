# ADR 0003: PostgreSQL and Redis

- **Status:** Accepted
- **Date:** 2026-10-29
- **Tags:** data, postgres, redis, cache

## Context

We need a system of record for users, graphs, posts, and moderation, plus a low-latency store for sessions, hot counters, rate-limit windows, and precomputed feeds. The design must allow read replicas and, much later, sharding by user ID.

## Decision

**PostgreSQL** is the system of record. **Redis** is the working set:

| Concern | Store |
| --- | --- |
| Users, credentials, profiles, friendships, posts, comments, reports | Postgres |
| Sessions | Redis (required) |
| Rate-limit counters | Redis (required) |
| Like/comment/friend/unread counts | Redis cache, Postgres authoritative |
| Home timelines | Redis lists/ZSETs, rebuilt from Postgres on miss |
| Email one-time tokens | Postgres (hashed) + short Redis lock for single use is acceptable |

Access: pgx pool + sqlc. Separate **read** and **write** pool config from day one even if both point at the same instance locally.

When a single primary is no longer enough: add read replicas, then shard by user ID with **Citus** or a distributed Postgres-compatible database. Do not switch the system of record to a wide-column store without a new ADR.

Local dev: Postgres and Redis via Docker Compose.

## Consequences

- Relational integrity (FKs, unique usernames, exclusion of duplicate friendships) is available until we shard. After sharding, cross-shard FKs go away; modules must not rely on cross-user FKs beyond `ON DELETE` that can be replaced by application-level cleanup.
- Redis downtime logs everyone out and disables rate limits — production Redis must be highly available, or we accept that tradeoff in writing (document it in ops). Cache-shaped keys must degrade to Postgres.
- Kafka, OpenSearch, and object storage are later and do not replace Postgres.

## Alternatives considered

- **Postgres only.** Possible for a prototype; sessions and feed fan-out at scale need Redis (or equivalent).
- **MySQL.** Fine, but Postgres is the better default for this team and for Citus-class options.
- **DynamoDB / Spanner as primary.** Premature; operational and query-model cost is not justified before product-market fit.
- **MongoDB.** Weak fit for graph + privacy rules we want as constraints.
