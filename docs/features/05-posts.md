# Feature 05: Posts

**Status:** Not started  
**Owner:** —  
**Plan version:** 1

## Goal

A user can publish a text post or a photo post (one or more images), choose who can see it (`public`, `friends`, `only me`), edit the text, and delete the post. Other people see the post on the author's profile when they are allowed to.

## Scope

### In scope

- Create post: body text (optional if there is at least one photo), 1–10 photos, privacy.
- Edit body and privacy (not replacing the photo set in MVP — see phases).
- Soft delete (`deleted_at`); owner only.
- Get a single post by ID with authz.
- List posts **by author** for the profile timeline (cursor, privacy-aware).
- Reuse `media_objects` / object store from [03](03-profiles.md).

### Out of scope

- Video, GIFs-as-video, albums as a separate entity, tagged people, location, feeling/activity, shares/reposts, hashtags, links unfurling (plain URLs as text are fine).
- The home **news feed** ([06](06-news-feed.md)).
- Likes and comments ([07](07-likes-and-comments.md)) — schema may include `like_count` / `comment_count` defaults of 0 so 07 does not rewrite the posts table, or 07 adds those columns. **Add nullable counters in 07**, not here.

## User stories

- As a **user**, I want to write a status and pick Friends or Public, so that the right people see it.
- As a **user**, I want to attach photos, so that I can share a moment.
- As a **user**, I want to fix a typo or delete a post.
- As a **visitor**, I want to see public posts on a profile; as a **friend**, I also want to see friends-only posts.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-05-01 | Body: plain text, 0–5000 characters after trim. Must have body and/or ≥1 photo. |
| F-05-02 | Privacy: `public` \| `friends` \| `onlyMe`. Default `friends`. |
| F-05-03 | Photos: ≤10, each ≤ 10 MiB, jpeg/png/webp; strip EXIF; transcode; store derivatives (feed-width and a larger view). |
| F-05-04 | `GET /v1/posts/{id}`: 404 if deleted or viewer not allowed. Owner always sees own `onlyMe`. |
| F-05-05 | `GET /v1/users/{username}/posts`: privacy-aware, cursor `created_at, id`. |
| F-05-06 | Edit: owner, body and/or privacy, sets `edited_at`. Photos immutable in MVP (delete and recreate if needed) **or** phase 3 adds photo edit — default **immutable photos**. |
| F-05-07 | Delete: owner, soft delete. GET returns 404. |
| F-05-08 | Composer on the profile and (once 06 exists) on the feed; this feature ships composer on `/` placeholder and profile. |
| F-05-09 | After create, call a `feed.OnPostCreated` interface (no-op until 06). |

## Data model

### `posts`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | Snowflake, time-ordered |
| `author_id` | `BIGINT NOT NULL` | shard key |
| `body` | `TEXT NOT NULL DEFAULT ''` | |
| `privacy` | `TEXT NOT NULL` | `public` \| `friends` \| `only_me` (DB snake; JSON `onlyMe`) |
| `created_at` | `timestamptz NOT NULL` | |
| `updated_at` | `timestamptz NOT NULL` | |
| `edited_at` | `timestamptz NULL` | |
| `deleted_at` | `timestamptz NULL` | |

Indexes:

- PK `(id)`
- `(author_id, created_at DESC) WHERE deleted_at IS NULL` — profile timeline
- `(id) WHERE deleted_at IS NULL` covering as needed

### `post_media`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | |
| `post_id` | `BIGINT NOT NULL` | |
| `object_key` | `TEXT NOT NULL` | |
| `url` | `TEXT NOT NULL` | public or signed |
| `width` | `INT NOT NULL` | |
| `height` | `INT NOT NULL` | |
| `mime` | `TEXT NOT NULL` | |
| `sort_order` | `INT NOT NULL` | 0-based |

Indexes:

- `(post_id, sort_order)` unique

Authz helper (in `post` service, using `friend` interfaces):

