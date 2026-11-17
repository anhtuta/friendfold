# Feature 03: Profiles

**Status:** In progress (phase 1)  
**Owner:** —  
**Plan version:** 2

> Plan-only until the stacked implementation PRs land. Accounts **phase 4 (Google)** and **phase 5 (deletion)** stay paused; this feature must not implement them. Touch `internal/auth` only for a small `UpdateDisplayName` integration (display name already lives on `users`).

## Goal

Every account has a profile people can recognize: a unique username, display name, bio, avatar, cover photo, and a public page at `/u/{username}`. The owner can view and edit their own profile. Visitors can view others' public profiles. Posts on the profile wait for [05](05-posts.md); this feature ships the **page shell**.

## Scope

### In scope

- Unique **username** (handle). Same identifier in the URL, API, and `@username` copy — not a second handle namespace.
- Display name (already collected at sign-up; editable here).
- Bio (plain text, ≤ 160 runes).
- Avatar and cover upload, replace, and delete.
- `GET` / `PATCH` own profile; public `GET` by username.
- Profile page `/u/{username}` (own and others); settings edit; first-time username onboarding.
- Object storage behind `platform` (`ObjectStore`), local-dev friendly, CDN-ready URLs.
- Default avatar in the UI when none is uploaded (initials, not a stock photo).
- Block-aware public GET via an interface; **no-op** until [04](04-friends.md).

### Out of scope

- Featured photos, albums, work/education, pronouns as a structured field.
- Profile completeness gamification, profile themes, pinned posts.
- Serving posts on the profile (shell only; timeline is 05).
- Followers / following (Friendfold is friends, not a follow graph).
- `profile_privacy = friends` (MVP profiles are **public**; post privacy is separate).
- Presigned browser-to-bucket uploads (revisit in 05 if photo volume needs it).
- Google sign-in, password reset, or account deletion (02 phases 4–5).
- Production CDN, Cloudflare Image Resizing, or an async transcode worker.

## User stories

- As a **new user**, I want to pick a username and photo, so that friends can find me.
- As a **user**, I want to edit my display name, bio, avatar, and cover, so that my page looks like me.
- As a **visitor**, I want to open `/u/{username}` and see who they are (name, bio, photos).
- As an **owner**, I want to see my email on my own profile payload, so that I can confirm the account, without that email appearing to anyone else.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-03-01 | Username: 3–20 characters, `[a-z0-9_]`, stored lowercase, unique among rows with a non-null username. Lookup is case-insensitive (normalize then query). |
| F-03-02 | Reserved usernames (exact, lowercase): `admin`, `support`, `help`, `friendfold`, `settings`, `setting`, `api`, `u`, `me`, `login`, `logout`, `signup`, `register`, `health`, `ready`, `onboarding`, `username`, `profile`, `profiles`, `account`, `accounts`, `auth`, `oauth`, `google`, `password`, `reset`, `verify`, `email`, `null`, `undefined`, `www`, `mail`, `static`, `assets`, `media`, `about`, `privacy`, `terms`, `legal`, `safety`, `moderation`, `mod`, `root`, `system`, `status`. Reject with 400 `username_reserved`. |
| F-03-03 | Username is **changeable** with a **14-day cooldown** after each successful set (including the first claim). A no-op PATCH of the same username does not refresh the cooldown. Too soon → 409 `username_cooldown` (not 429; 429 is the Redis limiter). |
| F-03-04 | Signed-in users with a null username are sent to `/onboarding/username` before `/`, `/settings/*`, and `/change-password`. `/login`, `/signup`, `/verify-email`, `/forgot-password`, `/reset-password`, `/health`, `/onboarding/username`, and `/u/{username}` stay reachable. |
| F-03-05 | `PATCH /v1/me/profile` updates `displayName`, `bio`, and/or `username`. Unknown JSON fields are rejected. Avatar and cover are **not** in this body. |
| F-03-06 | Display name: trim, 1–80 Unicode runes, no control characters (same bound as register). Empty after trim → 400 `validation_error`. Auth `GET /v1/auth/me` stays in sync via an `Account` interface (`UpdateDisplayName`). |
| F-03-07 | Bio: trim, 0–160 Unicode runes, no control characters (including newlines). Empty bio is allowed and stored as `''`. |
| F-03-08 | `GET /v1/me/profile` returns the owner payload: public fields plus `email`, `emailVerified`, and `usernameChangedAt` (nullable RFC 3339). Creates a `profiles` row if missing (lazy). |
| F-03-09 | `GET /v1/users/{username}` returns the **public** profile. 404 if missing, owner account `deleted_at` is set, or the viewer is blocked either way (once 04 implements `BlockChecker`). **Never 403** for those cases. |
| F-03-10 | Public JSON **never** includes `email`, `emailVerified`, `dateOfBirth`, or `usernameChangedAt`. |
| F-03-11 | Avatars ≤ 5 MiB, covers ≤ 10 MiB, types JPEG / PNG / WebP only. GIF, PDF, SVG, BMP, and HTML are rejected. Do **not** trust `Content-Type` or the filename; sniff magic bytes and `image.DecodeConfig`. |
| F-03-12 | Server transcodes to JPEG (quality 85), strips EXIF (including GPS) by re-encoding, resizes so the long edge is at most **1024** (avatar) or **2048** (cover). Pixel bomb: reject if `width * height > 40_000_000` or either edge > 8192 **before** full decode. |
| F-03-13 | `POST /v1/me/profile/avatar` and `.../cover` are `multipart/form-data` with field `file`. Success **200** returns the updated `MeProfile`. `DELETE` of avatar or cover **200** returns `MeProfile` with that URL null. |
| F-03-14 | Default avatar is **UI-only** (initials from display name) when `avatarUrl` is null. No stock image is stored. Cover absence is a solid empty banner. |
| F-03-15 | `alt` on avatar and cover uses the display name (cover: `{displayName}'s cover photo`). Initials avatar has an accessible name. |
| F-03-16 | Rate limits (Redis sliding window, 15 min, **fail-open** like auth): PATCH 30/user; avatar POST 10/user; cover POST 10/user; public GET 120/IP. |

