# Feature 03: Profiles

**Status:** Not started  
**Owner:** —  
**Plan version:** 1

## Goal

Every account has a profile people can recognize: unique username, display name, bio, avatar, cover photo, and a profile page at `/u/{username}` showing that user's public (and, for friends, friends-only) information and later their posts.

## Scope

### In scope

- Username (unique, immutable **or** changeable with a cooldown — see open questions; default **changeable, 14-day cooldown**).
- Display name, bio (plain text, ≤ 160 chars).
- Avatar and cover upload (images only).
- `GET` own profile and `PATCH` own profile.
- Public profile page by username, respecting block lists once friends exist (gracefully: if friend module is not up, skip block checks via interface, no-op implementation).
- Default avatar if none uploaded.

### Out of scope

- Featured photos, albums, work/education fields, pronouns as a structured enum (bio is enough).
- Profile completeness gamification.
- Serving posts on the profile (the **profile page shell** is this feature; the posts timeline **on the profile** is filled in [05](05-posts.md)).
- Followers (Friendfold is friends, not follow-graph, except high-follower feed mechanics later).

## User stories

- As a **new user**, I want to pick a username and photo, so that friends can find me.
- As a **user**, I want to edit my bio and cover, so that my page looks like me.
- As a **visitor**, I want to open `/u/{username}` and see who they are.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-03-01 | Username: 3–20 chars, `[a-z0-9_]`, stored lowercase, unique among non-deleted users. Reserved list: `admin`, `support`, `friendfold`, `settings`, `api`, `u`. |
| F-03-02 | On first visit after feature 03 ships, if username is null, force `/onboarding/username` before other `(main)` routes. |
| F-03-03 | `PATCH /v1/me/profile` updates displayName, bio, username (cooldown), avatar/cover via separate upload endpoints. |
| F-03-04 | Avatars ≤ 5 MiB, covers ≤ 10 MiB; types `image/jpeg`, `image/png`, `image/webp`. Server strips EXIF, transcodes to JPEG/WebP, max edge 1024 (avatar) / 2048 (cover). |
| F-03-05 | `GET /v1/users/{username}` returns the public profile. 404 if missing, deleted, or the viewer is blocked (once blocks exist). |
| F-03-06 | Bio and names are plain text; reject control chars; trim whitespace. |
| F-03-07 | Own profile includes email and `emailVerified`; other viewers never see email or DOB. |

## Data model

Extend `users` (migration, do not edit 02's merged migration):

| Column | Type | Notes |
| --- | --- | --- |
| `username` | `TEXT NULL` | unique lowercase; NULL only during onboarding |
| `username_changed_at` | `timestamptz NULL` | cooldown |
| `bio` | `TEXT NOT NULL DEFAULT ''` | max 160 |
| `avatar_url` | `TEXT NULL` | object-storage URL |
| `cover_url` | `TEXT NULL` | |
| `avatar_object_key` | `TEXT NULL` | for deletion |
| `cover_object_key` | `TEXT NULL` | |

Indexes:

- unique `(username) WHERE username IS NOT NULL AND deleted_at IS NULL`

### `media_objects` (optional but recommended; can live in `platform` or `profile`)

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | |
| `user_id` | `BIGINT NOT NULL` | uploader |
| `kind` | `TEXT NOT NULL` | `avatar` \| `cover` \| `post` (post used in 05) |
| `object_key` | `TEXT NOT NULL` | |
| `mime` | `TEXT NOT NULL` | |
| `bytes` | `INT NOT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |

Local dev: filesystem or MinIO. Production: S3/R2. Interface `platform.ObjectStore`. **Do not add MinIO to Compose until this phase if a local directory is simpler** — default: **local directory `./data/media` in API**, documented; MinIO can wait.

Redis: `profile:{userId}` JSON cache, TTL 5 minutes, invalidated on PATCH.

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/v1/me/profile` | user | Full own profile |
| `PATCH` | `/v1/me/profile` | user | Update name, bio, username |
| `POST` | `/v1/me/profile/avatar` | user | multipart upload |
| `DELETE` | `/v1/me/profile/avatar` | user | Remove avatar |
| `POST` | `/v1/me/profile/cover` | user | multipart upload |
| `DELETE` | `/v1/me/profile/cover` | user | |
| `GET` | `/v1/users/{username}` | optional | Public profile |

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `/onboarding/username` | Required username pick |
| `/settings/profile` | Edit all fields, previews |
| `/u/[username]` | Cover, avatar, name, bio, later posts slot |
| `Avatar`, `CoverPhoto` | Shared components with `alt` |

## Non-functional and scaling considerations

- Profile GET p95 100 ms; cache in Redis.
- Object storage and CDN later; API should return URLs that can switch from `/media/...` to CDN without a schema change (`avatarUrl` opaque string).
- Username unique index is a global constraint; after sharding, usernames need a directory — same comment as email in 02.
- Transcoding: keep in-request for MVP (avatars are small); if p95 exceeds 1 s, move to a worker in a later ADR.

## Security and privacy

- Only the owner can PATCH. Username change rate-limited.
- Image processing must not use a decoder crash as DoS without limits (timeout, max pixels).
- EXIF GPS stripped (privacy).
- Profile is visible to anyone who is not blocked. There is no `profile_privacy=friends` in MVP (open question). Email and DOB never appear on `/u/{username}`.
- Law 91/2025: user can access and correct their profile here; deletion is 02 phase 5.

## Dependencies on other features

- Requires: [02](02-accounts.md) phases 1+ (user row + session). Google and reset are not hard dependencies.
- Soft-requires: [04](04-friends.md) for block-aware profile GET (interface + no-op until 04).
- Enables: 04 (people to add), 05 (author byline).

## Implementation phases

### Phase 1 — Username, display name, bio

**Status:** Not started  
**PR:** —

Work:

- Migration adding columns; OpenAPI; GET/PATCH; onboarding + settings form (no images).

Acceptance:

- [ ] Unique username enforced.
- [ ] Reserved names rejected.
- [ ] `/u/{username}` shows name + bio.
- [ ] Email not in the public payload.

### Phase 2 — Avatar and cover

**Status:** Not started  
**PR:** —

Work:

- Object store interface, upload endpoints, transcode/strip, settings UI, default avatar.

Acceptance:

- [ ] JPEG/PNG/WebP accepted; GIF/PDF rejected.
- [ ] Oversize rejected.
- [ ] EXIF GPS not present in stored file (test with a fixture).
- [ ] Profile page shows images with alt text.

### Phase 3 — Cache, cooldown, polish

**Status:** Not started  
**PR:** —

Work:

- Redis profile cache + invalidation; username 14-day cooldown; empty states; 404 page.

Acceptance:

- [ ] Second username change within 14 days returns 429/409 as documented.
- [ ] Cache miss and hit both return the same JSON.

## Testing plan

- Unit: username validation, reserved list, cooldown.
- Handler: unique conflict 409, authz on PATCH, public GET hides email.
- Upload: MIME sniffing (do not trust `Content-Type` alone), pixel bomb rejection if implemented.
- Frontend: onboarding blocks other routes; settings save.

## Open questions

- **Profile visibility:** public vs friends-only profiles. Default: **profiles are public**, posts still honor post privacy.
- **Username immutability:** Default: **changeable with 14-day cooldown**.
- **Where files live in dev:** Default: **local disk**; Compose MinIO only if local disk is painful.
