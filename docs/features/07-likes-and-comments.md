# Feature 07: Likes and comments

**Status:** Not started  
**Owner:** —  
**Plan version:** 1

## Goal

People can like a post, unlike it, comment on it, and reply one level deep to a comment. Counts on `PostCard` stay correct. This is the engagement layer notifications (08) will hook.

## Scope

### In scope

- One reaction type in the UI: **Like**. Schema column `type` so more reactions can be added without a redesign.
- Comments on posts; replies to **top-level** comments only (no thread-of-thread).
- Edit/delete own comment (soft delete).
- Paginated comment list on the permalink and an inline preview (first N) on the feed card.
- Counters on the post: `like_count`, `comment_count` (top-level + replies, or top-level only — **count all non-deleted comments including replies**).

### Out of scope

- Reactions on comments (like a comment) — later.
- Nested replies beyond one level, mentions autocomplete, rich text, photos in comments, edit history.
- Shares.

## User stories

- As a **user**, I want to like a friend's post, so that they know I saw it.
- As a **user**, I want to comment and reply, so that we can talk on the post.
- As a **user**, I want to delete my comment.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-07-01 | Like is a toggle. Duplicate like is idempotent 200. Unlike of a non-like is idempotent 200. |
| F-07-02 | Viewer must `CanView` the post to like or comment; otherwise 404. |
| F-07-03 | Comment body 1–2000 chars, plain text. |
| F-07-04 | `parentId` null = top-level; non-null must reference a **top-level** comment on the **same** post. Reply-to-reply 400. |
| F-07-05 | Soft-delete comment: body replaced with empty in API as `deleted: true`, replies remain. |
| F-07-06 | Owner of the **post** may delete any comment on it (moderation-lite). |
| F-07-07 | Lists: likes (optional, paginated, **not** in MVP UI except count) and comments (cursor, top-level, with nested `replies` first page). |
| F-07-08 | After like/comment, call `notification` interface (no-op until 08). |

## Data model

### `posts` columns added

| Column | Type | Notes |
| --- | --- | --- |
| `like_count` | `INT NOT NULL DEFAULT 0` | |
| `comment_count` | `INT NOT NULL DEFAULT 0` | |

Maintain in the same transaction as the like/comment write. Redis cache `post:counts:{id}` optional.

### `reactions`

| Column | Type | Notes |
| --- | --- | --- |
| `user_id` | `BIGINT NOT NULL` | |
| `post_id` | `BIGINT NOT NULL` | |
| `type` | `TEXT NOT NULL` | `like` |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- PK `(post_id, user_id)` — one reaction per user per post in MVP
- `(user_id, created_at DESC)` if we ever list "my likes"

### `comments`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | |
| `post_id` | `BIGINT NOT NULL` | |
| `author_id` | `BIGINT NOT NULL` | |
| `parent_id` | `BIGINT NULL` | top-level comment id |
| `body` | `TEXT NOT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |
| `updated_at` | `timestamptz NOT NULL` | |
| `deleted_at` | `timestamptz NULL` | |

Indexes:

- `(post_id, created_at, id) WHERE parent_id IS NULL AND deleted_at IS NULL` — top-level list
- `(parent_id, created_at, id) WHERE parent_id IS NOT NULL` — replies
- PK `(id)`

Do not FK `parent_id` across shards later; in monolith, FK to `comments(id)` is fine.

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `PUT` | `/v1/posts/{id}/like` | user | Like (idempotent) |
| `DELETE` | `/v1/posts/{id}/like` | user | Unlike |
| `GET` | `/v1/posts/{id}/likes` | user + CanView | Optional; skip UI |
| `POST` | `/v1/posts/{id}/comments` | user | Create (`parentId` optional) |
| `GET` | `/v1/posts/{id}/comments` | optional + CanView | Top-level page; embed first 3 replies |
| `GET` | `/v1/comments/{id}/replies` | optional + CanView | More replies |
| `PATCH` | `/v1/comments/{id}` | author | Edit body |
| `DELETE` | `/v1/comments/{id}` | author or post owner | Soft delete |

`Post` JSON from 05/06 gains `likeCount`, `commentCount`, `likedByMe`.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `LikeButton` | Toggle, optimistic count, auth gate |
| `CommentList` | Thread on `/p/[id]` |
| `CommentComposer` | Top-level and reply |
| `PostCard` engagement row | Like + comment count; click comment → permalink or expand |

## Non-functional and scaling considerations

- Like p95 100 ms (single upsert + increment). Counter drift: nightly reconcile from `COUNT(*)` if we see inconsistency; Redis `INCR` must not be the only source.
- Comment create p95 200 ms.
- Hot posts (viral public post): `reactions` PK by `post_id` localizes likes. At extreme load, shard likes by `post_id` or use a counter service — later.
- Feed cards should **not** fetch full comment threads; counts + `likedByMe` only. Hydrate `likedByMe` via a batch `WHERE user_id = me AND post_id IN (...)`.

## Security and privacy

- `CanView` on every read/write. 404 otherwise.
- Rate limit comments (e.g. 30/hour) and likes (higher, e.g. 200/hour) via platform limiter.
- Soft-deleted comments do not expose the old body.
- XSS: plain text, escaped.

## Dependencies on other features

- Requires: [05](05-posts.md) (and 06 if we show buttons on the feed — feed cards should still update if 06 shipped).
- Enables: 08 (like/comment events), 09 (report comment — out of 09 MVP unless we add it; 09 says post/user).

## Implementation phases

### Phase 1 — Likes + counts + `likedByMe` on cards

**Status:** Not started  
**PR:** —

Work:

- `reactions` table, counters, PUT/DELETE like, OpenAPI, `LikeButton`, batch likedByMe in feed/profile hydrate.

Acceptance:

- [ ] Toggle like updates count without reload.
- [ ] Two likes from the same user do not increment twice.
- [ ] Cannot like a 404 post.

### Phase 2 — Comments + permalink thread

**Status:** Not started  
**PR:** —

Work:

- `comments` table, POST/GET/PATCH/DELETE, thread UI on `/p/[id]`.

Acceptance:

- [ ] Comment appears, count increments.
- [ ] Author and post owner can delete; others 403/404.
- [ ] Deleted comment shows a placeholder, not the body.

### Phase 3 — Replies (one level) + feed preview

**Status:** Not started  
**PR:** —

Work:

- `parentId`, replies endpoint, reply composer, reject deeper nesting, optional first-two comments preview on `PostCard`.

Acceptance:

- [ ] Reply attaches to top-level; reply-to-reply 400.
- [ ] Pagination of replies does not duplicate.

## Testing plan

- Unit: counter increment/decrement never negative; parent validation.
- Handler: authz matrix, idempotent like.
- Frontend: optimistic like rollback on 401.

## Open questions

- **More reaction types (wow/sad):** schema-ready; UI is like-only.
- **Count replies in `commentCount`:** Default **yes**.
- **Post owner deletes comments:** Default **yes**.
