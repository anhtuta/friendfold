# ADR 0006: 64-bit Snowflake IDs

- **Status:** Accepted
- **Date:** 2026-10-29
- **Tags:** ids, postgres, json

## Context

`BIGSERIAL` / `IDENTITY` primary keys are easy, but they:

- bottleneck on one sequence when we shard or run multiple writers,
- leak row counts,
- sort by insert order per sequence, not globally by time.

UUIDv4 is easy to generate anywhere but random, large (16 bytes vs 8), and hostile to B-tree locality. UUIDv7 is time-ordered but still 128-bit.

We need IDs that can be produced on any API instance, roughly time-ordered, and compact in indexes at 100M-user scale.

## Decision

Use **Snowflake-style 64-bit integer IDs** for all public aggregates (`users`, `posts`, `comments`, `notifications`, `reports`, media rows, friend requests).

- **Storage:** Postgres `BIGINT`, Go `int64`.
- **Generation:** in the API process (e.g. sonyflake / a small internal wrapper). Timestamp + worker ID + sequence. Worker ID comes from config (pod ordinal, or a small Redis assignment at boot).
- **JSON / OpenAPI:** IDs are **strings** (`"193848201234567890"`). JavaScript `number` cannot represent all int64 values safely (`Number.MAX_SAFE_INTEGER` is 2^53−1).
- **URLs:** prefer username for profiles (`/u/{username}`); use the string ID for posts and comments (`/p/{id}`).
- Do **not** use `SERIAL` for these tables. Tiny internal lookup tables (if any) may still use small serials; they are not public.

## Consequences

- Indexes stay 8-byte and roughly time-clustered, which helps feed and notification scans.
- Clients, TypeScript types, and `<input>` values must treat IDs as strings end-to-end. Never `parseInt` a resource ID.
- Clock skew: NTP on hosts matters; the generator must refuse to issue IDs if the clock moves backwards beyond a small tolerance.
- Sharding by user ID still uses the user Snowflake as the shard key; post IDs are not the shard key.
- Specifying IDs as JSON numbers is a bug; OpenAPI must use `type: string` with a digit pattern.

## Alternatives considered

- **Postgres `BIGINT GENERATED ALWAYS AS IDENTITY`.** Simplest; fails the multi-writer and sharding test.
- **UUIDv7.** Good locality, 2× index size, still awkward in JS if we ever treat them as anything but strings. Acceptable fallback if Snowflake worker-ID assignment becomes painful — would require a new ADR.
- **ULID / KSUID.** Similar to UUIDv7; 128-bit.
- **Twitter Snowflake as a separate ID service.** An extra network hop we do not need; embed the generator in the API.