```
CanView(viewer *User, post) bool
  // nil viewer: public only
  // owner: yes
  // blocked either way: no (except owner's own)
  // public: yes
  // friends: AreFriends
  // only_me: owner only
```

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/v1/posts` | user | Create (JSON body + media IDs **or** multipart). Prefer **two-step**: upload media → create post with `mediaIds`. |
| `POST` | `/v1/media` | user | Upload an image, returns `{ id, url, width, height }` not yet attached |
| `GET` | `/v1/posts/{id}` | optional | Get one |
| `PATCH` | `/v1/posts/{id}` | user (owner) | Edit body/privacy |
| `DELETE` | `/v1/posts/{id}` | user (owner) | Soft delete |
| `GET` | `/v1/users/{username}/posts` | optional | Profile timeline |

Orphan media (uploaded but never attached) is deleted by a later GC job; for MVP, attach in the same session and ignore orphans.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `PostComposer` | Text, photo picker, privacy select, submit |
| `PostCard` | Author, time, body, images, privacy icon, owner menu |
| `PostMenu` | Edit, delete, later Report |
| `/p/[id]` | Single post permalink |
| `/u/[username]` | Timeline list (infinite scroll can wait for 06; page-size 20 + "more" is enough) |

## Non-functional and scaling considerations

- Create (without media) p95 200 ms. Media upload excluded (NFR 1 s processing).
- Profile timeline is `author_id` clustered — shard-friendly.
- Public posts will eventually be cached; skip CDN HTML cache until we have CDN.
- At 100M users, posts table is the largest. Partition by `created_at` or shard by `author_id` later; **include `author_id` on every query that can**.
- Do not store the body in Redis; cache the rendered `PostCard` JSON for hot permalinks if needed (phase 3 optional).

## Security and privacy

- Authz as `CanView`; leaks of `only_me` via 403 vs 404 are forbidden — **always 404**.
- Edit/delete: owner only (staff override is 09, not here).
- Rate limit creates: e.g. 10/hour, 50/day (platform limiter).
- Plain text only; React escape. No `dangerouslySetInnerHTML`.
- EXIF strip on photos.
- Privacy change from `public` → `friends` must take effect on subsequent GET immediately (invalidate any cache).

## Dependencies on other features

- Requires: [03](03-profiles.md) (author byline, uploads), [04](04-friends.md) (`AreFriends` / `IsBlocked`).
- Enables: 06, 07, 09 reports on posts.

## Implementation phases

### Phase 1 — Text posts + privacy + profile list + permalink

**Status:** Not started  
**PR:** —

Work:

- `posts` table, create/get/patch/delete, profile list, composer without photos, `PostCard`, `/p/[id]`.
- OpenAPI first.

Acceptance:

- [ ] Friends-only post 404 for a stranger; 200 for a friend and the owner.
- [ ] `onlyMe` 404 for a friend.
- [ ] Edit and delete owner-only.
- [ ] Soft-deleted 404.

### Phase 2 — Photo uploads attached to posts

**Status:** Not started  
**PR:** —

Work:

- `POST /v1/media`, `post_media`, composer picker, gallery in `PostCard`.

Acceptance:

- [ ] Post with 1–10 images renders in order.
- [ ] Text-only still works; image-only (empty body) works.
- [ ] Bad MIME rejected.

### Phase 3 — Composer polish and cache invalidation

**Status:** Not started  
**PR:** —

Work:

- Privacy default, edited label, empty profile state, optional permalink cache.

Acceptance:

- [ ] `edited` visible when `editedAt` set.
- [ ] Privacy change is visible on the next anonymous GET without restart.

## Testing plan

- Unit: `CanView` matrix (9 cases: 3 privacy × stranger/friend/owner, plus blocked).
- Handler: validation (empty post), pagination cursors.
- Frontend: composer disable when empty; delete confirm.

## Open questions

- **Multipart vs two-step media:** Default **two-step** (`POST /v1/media` then `POST /v1/posts`).
- **Photo edit:** Default **not in MVP**.
- **Link previews:** Default **no**.
