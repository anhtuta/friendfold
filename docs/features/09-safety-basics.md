# Feature 09: Safety basics

**Status:** Not started  
**Owner:** —  
**Plan version:** 1

## Goal

Users can report a post or a person. Staff can review those reports in a gated admin view and resolve them (dismiss, hide post, suspend user). Abuse is slowed with **rate limits** on the endpoints that attract spam. This is not a full trust-and-safety platform.

## Scope

### In scope

- Report post / report user with a reason enum + optional details.
- Admin list of open reports, detail, resolve with a note.
- Hide a post (`posts.hidden_at` or reuse `deleted_at` with a `deleted_reason=staff` — **prefer `hidden_at` + `hidden_by`** so owner delete and staff hide stay distinct).
- Suspend a user (`users.suspended_at`) so they cannot sign in.
- Rate limiting: shared Redis sliding window used by auth, friends, posts, comments, reports (replace ad-hoc limits from earlier features with one `platform/ratelimit` if not already unified).
- Staff role: `users.role` `user` \| `admin`. Seed one admin via env `BOOTSTRAP_ADMIN_EMAIL` in non-prod; production grants via SQL/runbook, not a public API.

### Out of scope

- Automated malware/CSAM classifiers, photo hashing, legal hold, appeals workflow, IP bans UI, report-a-comment (can add as a small extra if cheap — **default include comment reports** if 07 exists, as a third target type).
- Public "community standards" CMS. A static `/safety` page with English copy is enough.
- Full GDPR/PDPD case-management console (72-hour restriction requests can be a mailbox in MVP).

## User stories

- As a **user**, I want to report a post or a person, so that staff can look at it.
- As a **staff member**, I want a queue of open reports and actions to hide or suspend.
- As an **operator**, I want expensive endpoints to shed load when someone scripts them.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-09-01 | Authenticated user can `POST /v1/reports` with `targetType` (`post` \| `user`), `targetId`, `reason`, optional `details` ≤ 1000 chars. |
| F-09-02 | Reasons: `spam`, `harassment`, `hate`, `nudity`, `violence`, `impersonation`, `other`. |
| F-09-03 | Duplicate open report by the same reporter on the same target is idempotent (return existing). |
| F-09-04 | Reporter cannot report themselves or their own post (400). |
| F-09-05 | Admin `GET /v1/admin/reports?status=open` paginated; `GET /v1/admin/reports/{id}` includes target snapshot (post body/url or user profile). |
| F-09-06 | `POST /v1/admin/reports/{id}/resolve` with `action`: `dismiss` \| `hide_post` \| `suspend_user`. Sets `status=resolved`, `resolved_by`, `resolved_at`, `note`. |
| F-09-07 | Hidden posts 404 for everyone except admin and (optionally) the author — **author sees a "removed" placeholder; others 404**. |
| F-09-08 | Suspended user: login 403 `account_suspended`; sessions revoked. |
| F-09-09 | Rate limits use a shared Redis sliding window. Return `429` with `Retry-After`. Anonymous endpoints keyed by IP; authenticated by `userId` and IP. Starting values are in the table below. |
| F-09-10 | Admin routes require `role=admin`; 404 for non-admin (do not advertise the admin surface). |

Rate-limit starting values:

| Action | Limit |
| --- | --- |
| Register / IP | 5 / hour |
| Login / IP | 20 / 15 min |
| Password forgot / email+IP | 3 / hour |
| Friend request send / user | 50 / day |
| Post create / user | 20 / hour |
| Comment create / user | 30 / hour |
| Report create / user | 10 / hour |
| Media upload / user | 60 / hour |

## Data model

### `users` columns added

| Column | Type | Notes |
| --- | --- | --- |
| `role` | `TEXT NOT NULL DEFAULT 'user'` | `user` \| `admin` |
| `suspended_at` | `timestamptz NULL` | |
| `suspended_reason` | `TEXT NULL` | internal |

### `posts` columns added

| Column | Type | Notes |
| --- | --- | --- |
| `hidden_at` | `timestamptz NULL` | |
| `hidden_by` | `BIGINT NULL` | admin user id |

Treat `hidden_at IS NOT NULL` like not viewable in `CanView` (except admin tools).

### `reports`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | |
| `reporter_id` | `BIGINT NOT NULL` | |
| `target_type` | `TEXT NOT NULL` | |
| `target_id` | `BIGINT NOT NULL` | |
| `reason` | `TEXT NOT NULL` | |
| `details` | `TEXT NOT NULL DEFAULT ''` | |
| `status` | `TEXT NOT NULL` | `open` \| `resolved` |
| `resolution` | `TEXT NULL` | `dismiss` \| `hide_post` \| `suspend_user` |
| `resolved_by` | `BIGINT NULL` | |
| `resolved_at` | `timestamptz NULL` | |
| `note` | `TEXT NOT NULL DEFAULT ''` | staff |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- `(status, created_at DESC)` admin queue
- unique `(reporter_id, target_type, target_id) WHERE status = 'open'`
- `(target_type, target_id)`

