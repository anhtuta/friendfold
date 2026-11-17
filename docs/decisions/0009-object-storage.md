# ADR 0009: Object storage behind an interface

- **Status:** Accepted
- **Date:** 2026-11-17
- **Tags:** media, storage, s3, cdn, platform

## Context

Profiles (feature 03) and later posts (feature 05) need to store user images (avatars, covers, post photos). The MVP runs on a laptop and a single API process. A path to ~100M users requires object storage plus a CDN: the API must not be the long-term origin for bytes, keys must not collide, and a bucket/CDN cutover must not rewrite rows.

Local options: a directory on the API host, or S3-compatible MinIO in Docker Compose. Production options: Amazon S3 or Cloudflare R2 in **ap-southeast-1** (see [00-project-overview.md](../00-project-overview.md)). Embedding a vendor SDK in `internal/profile` would leak storage into a product module and make a later worker split harder.

## Decision

Put a small **`ObjectStore` interface in `internal/platform/blob`** (or equivalent `internal/platform` package). Feature modules call the interface; they do not open files or speak S3.

```
Put(ctx, key, contentType, body) error
Get(ctx, key) (body, contentType, error)
Delete(ctx, key) error
```

Public URLs are **not** stored. Postgres keeps an **object key**. JSON `avatarUrl` / `coverUrl` are composed at read time as `{MEDIA_PUBLIC_BASE_URL}/{key}`. Switching local `/v1/media/...` to a CDN is an env change, not a backfill.

Keys are server-minted and unique: `profile/{userId}/{kind}/{snowflake}.jpg`. Callers never send a key. Object names are not user-controlled.

Drivers:

| Driver | When | Notes |
| --- | --- | --- |
| `fs` | Default local | Directory `MEDIA_DIR` (gitignored). `GET /v1/media/*` streams bytes. |
| `s3` | MinIO locally; S3/R2 in production | S3-compatible API (MinIO Go client is acceptable). Path-style for MinIO. |

`MEDIA_DRIVER` selects the driver. Production **must** use `s3` (or a later driver); the API must not be the durable disk for user bytes.

Image **validation and transcode** stay in the product module that accepts the upload (`internal/profile` for avatars/covers). The store persists already-trusted bytes. Do not trust client `Content-Type` or filename. DecodeConfig + magic bytes, size cap, pixel cap, re-encode (strips EXIF), then `Put`.

MinIO is added to local Compose so the `s3` driver is exercisable without AWS. Default driver remains **`fs`** so `make api` still works if MinIO is down. `GET /v1/ready` does **not** ping object storage (cache-shaped dependency; miss/failure is an upload error, not process unreadiness).

## Consequences

- Profile and post rows stay small (keys, not blobs in Postgres).
- CDN and bucket hostname changes do not require a SQL migration.
- A later transcode worker can `Get`/`Put` through the same interface.
- Local `fs` is a footgun if accidentally used in production — document `MEDIA_DRIVER=s3` as required for deploy; add a startup warning when `fs` is selected and `WEB_ORIGIN` is https.
- Orphan objects (failed DB write after `Put`) need a later GC job; `media_objects` in feature 03 is the index for that.

## Alternatives considered

- **Postgres `BYTEA`.** Fine for tests; hostile at avatar/cover scale and useless for a CDN.
- **API disk only, no interface.** Fastest MVP; a production cutover becomes a rewrite of profile and post modules.
- **Presigned upload from the browser.** Better at 100M (API never sees bytes) but pushes MIME sniffing and transcode into a worker on day one. Revisit when post photo volume justifies it; avatars are small enough to transcode in-request.
- **Vendor-specific SDK in `internal/profile`.** Rejected; the module would import AWS/R2 details and 05 would copy them.