## Data model

`users` stays owned by [02](02-accounts.md). Profile does **not** `SELECT` from `users`. Display name, email, `email_verified_at`, and `deleted_at` are read/written through a caller-owned `Account` interface implemented by auth in `cmd/api`.

A dedicated `profiles` table keeps the module boundary ([ADR 0004](../decisions/0004-modular-monolith.md)). This replaces the v1 sketch that added columns onto `users`.

### `profiles`

| Column | Type | Notes |
| --- | --- | --- |
| `user_id` | `BIGINT PK` | Same Snowflake as `users.id` (shard key). Not a second ID. |
| `username` | `TEXT NULL` | Lowercase; NULL until onboarding. |
| `username_changed_at` | `timestamptz NULL` | Set on every successful username write. |
| `bio` | `TEXT NOT NULL DEFAULT ''` | Max 160 runes enforced in the service (CHECK char_length 160 is bytes-ish; do **not** rely on it for Unicode — validate in Go). CHECK `char_length(bio) <= 640` as a backstop. |
| `avatar_object_key` | `TEXT NULL` | Store key only; no persisted URL. |
| `cover_object_key` | `TEXT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |
| `updated_at` | `timestamptz NOT NULL` | |

Indexes:

- PK `(user_id)`
- unique `profiles_username_key` on `(username) WHERE username IS NOT NULL`

Optional FK `user_id → users(id) ON DELETE CASCADE` is allowed until we shard; it is not a cross-module SELECT. Soft-delete in 02 phase 5 must **null `username`** (or delete the profile row) so the unique index does not trap the handle; record that hook in 02 when deletion is implemented — do not build deletion here.

### `media_objects`

Owned by this module. Feature 05 uses `ObjectStore` and its own `post_media` table, **not** this table (do not add a `post` kind).

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | Snowflake |
| `user_id` | `BIGINT NOT NULL` | Uploader (shard key) |
| `kind` | `TEXT NOT NULL` | `avatar` \| `cover` (`CHECK`) |
| `object_key` | `TEXT NOT NULL UNIQUE` | |
| `mime` | `TEXT NOT NULL` | Stored type after transcode (`image/jpeg`) |
| `bytes` | `INT NOT NULL` | Encoded size |
| `width` | `INT NOT NULL` | |
| `height` | `INT NOT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- PK `(id)`
- `(user_id, created_at DESC)`
- unique `(object_key)`