Redis rate-limit keys: `rl:{action}:{subject}` with sliding window or token bucket. Implementation in `internal/platform/ratelimit`.

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/v1/reports` | user | Create report |
| `GET` | `/v1/admin/reports` | admin | Queue |
| `GET` | `/v1/admin/reports/{id}` | admin | Detail |
| `POST` | `/v1/admin/reports/{id}/resolve` | admin | Resolve + action |

There is no public "list my reports" in MVP.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `ReportDialog` | Opened from `PostMenu` and profile menu |
| `/safety` | Short English standards + how to report |
| `/admin/reports` | Table of open reports (admin layout, no link in the user header) |
| `/admin/reports/[id]` | Snapshot, actions, note |
| 429 handling | Toast "Too many requests, try later" |

Admin layout: if `me.role !== 'admin'`, render the generic 404 page (same as unknown route).

## Non-functional and scaling considerations

- Report create p95 200 ms.
- Admin queue will be small in MVP; index on `status` is enough. At scale, partition by time and assign reports to queues.
- Rate limiter must fail **closed** on Redis errors for login/register (deny) and fail **open** for read GETs (do not take the site down). Writes that are not auth: fail closed.
- Hide/suspend must be immediate (authz reads the columns; no 5-minute cache of `CanView` without invalidation).

## Security and privacy

- Reports are personal data (who reported whom). Admins are the audience; log admin actions (`slog` with `admin_id`, `report_id`, `action`).
- Do not notify the reported user that they were reported (avoid retaliation). They notice if a post is hidden or they are suspended.
- Vietnam PDPD / GDPR: reporters can be acting in safety interest; still minimize `details` stored; retain reports ≥ 1 year for abuse patterns (open question with legal).
- Staff accounts use the same argon2id login; no shared "admin" password.
- Rate limits are a security control, not a product feature — do not bypass in production with a header.

## Dependencies on other features

- Requires: [02](02-accounts.md) (users, sessions), [05](05-posts.md) (posts to hide). [07](07-likes-and-comments.md) not required unless we add comment targets.
- Rate limiter should already exist in a thin form from 02; this feature **owns** the documented numbers and the admin UI.
- Enables: a livable MVP.

## Implementation phases

### Phase 1 — Platform rate limiter applied to existing write endpoints

**Status:** Not started  
**PR:** —

Work:

- `internal/platform/ratelimit`, Redis sliding window, middleware or service-level checks on the table in F-09-09, OpenAPI `429` responses, frontend toast.
- If 02 already added a limiter, unify here rather than stacking two.

Acceptance:

- [ ] Exceeding login limit returns 429 with `Retry-After`.
- [ ] Tests use a fake clock or low limits via config.
- [ ] Redis down: login denied (fail closed), documented.

### Phase 2 — User reports + Report dialog

**Status:** Not started  
**PR:** —

Work:

- `reports` table, `POST /v1/reports`, dialog on post and profile, `/safety` page, OpenAPI.

Acceptance:

- [ ] User can report a post and another user.
- [ ] Duplicate open report is idempotent.
- [ ] Self-report 400.
- [ ] Non-viewable post 404 (cannot report what you cannot see).

### Phase 3 — Admin queue, hide post, suspend user

**Status:** Not started  
**PR:** —

Work:

- `role`, `hidden_at`, `suspended_at`, admin APIs, `/admin/reports` UI, bootstrap env for local admin, session revoke on suspend, `CanView` honors hidden, login honors suspended.

Acceptance:

- [ ] Non-admin `/admin/reports` is 404.
- [ ] Hide post: strangers 404, author sees removed state, feed skips it.
- [ ] Suspend: user cannot log in; existing cookie dies.
- [ ] Dismiss leaves content up.
- [ ] This feature Status **Shipped** when phase 3 merges.

## Testing plan

- Unit: limiter window; duplicate report; CanView + hidden.
- Handler: admin authz 404, resolve actions side effects.
- Frontend: report dialog validation; admin table empty state.
- Abuse: scripted login attempts hit 429 (integration).

## Open questions

- **Comment reports:** Default **yes**, `targetType=comment`, hide comment via `deleted_at` by staff.
- **Appeal:** Default **no** in MVP; suspended users see a static message with `support@` placeholder.
- **Retention of reports:** Default **indefinite in MVP**, add a 24-month purge later with legal review.
- **How production admins are granted:** Default **manual SQL / ops runbook**, not a self-serve promote API.
