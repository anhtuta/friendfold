# Feature NN: Title

**Status:** Not started  
**Owner:** (team / agent filling this in)  
**Plan version:** 1

> Copy this file to `docs/features/NN-short-name.md`. Number `NN` is build order. Do not start code until this plan exists. Implement **one phase per PR**, in order, and update statuses when a phase ships.

## Goal

One paragraph: what a user (or operator) can do when this feature is done, and why it exists.

## Scope

### In scope

- Bullet list of capabilities that will ship in this feature's phases.

### Out of scope

- Explicit non-goals (including things a reader might assume). Point at a later feature if known.

## User stories

- As a **role**, I want **action**, so that **outcome**.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-nn-01 | … |

## Data model

Postgres tables this feature owns. Column types, nullability, unique constraints, and indexes. IDs are Snowflake `BIGINT` unless noted. JSON still serializes them as strings.

### `table_name`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | Snowflake |

Indexes:

- `table_name_pkey` on `(id)`
- …

Redis keys, if any: `namespace:{id}` → type, TTL, who writes/reads.

## API endpoints

All paths under `/v1`. Auth: session cookie unless marked public.

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/v1/...` | user | … |

Update `backend/api/openapi.yaml` **before** implementing each row.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `/...` | … |

## Non-functional and scaling considerations

Latency, pagination, caching, expected row growth, shard key, what happens at 100M users. Tie numbers back to [00-project-overview.md](../00-project-overview.md) NFRs.

## Security and privacy

Authz matrix, validation, rate limits, personal data collected, retention. Hidden resources return 404.

## Dependencies on other features

- Requires: …
- Enables: …

## Implementation phases

Each phase is one PR. Do not start phase N+1 until N is shipped.

### Phase 1 — title

**Status:** Not started  
**PR:** —

Work:

- …

Acceptance:

- [ ] …

### Phase 2 — title

**Status:** Not started  
**PR:** —

Work:

- …

Acceptance:

- [ ] …

## Testing plan

- Unit: …
- Handler / API: …
- Frontend: …
- Abuse / authz cases: …

## Open questions

- Question — default assumption if we must proceed: …
