# Feature 04: Friends

**Status:** Not started  
**Owner:** —  
**Plan version:** 1

## Goal

People can request, accept, decline, and cancel friendships; unfriend; block; and list friends. The friends graph is the ACL for posts and the source for the news feed.

## Scope

### In scope

- Friend requests: send, accept, decline, cancel (by sender).
- Friends list (paginated).
- Incoming and outgoing pending lists.
- Unfriend.
- Block / unblock. Block implies: no new requests, existing friendship removed, neither can see the other's `friends`/`only_me` content (enforced here for graph; posts/feed consult the same store).
- Mutual-friend count on a profile (optional if cheap; include if a single SQL can do it).

### Out of scope

- Followers / following as a public graph.
- Friend lists / circles (close friends).
- "People you may know" recommendations (can be a later plan).
- Friends of friends visibility as a post privacy (MVP has public / friends / only me only).

## User stories

- As a **user**, I want to add someone as a friend, so that I see their friends-only posts.
- As a **user**, I want to decline or ignore a request.
- As a **user**, I want to unfriend or block someone, so that they leave my graph and cannot bother me.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-04-01 | `POST /v1/friends/requests` with `{ username }` or `{ userId }` creates `pending` if none exists, the pair is not friends, and neither has blocked the other. Idempotent if already pending from the same sender. |
| F-04-02 | Cannot friend yourself. |
| F-04-03 | Accept creates **two** `friendships` rows `(a,b)` and `(b,a)` and marks the request `accepted`. Decline marks `declined`. Cancel by sender marks `cancelled`. |
| F-04-04 | Unfriend deletes both friendship rows. Either party may unfriend. |
| F-04-05 | Block: insert `blocks`, remove friendship if any, cancel pending requests between the pair. Unblock removes the row; it does **not** restore friendship. |
| F-04-06 | Lists: friends, incoming pending, outgoing pending, blocked — cursor pagination. |
| F-04-07 | Friendship is not public API for strangers beyond a `friendCount` on the profile (counts only; no list). Own friends list is private to the owner in MVP (open question). |
| F-04-08 | After accept, emit a domain event in-process so notifications (08) can hook up; until 08, a no-op listener is fine. |

## Data model

### `friend_requests`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | |
| `from_user_id` | `BIGINT NOT NULL` | |
| `to_user_id` | `BIGINT NOT NULL` | |
| `status` | `TEXT NOT NULL` | `pending` \| `accepted` \| `declined` \| `cancelled` |
| `created_at` | `timestamptz NOT NULL` | |
| `responded_at` | `timestamptz NULL` | |

Indexes:

- unique `(from_user_id, to_user_id) WHERE status = 'pending'`
- `(to_user_id, created_at DESC) WHERE status = 'pending'`
- `(from_user_id, created_at DESC) WHERE status = 'pending'`
- check `from_user_id <> to_user_id`

### `friendships`

| Column | Type | Notes |
| --- | --- | --- |
| `user_id` | `BIGINT NOT NULL` | |
| `friend_id` | `BIGINT NOT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- PK `(user_id, friend_id)`
- `(friend_id)` not strictly needed if we always query by `user_id`
- check `user_id <> friend_id`

Always write both directions in one transaction.

### `blocks`

| Column | Type | Notes |
| --- | --- | --- |
| `blocker_id` | `BIGINT NOT NULL` | |
| `blocked_id` | `BIGINT NOT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- PK `(blocker_id, blocked_id)`
- `(blocked_id)` (to answer "did they block me?")

`platform` helper (interface used by profile/post/feed):

```
IsBlocked(ctx, a, b) bool   // either direction
AreFriends(ctx, a, b) bool
```

