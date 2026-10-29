# Feature 06: News feed

**Status:** Not started  
**Owner:** —  
**Plan version:** 1

## Goal

A signed-in user opening Home sees **friends' posts they are allowed to see**, newest first, and can keep scrolling (infinite scroll / cursor). This is a chronological friends feed. A ranked "For You" feed is explicitly later. The write/read path is designed so ordinary users use **fan-out-on-write** and high-degree accounts use **fan-out-on-read**.

## Scope

### In scope

- Home timeline API and UI: infinite scroll, newest first.
- Hybrid fan-out as specified below.
- Respect post privacy, deletions, blocks, unfriends (eventually consistent within a short bound).
- Empty state when the user has no friends / no posts.

### Out of scope

- Ranked feed, ads, reels, suggested strangers' public posts in the home feed.
- Comments/likes in the feed card beyond counts — counts come in [07](07-likes-and-comments.md); until then omit or show 0.
- Read-state ("seen") tracking.

## User stories

- As a **user**, I want to open Friendfold and see what my friends posted lately, newest first.
- As a **user**, I want to scroll for more without paging controls.
- As a **user with many friends**, I still want Home to load quickly.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-06-01 | `GET /v1/feed?cursor&limit` returns posts for the session user, newest first, only `CanView`. |
| F-06-02 | Infinite scroll in the Home route; same `PostCard` as 05; composer at top. |
| F-06-03 | Default `limit=20`, max 50. Cursor opaque (base64 of `{createdAt, id}`). |
| F-06-04 | Unfriend/block: their posts disappear from subsequent fetches (Redis timeline entries may lag; **max lag 60 s** or next write repair). |
| F-06-05 | Deleted posts: skip; do not 500. |
| F-06-06 | Own posts **do** appear in Home. |
| F-06-07 | Anonymous `/` does not show a feed; it shows the marketing/login page. |

## Data model

Postgres remains source of truth (`posts` + `friendships`). Redis holds precomputed timelines.

### Redis `timeline:{userId}`

- Type: **ZSET**, score = post created_at millis (or Snowflake id, which is time-ordered).
- Member: post ID string.
- Cap: **1000** entries (trim on write). Older history is **fan-out-on-read** / SQL fallback.
- TTL: none (trim by rank). If the key is missing, rebuild from SQL (last N posts from friends + self, privacy-aware — expensive; cache the result).

### Redis `user:fanout_mode:{userId}`

- `write` (default) or `read` (celebrity / high-degree).
- Threshold: **friend count ≥ 5,000** (or the friend cap from 04) flips the user to `read` mode. Until we have celebrities, almost everyone is `write`.

### Optional `feed_items` (not required for MVP)

A Postgres table of precomputed edges is a later scale-up; skip unless Redis rebuilds are too slow. If added:

| Column | Type | Notes |
| --- | --- | --- |
| `user_id` | `BIGINT` | timeline owner |
| `post_id` | `BIGINT` | |
| `author_id` | `BIGINT` | |
| `created_at` | `timestamptz` | |

PK `(user_id, post_id)`, index `(user_id, created_at DESC)`.

### Hybrid fan-out

**Fan-out-on-write (default authors):**

1. User A creates a post that is `public` or `friends`.
2. For each friend B of A (and A themselves), `ZADD timeline:{B} score postId`.
3. `only_me` posts go only to `timeline:{A}`.
4. If A has > threshold friends, **skip** 2 and mark A `read` mode.

**Fan-out-on-read (high-degree authors):**

At read time for viewer V:

1. `ZREVRANGE timeline:{V}` for the cached ordinary-friends posts.
2. Load the small set of V's friends who are in `read` mode (Redis SET `friends_read_mode:{V}` maintained on friendship changes, expected tiny).
3. For each such friend C, fetch C's recent post IDs from Postgres `(author_id=C ORDER BY created_at DESC LIMIT k)` with privacy.
4. Merge-sort by time, apply `CanView`, paginate.

This is the Instagram/Twitter hybrid: pay write amplification for typical friend graphs; pay read merge for the rare huge degree.

**Fan-out-on-read fallback for everyone** (phase 1): if Redis is empty, SQL:

```sql
SELECT p.* FROM posts p
JOIN friendships f ON f.friend_id = p.author_id
WHERE f.user_id = $viewer
  AND p.deleted_at IS NULL
  AND p.created_at < $cursor
  AND (p.privacy = 'public' OR p.privacy = 'friends' OR p.author_id = $viewer)
ORDER BY p.created_at DESC, p.id DESC
LIMIT $limit
```

Plus viewer's own posts (already included if self-friend is stored — **do not** insert a self-friendship; `UNION` own posts). Still filter blocks.

Phase 1 may ship **only** this SQL (plus Redis cache of the page). Phase 2 adds write fan-out. Phase 3 adds hybrid read-mode.

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/v1/feed` | user | Home timeline page |

No write endpoints; 05's create hooks `feed.OnPostCreated`.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `/` (authenticated) | Composer + infinite list |
| `FeedList` | Intersection Observer / "Load more", error retry |
| Empty state | "Add friends to see posts" → `/friends` |

## Non-functional and scaling considerations

- First page p95 **150 ms** with warm Redis; SQL fallback p95 **300 ms** at modest degree.
- Write fan-out of 500 friends must finish **asynchronously** if it would exceed the 200 ms post-create budget: the author gets 201/200 immediately; workers push Redis. MVP: **in-process goroutine with bounded concurrency**, not Kafka. Kafka is the later replacement (overview).
- 100M users: Redis timeline memory ≈ DAU × 1000 × ~40 bytes. Size it when we have DAU, not now.
- Never `OFFSET`. Never `SELECT * FROM posts ORDER BY created_at` globally.

## Security and privacy

- Always re-check `CanView` at read even if the ID is in Redis (privacy may have changed, block may have landed).
- Feed is authenticated; do not offer a public global firehose.
- Do not leak `only_me` posts of others via timing (skip, don't 403).

## Dependencies on other features

- Requires: [05](05-posts.md), [04](04-friends.md).
- Enables: 07 (engagement on cards), 08 (not required for feed).

## Implementation phases

### Phase 1 — SQL chronological feed + Home UI + infinite scroll

**Status:** Not started  
**PR:** —

Work:

- `GET /v1/feed` SQL implementation, OpenAPI, Home route, infinite scroll, empty state, `OnPostCreated` no-op.

Acceptance:

- [ ] Friend's new post appears on Home after refresh/scroll from top.
- [ ] Stranger's public post does **not** appear unless they are a friend (home is friends-only, including their public posts).
- [ ] `onlyMe` of a friend does not appear.
- [ ] Infinite scroll fetches the next cursor without duplicates.

### Phase 2 — Fan-out-on-write to Redis ZSETs

**Status:** Not started  
**PR:** —

Work:

- `OnPostCreated` / delete / privacy change / unfriend repair; read path prefers ZSET then hydrates posts by ID; miss rebuilds from SQL.

Acceptance:

- [ ] After create, friend's next `GET /v1/feed` can be served from Redis (test with a spy or key assert).
- [ ] Delete removes the member from affected timelines (best-effort + skip on hydrate).
- [ ] Cap 1000 enforced.

### Phase 3 — Fan-out-on-read for high-degree authors

**Status:** Not started  
**PR:** —

Work:

- `fanout_mode`, skip write fan-out above threshold, merge at read, fixture test with a fake high-degree user (threshold injectable, e.g. 3 in tests).

Acceptance:

- [ ] Author over threshold: no `ZADD` to all friends (assert write count).
- [ ] Viewer still sees that author's viewable posts in the merged page, correctly ordered.

## Testing plan

- Unit: merge of ZSET + celebrity SQL, cursor encoding, `CanView` on hydrate.
- Handler: empty feed, pagination uniqueness, unauthenticated 401.
- Frontend: scroll loads page 2; composer at top; empty state CTA.

## Open questions

- **Include public posts from non-friends?** Default **no** (pure friends feed).
- **Async fan-out in MVP:** Default **goroutine**; Kafka later.
- **Threshold 5000:** injectable config `FEED_WRITE_FANOUT_MAX_FRIENDS`.