Replacing an avatar `Put`s a new key, updates `profiles.avatar_object_key`, inserts `media_objects`, then `Delete`s the previous key (best-effort; failure is logged, not a 500). Orphans are GC later.

### Redis

| Key | Value | TTL | Notes |
| --- | --- | --- | --- |
| `profile:pub:{username}` | JSON public profile | 5 min | Invalidate on PATCH / upload / delete. Miss → Postgres. |
| `profile:uid:{userId}` | username string | 5 min | Lets invalidation find the public key after a rename. |
| `rate:profile:patch:{userId}` | sliding window | 15 min | |
| `rate:profile:avatar:{userId}` | sliding window | 15 min | |
| `rate:profile:cover:{userId}` | sliding window | 15 min | |
| `rate:profile:get:{ip}` | sliding window | 15 min | Public GET only |

Do **not** cache `MeProfile` (contains email). Cache failures fall back to Postgres (same policy as “Redis cache is optional” in [ADR 0003](../decisions/0003-postgres-redis.md)). Rate-limit Redis errors fail **open**.

## API endpoints

All paths under `/v1`. JSON `camelCase`. IDs are decimal strings. Mutating `/v1/me/profile*` require `Origin` matching `WEB_ORIGIN` and `cookieAuth`. Precise schemas live in `backend/api/openapi.yaml` (contract first). Examples use obviously fake values (`minh`, `minh@example.com`, `example-object`, `193848201234567890`) — never high-entropy tokens.

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/v1/me/profile` | user | Owner profile; lazy-create row |
| `PATCH` | `/v1/me/profile` | user | `displayName`, `bio`, `username` |
| `POST` | `/v1/me/profile/avatar` | user | multipart `file` |
| `DELETE` | `/v1/me/profile/avatar` | user | Clear avatar |
| `POST` | `/v1/me/profile/cover` | user | multipart `file` |
| `DELETE` | `/v1/me/profile/cover` | user | Clear cover |
| `GET` | `/v1/users/{username}` | optional | Public profile |
| `GET` | `/v1/media/{key}` | public | Stream stored bytes (local `fs`, and `s3` when `MEDIA_PUBLIC_BASE_URL` points at the API). Production CDN URLs do not hit this. |

### Schemas (logical)

`PublicProfile`: `id`, `username`, `displayName`, `bio`, `avatarUrl` (nullable string), `coverUrl` (nullable string).

`MeProfile`: `PublicProfile` plus `email`, `emailVerified`, `usernameChangedAt` (nullable). `username` is nullable on `MeProfile` only (pre-onboarding).

`PatchProfileRequest`: optional `displayName`, `bio`, `username`; `additionalProperties: false`. At least one field required.

### Errors

| Status | Code | When |
| --- | --- | --- |
| 400 | `validation_error` | Bad JSON, empty patch, bad display name/bio, missing multipart field |
| 400 | `username_invalid` | Length or charset |
| 400 | `username_reserved` | Reserved list |
| 400 | `invalid_image` | Type, corrupt, pixel cap, decode failure |
| 401 | `unauthenticated` | Missing/invalid session on `/v1/me/*` |
| 403 | `origin_mismatch` | Mutating call, Origin ≠ `WEB_ORIGIN` |
| 404 | `not_found` | Public GET: unknown, deleted, or blocked |
| 409 | `username_taken` | Unique conflict |
| 409 | `username_cooldown` | 14-day rule |
| 413 | `payload_too_large` | Oversize body / file |
| 429 | `rate_limited` | Redis window; `Retry-After` |

CORS: allow `GET, HEAD, POST, PATCH, DELETE, OPTIONS` from `WEB_ORIGIN` (today POST-only). Credentials stay true.

## Image upload and storage

Lasting choice: [ADR 0009](../decisions/0009-object-storage.md).

### Interface (`internal/platform/blob`)

```
Put(ctx, key, contentType string, r io.Reader) error
Get(ctx, key string) (io.ReadCloser, contentType string, err error)
Delete(ctx, key string) error
```

`ErrNotFound` on missing Get/Delete. Keys are validated (`^[a-z0-9][a-z0-9/_-]*\.(jpg)$`, no `..`, no leading `/`) before any driver sees them.

URL composition (not on the store): `publicURL = strings.TrimRight(MEDIA_PUBLIC_BASE_URL, "/") + "/" + key`.

### Key layout

```
profile/{userId}/{kind}/{snowflake}.jpg
```

`kind` is `avatar` or `cover`. Snowflake from `internal/platform/id`. One object per upload. **No extra thumbnail object in MVP** — long-edge resize is enough; a CDN can derive thumbs later without a schema change because JSON URLs are opaque.

### Drivers

| `MEDIA_DRIVER` | Default | Behavior |
| --- | --- | --- |
| `fs` | **yes (local)** | Files under `MEDIA_DIR` (default `data/media`, gitignored). `Get` reads the file. |
| `s3` | production; optional local | S3-compatible (MinIO / R2 / AWS). Path-style. Create bucket on startup if missing. |

Env (names in `.env.example`, values not committed):

| Variable | Default (local) | Notes |
| --- | --- | --- |
| `MEDIA_DRIVER` | `fs` | `fs` or `s3` |
| `MEDIA_DIR` | `data/media` | `fs` only; relative to process cwd |
| `MEDIA_PUBLIC_BASE_URL` | `http://localhost:8080/v1/media` | No trailing slash; CDN in prod |
| `MEDIA_S3_ENDPOINT` | `http://127.0.0.1:9000` | MinIO host port |
| `MEDIA_S3_REGION` | `us-east-1` | Required by the SDK; unused by MinIO |
| `MEDIA_S3_BUCKET` | `friendfold-media` | |
| `MEDIA_S3_ACCESS_KEY` | empty → local `friendfold` | Local-only default, same idea as Compose Postgres |
| `MEDIA_S3_SECRET_KEY` | empty → local `friendfold` | |
| `MEDIA_S3_USE_SSL` | `false` | |
| `MEDIA_S3_PATH_STYLE` | `true` | MinIO |

Startup: if `MEDIA_DRIVER=fs` and `WEB_ORIGIN` is https, log a warning (not fatal) that production must use `s3`.

MinIO in Compose (phase 3): API **9000**, console **9001**, root user/password **local-only** `friendfold`/`friendfold`. `make up` starts it. Default driver stays `fs` so a down MinIO does not break `make api`. Document `MEDIA_DRIVER=s3` for the production-shaped path.

`GET /v1/ready` does not ping MinIO or the disk.

### Processing pipeline (in `internal/profile`, in-request)

1. `MaxBytesReader` at 5 MiB (avatar) or 10 MiB (cover) + small multipart overhead; oversize → 413.
2. Read into a bounded buffer. Sniff `http.DetectContentType` on the first 512 bytes; allow only `image/jpeg`, `image/png`, `image/webp`.
3. `image.DecodeConfig` for dimensions; enforce pixel/edge caps; then `image.Decode` with `image`, `image/jpeg`, `image/png`, and `golang.org/x/image/webp`.
4. Resize with `golang.org/x/image/draw` (CatmullRom) if long edge exceeds the cap; never upscale.
5. Encode JPEG quality 85 (stdlib). Re-encode **always**, even when the input is already JPEG — that is the EXIF/GPS strip.
6. Mint key, `Put`, write DB, delete previous key.

p95 target for upload (excluding client transfer) is 1 s ([00](../00-project-overview.md)). If in-request transcode exceeds that under load, move to a worker in a **new ADR**; do not silently add a queue in this feature.

No `disintegration/imaging` / ImageMagick. Stdlib + `golang.org/x/image` only besides the S3 client.

## Frontend pages / components

Dark theme tokens only (`background`, `foreground`, `muted`, `subtle`, `accent`, `surface`, `border`, `danger`) — [ADR 0008](../decisions/0008-dark-theme.md). Copy in `frontend/lib/copy.ts`. `"use client"` only for forms and file inputs. Named exports; filename matches the export.

| Route or component | Responsibility |
| --- | --- |
| `/u/[username]` | Cover, avatar, display name, `@username`, bio, posts slot (empty copy). 404 state for missing users. Public (no login required). |
| `/settings/profile` | Edit display name, bio, username (show cooldown). Session required. |
| `/onboarding/username` | Required username pick; session required. |
| `Avatar` | Image or initials; `alt` / accessible name = display name. |
| `CoverPhoto` | Image or empty banner; `alt` as in F-03-15. |
| `UsernameGate` | Client redirect when `user && me.username == null` (see F-03-04). |
| Header | Link to `/u/{username}` or onboarding; keep existing auth links. |

`pnpm` in `frontend/` ([12](12-frontend-pnpm.md)). Tests: Vitest + Testing Library for happy path, 404, validation errors, onboarding gate. UI PRs embed **real dark-theme screenshots** in the PR body; PNGs go in `docs/web/screenshots/` via **Git LFS**.

## Privacy, authz, and personal data

| Action | Anonymous | Other user | Owner | Blocked pair (04) |
| --- | --- | --- | --- | --- |
| `GET /v1/users/{username}` | public fields | public fields | public fields (use `/me` for email) | **404** |
| `GET /v1/me/profile` | 401 | 401 | full | n/a |
| `PATCH` / upload / delete | 401 | 401 | yes | n/a |

- Only the owner mutates. Resource existence for hidden/blocked profiles is **404**.
- Email and DOB never appear on `/u/{username}` or in `PublicProfile`.
- Avatars are not biometric templates (Law 91/2025). EXIF GPS is stripped.
- Correction of name/bio/photos is this feature; erasure remains 02 phase 5.
- Do not log object keys next to session IDs, raw files, or emails. Log `user_id`, `kind`, `bytes`, `request_id`.

## Non-functional and scaling considerations

- Public profile GET p95 **100 ms** with Redis cache (NFR table in [00](../00-project-overview.md)).
- `user_id` is the shard key. Username uniqueness is a **global** constraint (same class of problem as email in 02). After shard, a username directory service is required; do not over-design now. The unique index is enough for a single primary.
- Object keys are globally unique and immutable; CDN cache can be `immutable` / long `max-age` because a replace mints a new key.
- 100M users × ~2 images × ~200 KiB transcoded ≈ tens of TiB — that is why bytes live in object storage, not Postgres.
- In-request transcode is acceptable for avatars/covers (small, one file). Post bursts in 05 may need a worker (new ADR).
- `GET /v1/media/*` is a **dev/origin convenience**, not the 100M path. Production `MEDIA_PUBLIC_BASE_URL` is the CDN.

## Security extras

- Multipart: reject extra files; one `file` field.
- Media GET: the key allowlist above; never `filepath.Join` with raw URL path into `MEDIA_DIR` without cleaning; `http.ServeContent` with `Content-Type` from the store (JPEG), `Cache-Control: public, max-age=31536000, immutable`.
- Username enumeration on public GET is **accepted** (profiles are public). Rate limit per IP.
- CSRF: Origin check on mutating `/v1/me/profile*` (same as `/v1/auth/*`).

## Dependencies on other features

- Requires: [01](01-project-foundation.md) shipped; [02](02-accounts.md) **phases 1–3** (user row, session cookie, password flows). **Do not implement 02 phase 4 or 5.**
- Soft-requires: [04](04-friends.md) for `BlockChecker` (interface + no-op).
- Tooling: [11](11-local-dev-tooling.md) Makefile / ports; [12](12-frontend-pnpm.md) pnpm. Phase 3 updates Makefile, Compose, and [deployment-local.md](../deployment/deployment-local.md) when MinIO is added.
- Enables: 04 (people to add), 05 (author byline, `ObjectStore` reuse).

## Implementation phases

Each phase is **one PR**, one concern. Stack on the previous branch; merge **in this order**. Do not mark a phase **Shipped** until it is merged to `main`. Open PRs use **In progress** and the PR link.

### Merge order (all of feature 03)

1. This plan + [ADR 0009](../decisions/0009-object-storage.md) + docs index
2. Phase 1 — OpenAPI
3. Phase 2 — Migration + sqlc
4. Phase 3 — Object storage platform
5. Phase 4 — Profile backend (text fields)
6. Phase 5 — Upload backend
7. Phase 6 — Frontend profile page
8. Phase 7 — Frontend edit + onboarding
9. Phase 8 — Avatar / cover upload UI

### Phase 1 — OpenAPI contract

**Status:** In progress  
**PR:** https://github.com/anhtuta/friendfold/pull/43 (after https://github.com/anhtuta/friendfold/pull/42)

Work:

- Add tag `profile` and the endpoints/schemas/errors listed above to `backend/api/openapi.yaml`.
- Document Origin on mutating `/v1/me/profile*`, optional auth on public GET, multipart `file`.
- Fake examples only.
- Regenerate `frontend/lib/api/schema.d.ts` (`pnpm openapi:types`) so the contract is in the client types.

Acceptance:

- [ ] Redocly lint passes.
- [ ] `MeProfile` has email; `PublicProfile` does not.
- [ ] Snowflake IDs are strings; examples are low-entropy fakes.

### Phase 2 — Migration + sqlc

**Status:** In progress  
**PR:** https://github.com/anhtuta/friendfold/pull/45 (after https://github.com/anhtuta/friendfold/pull/43)

Work:

- Goose `00004_profiles.sql`: `profiles` and `media_objects` as specified. Do not edit 00002/00003.
- `backend/db/queries/profile.sql` → `internal/profile/sqlc`.
- sqlc.yaml second (third) package. No handler/service yet.

Acceptance:

- [ ] `make sqlc-diff` clean after generate.
- [ ] Unique username partial index exists.
- [ ] Down migration drops the new tables.

### Phase 3 — Object storage platform

**Status:** In progress  
**PR:** https://github.com/anhtuta/friendfold/pull/46 (after https://github.com/anhtuta/friendfold/pull/45)

Work:

- `internal/platform/blob`: interface, `fs` driver, `s3` driver, key validation, tests (temp dir; MinIO optional / fake).
- Config + `.env.example` names for `MEDIA_*`.
- MinIO service in `docker-compose.yml`; gitignore `data/media/`.
- Makefile help text if new targets are needed (`make up` already starts Compose).
- Update [deployment-local.md](../deployment/deployment-local.md) (ports 9000/9001, env table, troubleshooting).
- No profile HTTP yet.

Acceptance:

- [ ] Put/Get/Delete round-trip on `fs`.
- [ ] Invalid keys (`..`, absolute, wrong suffix) rejected.
- [ ] Compose file includes MinIO; local-only credentials documented.

### Phase 4 — Profile backend (username, display name, bio)

**Status:** In progress  
**PR:** (stacked; URL filled when the PR opens)

Work:

- `internal/profile` service + handlers: GET/PATCH `/v1/me/profile`, GET `/v1/users/{username}`.
- Username validation, reserved list, 14-day cooldown, unique 409.
- Redis public-profile cache + invalidation; rate limits.
- `Account` interface: `ByID`, `UpdateDisplayName`. Auth store gains `UpdateDisplayName` only (no Google, no deletion).
- `BlockChecker` interface, no-op implementation wired in `cmd/api`.
- CORS allows PATCH (and DELETE, for phase 5).
- Preserve comments in `backend/cmd/api/main.go`; mount profile next to auth.
- Table-driven service tests; handler tests for authz, 409, public GET hiding email, cooldown, reserved names.

Acceptance:

- [ ] Unique username enforced (index + 409).
- [ ] Reserved names 400 `username_reserved`.
- [ ] Public GET has no email.
- [ ] Second username change within 14 days is 409 `username_cooldown`.
- [ ] Cache miss and hit return the same public JSON.
- [ ] Unauthenticated PATCH is 401; Origin mismatch 403.

### Phase 5 — Upload backend

**Status:** Not started  
**PR:** —

Work:

- POST/DELETE avatar and cover; transcode pipeline; `media_objects`; `GET /v1/media/*`.
- Replace deletes previous object best-effort.
- Tests: JPEG/PNG/WebP accepted; GIF/PDF rejected; oversize 413; in-memory JPEG with GPS EXIF → stored file has no EXIF APP1/GPS; pixel-bomb header rejected; unauthenticated upload 401.

Acceptance:

- [ ] F-03-11, F-03-12, F-03-13 covered by tests.
- [ ] Stored object is JPEG; JSON URL uses `MEDIA_PUBLIC_BASE_URL`.
- [ ] DELETE avatar leaves initials path (`avatarUrl` null).

### Phase 6 — Frontend profile page

**Status:** Not started  
**PR:** —

Work:

- `/u/[username]`: cover, avatar, name, bio, empty posts slot, 404 state.
- `Avatar`, `CoverPhoto`.
- Header link when the viewer has a username.
- Vitest for happy path + 404. Dark-theme screenshots (found, 404, own vs other if distinguishable) in `docs/web/screenshots/` via LFS.

Acceptance:

- [ ] `/u/{username}` shows name + bio.
- [ ] Missing user is a 404 page, not a blank error.
- [ ] Email is not rendered.
- [ ] Screenshots in the PR description.

### Phase 7 — Frontend edit and onboarding

**Status:** Not started  
**PR:** —

Work:

- `/onboarding/username` + `UsernameGate` (F-03-04).
- `/settings/profile` for display name, bio, username (cooldown copy).
- Tests for gate, validation, save success/error.
- Screenshots: onboarding, settings, validation error, cooldown if mockable.

Acceptance:

- [ ] New user cannot use `/` or settings until a username is set.
- [ ] Settings save PATCHes and shows the new bio on `/u/{username}`.
- [ ] Screenshots in the PR description.

### Phase 8 — Avatar and cover upload UI

**Status:** Not started  
**PR:** —

Work:

- File inputs on settings (and optional onboarding skip for photos).
- Preview, replace, remove; client-side type/size hint (server still authoritative).
- Tests for reject-on-client-hint and success path (mock `fetch`).
- Screenshots: default initials, after avatar, after cover, remove.

Acceptance:

- [ ] JPEG/PNG/WebP can be uploaded from settings and appear on `/u/{username}`.
- [ ] `alt` present.
- [ ] Screenshots in the PR description.

When **all eight implementation phases are merged**, set this feature **Status** to **Shipped** and tick the checklist below. Until then, phases on open PRs stay **In progress**.

## Testing plan

- Unit: username charset/length/reserved, cooldown math, URL join, key validation, image sniff/resize/EXIF strip (synthetic JPEG), pixel cap.
- Handler: lazy me, public GET hides email, 404 unknown, 401/403, 409 taken and cooldown, 429 shape, multipart happy path and GIF reject, media GET 200/404, path traversal 404.
- Authz: blocked viewer 404 once a fake `BlockChecker` returns true; `only_me` N/A here.
- Frontend: profile found/404, onboarding blocks `/`, settings save, upload control.
- Manual: `make up && make migrate && make api` + `make frontend`; Mailpit unused. Optional `MEDIA_DRIVER=s3` against Compose MinIO.

## Open questions

Resolved in this version (v2):

- **Profile visibility:** **Public.** Posts still honor post privacy in 05.
- **Username immutability:** **Changeable, 14-day cooldown** after each successful set.
- **Username vs handle:** **One identifier**, called username, URL `/u/{username}`, UI `@username`.
- **Where files live in dev:** **`ObjectStore` with `fs` default**; MinIO in Compose for the `s3` driver ([ADR 0009](../decisions/0009-object-storage.md)).
- **Thumbnails:** **One transcoded JPEG per upload** (max long edge). CDN derivatives later.
- **Columns on `users` vs `profiles`:** **`profiles` table**; display name remains on `users`.
- **Cooldown HTTP status:** **409 `username_cooldown`**. Rate limits stay 429.

Still open (do not block 03):

- **Soft-delete vs username:** 02 phase 5 must free or tombstone the handle. Default suggestion: null `profiles.username` in the same deletion transaction.