Redis: `friends:{userId}` as a Redis SET of friend IDs, rebuilt on miss, invalidated on accept/unfriend/block. Optional in phase 2; Postgres is enough for MVP friend counts < 5000.

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/v1/friend-requests` | user | Send request |
| `POST` | `/v1/friend-requests/{id}/accept` | user (recipient) | Accept |
| `POST` | `/v1/friend-requests/{id}/decline` | user (recipient) | Decline |
| `POST` | `/v1/friend-requests/{id}/cancel` | user (sender) | Cancel |
| `GET` | `/v1/friend-requests?box=incoming\|outgoing` | user | Pending lists |
| `GET` | `/v1/friends` | user | Friends of **me** |
| `DELETE` | `/v1/friends/{userId}` | user | Unfriend |
| `GET` | `/v1/blocks` | user | Blocked users |
| `POST` | `/v1/blocks` | user | Body `{ userId }` |
| `DELETE` | `/v1/blocks/{userId}` | user | Unblock |
| `GET` | `/v1/users/{username}/relationship` | user | `none` \| `outgoing` \| `incoming` \| `friends` \| `blocked` |

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `/friends` | Tabs: friends, incoming, outgoing |
| `/friends/requests` | Incoming with accept/decline |
| Profile action button | Add friend / Requested / Accept / Friends / Unfriend / Block |
| `/settings/blocks` | Unblock list |
| Header badge | Incoming request count (can wait for 08) |

## Non-functional and scaling considerations

- Accept/unfriend p95 200 ms including both-direction write.
- List friends p95 100 ms; cursor on `(created_at, friend_id)`.
- At 100M users, average degree is small (hundreds). Store two rows per edge (~200 B) is fine. Hot celebrities: Friendfold is **friends**, not follows, so we still cap **friends** (see open questions) to bound feed fan-out. High-follower **pages** are out of scope; if we later add follow, feed 06's hybrid model applies.
- Do not use recursive CTEs for feed; feed reads this table (or the Redis SET) as a simple `friend_id` list.

## Security and privacy

- All endpoints authenticated. You cannot list another user's friends in MVP (`GET /v1/users/{id}/friends` is out).
- Block is one-way: A blocks B → B sees 404 on A's friends-only content and cannot send requests. A can still see B's **public** posts (like Facebook). Document this; open question if we hide public too.
- Rate limit sends: e.g. 20 pending outgoing, 50 sends/day (share limiter with 09).
- No enumeration via accept on random IDs: 404 if not the recipient.

## Dependencies on other features

- Requires: [03](03-profiles.md) (username to send a request).
- Enables: 05 privacy `friends`, 06 feed, 08 friend-request notifications.

## Implementation phases

### Phase 1 — Requests + accept/decline/cancel + dual friendship rows

**Status:** Not started  
**PR:** —

Work:

- Migrations, OpenAPI, service with transaction, tests for double-accept and self-friend, profile button + `/friends/requests`.

Acceptance:

- [ ] Send → accept produces two rows; both users see each other in `GET /v1/friends`.
- [ ] Duplicate pending request 409 or 200 with same id (document one: **200 idempotent**).
- [ ] Decline does not create friendship.

### Phase 2 — Unfriend, friends list UI, relationship endpoint

**Status:** Not started  
**PR:** —

Work:

- DELETE friend, paginated list, `/friends` page, profile relationship.

Acceptance:

- [ ] Unfriend is symmetric.
- [ ] Cursor pagination stable when a new friend is added.

### Phase 3 — Block / unblock

**Status:** Not started  
**PR:** —

Work:

- `blocks` table, APIs, settings page; `IsBlocked` used by profile GET.

Acceptance:

- [ ] Block removes friendship and pending requests.
- [ ] Blocked user gets 404 on blocker's profile (or a generic profile-unavailable — **404**).
- [ ] Unblock does not restore friendship.

## Testing plan

- Unit: graph invariants (two rows or none), block+request denied, self-friend.
- Handler: authz on accept (only recipient), pagination.
- Frontend: accept/decline from incoming list; profile button state machine.

## Open questions

- **Friend cap:** Default **5000** friends (Facebook-like) to bound fan-out. Confirm before phase 1.
- **Friends list privacy:** Default **only me** (owner sees their list).
- **Public posts from someone you blocked:** Default **still visible** if you navigate to them; they cannot see your non-public content. Revisit if abuse warrants hiding.
