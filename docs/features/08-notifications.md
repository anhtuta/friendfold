# Feature 08: Notifications

**Status:** Not started  
**Owner:** —  
**Plan version:** 1

## Goal

When someone sends a friend request, likes your post, or comments (or replies) on your post, you see an in-app notification. The tray lists them, they can be marked read, an unread badge updates, and while the app is open they arrive in **realtime**.

## Scope

### In scope

- Persist notifications for: `friend_request`, `friend_accept`, `post_like`, `post_comment`, `comment_reply`.
- List, mark one read, mark all read, unread count.
- Realtime: a connection the browser holds (see phases: SSE first, then WebSocket if needed).
- Deep link to the relevant request, post, or comment.
- Grouping **likes** on the same post from many people is nice-to-have (phase 3); MVP can be one row per actor.

### Out of scope

- Email/push notifications (no mobile app; email for verify/reset only).
- A dedicated WebSocket **gateway service** (that is the later topology in the overview). MVP realtime runs inside the API process.
- Notifications for "someone posted" (that is the feed).
- Notification settings matrix (mute likes, etc.) — one global "pause" is optional; default **all on**.

## User stories

- As a **user**, I want to know when someone wants to be friends or talks on my post, without refreshing.
- As a **user**, I want to open the tray, jump to the content, and clear unread.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-08-01 | Insert a notification when the matching event fires, **except** when actor == recipient (no self-notify). |
| F-08-02 | `GET /v1/notifications` newest first, cursor pagination, includes actor summary (username, avatar, displayName) and `readAt`. |
| F-08-03 | `POST /v1/notifications/{id}/read` and `POST /v1/notifications/read-all`. |
| F-08-04 | `GET /v1/notifications/unread-count` for the badge. Redis cache `notif:unread:{userId}`. |
| F-08-05 | Realtime: authenticated stream pushes `{ type, notification }` so the tray and badge update without polling more often than a slow fallback. |
| F-08-06 | Clicking a friend_request item goes to `/friends/requests`; post_like / comment to `/p/{postId}`. |
| F-08-07 | If the underlying post is no longer viewable, opening still 404s; the tray item can remain with a generic "this content is unavailable" on click. |
| F-08-08 | Fan-in: comment on your post notifies the **post author**; reply notifies the **parent comment author** (and the post author if different — **notify both**, skip duplicates if same person). |

## Data model

### `notifications`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | |
| `recipient_id` | `BIGINT NOT NULL` | shard key |
| `actor_id` | `BIGINT NOT NULL` | |
| `type` | `TEXT NOT NULL` | enum above |
| `entity_type` | `TEXT NOT NULL` | `user` \| `post` \| `comment` \| `friend_request` |
| `entity_id` | `BIGINT NOT NULL` | |
| `read_at` | `timestamptz NULL` | |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- `(recipient_id, created_at DESC, id)`
- `(recipient_id) WHERE read_at IS NULL` for counts (or Redis only)

Redis:

- `notif:unread:{userId}` integer
- Pub/sub or in-process hub: `notif:stream:{userId}` for realtime

In-process: a `Hub` map of `userId → connections` is enough for a single API replica. **When we run two API replicas, SSE/WebSocket must use Redis pub/sub** so the replica that handled the like can publish and the replica holding the socket can forward. Phase 2 should use **Redis pub/sub from the start** so we do not ship a single-replica-only design.

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/v1/notifications` | user | List |
| `GET` | `/v1/notifications/unread-count` | user | Badge |
| `POST` | `/v1/notifications/{id}/read` | user | Mark one |
| `POST` | `/v1/notifications/read-all` | user | Mark all |
| `GET` | `/v1/notifications/stream` | user | SSE (`text/event-stream`) |

Cookie auth on SSE: EventSource cannot set custom headers; the session cookie must be sent (`credentials`). Same-origin or CORS with `Access-Control-Allow-Credentials`.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| Header bell | Badge, dropdown tray (last 20), link to `/notifications` |
| `/notifications` | Full list, infinite scroll, mark all |
| `NotificationItem` | Copy per type, avatar, time, unread style |
| `useNotificationStream` | EventSource, reconnect with backoff, fallback poll every 30 s if stream fails |

Copy examples (English):

- `{name} sent you a friend request`
- `{name} accepted your friend request`
- `{name} liked your post`
- `{name} commented on your post`
- `{name} replied to your comment`

## Non-functional and scaling considerations

- List p95 100 ms; unread count p95 20 ms (Redis).
- SSE: idle connections — heartbeat comment every 15 s. Timeouts at proxies (Cloudflare ~100 s) require client reconnect; that is OK.
- 100M users: do not keep 100M sockets. Target **online DAU**. A later **WebSocket gateway** (overview) owns this; MVP API-hosted SSE + Redis pub/sub is the stepping stone.
- Notification rows grow fast. Partition by `created_at` or drop older than 90 days later. Index by `recipient_id`.

## Security and privacy

- Recipient can only list their rows.
- Stream must authenticate before sending events; no user ID in the URL as the sole auth.
- Do not put post bodies in the notification payload (avoid leaking `only_me` via a leftover row after privacy change — hydrate on click).
- Rate: notifications are server-generated; still cap list polling.

## Dependencies on other features

- Requires: 04 (requests), 07 (likes/comments). Can start tables after 01, but emitters wait for those modules.
- `friend`, `reaction`, and `comment` services already call no-op interfaces; this feature provides the implementation.

## Implementation phases

### Phase 1 — Persist + list + badge + tray (polling)

**Status:** Not started  
**PR:** —

Work:

- Table, OpenAPI, emitters wired from friend/reaction/comment, REST endpoints, bell UI, 30s poll of unread count + first page.

Acceptance:

- [ ] Like on your post creates one row; self-like does not.
- [ ] Badge matches unread.
- [ ] Mark read and mark all work.
- [ ] Friend request appears for the recipient.

### Phase 2 — Realtime SSE + Redis pub/sub

**Status:** Not started  
**PR:** —

Work:

- `GET /v1/notifications/stream`, hub + Redis subscription, client EventSource, keep poll as fallback.

Acceptance:

- [ ] Two browser tabs: a like in one updates the other tab's badge without waiting 30 s (test locally with two sessions or two windows).
- [ ] Heartbeat keeps the connection from going silent.
- [ ] Unauthorized stream 401.

### Phase 3 — Grouping likes (optional but planned)

**Status:** Not started  
**PR:** —

Work:

- Collapse multiple `post_like` on the same post within 24 h into one row with `actors_count` / latest actor; or a `notification_groups` table. If this blows the PR, ship copy-only "and N others" computed at read time from recent likes.

Acceptance:

- [ ] 50 likes do not produce 50 tray rows for the author (cap or group).

If phase 3 is too large, mark it **deferred** in the plan rather than blocking 09. Default: **implement a read-time cap** (show latest like + `likeCount` on the post) and skip a new table.

## Testing plan

- Unit: no self-notify; reply notifies the right people.
- Handler: pagination, cannot read another's id (404).
- Stream: test with httptest SSE or a short integration test.
- Frontend: reconnect; badge a11y (`aria-label` unread count).

## Open questions

- **SSE vs WebSocket in MVP:** Default **SSE** (simpler auth with cookies, one-way is enough). WebSocket gateway later.
- **Email for notifications:** Default **no**.
- **Like grouping table:** Default **read-time aggregation** in phase 3, not a new table.
