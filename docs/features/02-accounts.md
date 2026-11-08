# Feature 02: Accounts

**Status:** In progress (phase 3)  
**Owner:** —  
**Plan version:** 3

## Goal

A person can create a Friendfold account with email and password or with Google, verify their email, recover a forgotten password, stay signed in across visits, and sign out. This is the identity root for every later feature.

## Scope

### In scope

- Sign up with email + password (argon2id).
- Sign in / sign out.
- Google OAuth 2.0 (authorization code + PKCE at the browser, code exchange on the API).
- Email verification (link with a hashed one-time token; Mailpit in dev).
- Password reset (same token pattern).
- Server-side sessions in Redis; session cookie.
- Basic account settings: change password (when a password exists), add a password to a Google-only account.
- Soft-delete / deactivate hook in the schema (`deleted_at`) even if the full "download my data / delete my account" UI is a last phase.

### Out of scope

- Magic-link-only login, phone numbers, passkeys (later).
- 2FA/TOTP.
- Linking multiple Google accounts.
- Using government ID photos to verify identity (forbidden under Vietnam PDPD Art. 29).
- People search, username (that is [03](03-profiles.md); sign-up collects a display name and a unique email only. Username can be reserved as a nullable unique column here or added in 03 — **add the column in 03** to keep this PR small).

## User stories

- As a **visitor**, I want to register with email and password, so that I can join.
- As a **visitor**, I want to continue with Google, so that I do not invent another password.
- As a **new user**, I want to verify my email, so that I prove I own it.
- As a **user**, I want to reset my password if I forget it.
- As a **user**, I want to stay signed in on my laptop until I sign out or the session expires.
- As a **user**, I want to sign out on this device.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-02-01 | `POST /v1/auth/register` creates a user + password hash, starts a session, and sends a verification email. Password min 8 characters, max 72 bytes (hash input bound), not the email itself. |
| F-02-02 | Passwords stored with **argon2id** (parameters documented in code constants, e.g. memory ≥ 19 MiB, time ≥ 2, parallelism ≥ 1). |
| F-02-03 | `POST /v1/auth/login` authenticates email+password, creates a session, rejects wrong password with the same 401 shape as unknown email (no user enumeration). |
| F-02-04 | `POST /v1/auth/logout` revokes the current session. |
| F-02-05 | `GET /v1/auth/me` returns the current user (id, email, emailVerified, displayName). |
| F-02-06 | Google: `GET /v1/auth/google/start` returns the authorization URL (or 302); `GET /v1/auth/google/callback` upserts `oauth_accounts`, creates user if needed, starts session. |
| F-02-07 | Verification email contains a single-use token (≥ 32 bytes, stored hashed, 24h TTL). `POST /v1/auth/verify-email` marks `email_verified_at`. Unverified users can still use the product in MVP. Password reset **does** require a verified email, or we send the reset only if the email exists **and** is verified — still no enumeration (phase 3). |
| F-02-08 | Password reset: request always returns 202; if the email exists and is verified, send a link. Token 1 hour TTL, single use. Completing reset revokes **all** sessions. |
| F-02-09 | Sessions: opaque ID in cookie `friendfold_session`, Redis key `session:{id}` → `{userId, createdAt, userAgent, ip}`, TTL 30 days, sliding refresh at most once per hour on authenticated requests. |
| F-02-10 | Cookie flags: `HttpOnly; Secure; SameSite=Lax; Path=/`. `Secure` off only when `COOKIE_SECURE=false` (local HTTP). |
| F-02-11 | Age gate: date of birth collected at sign-up, must be **13+**. Store `date_of_birth` as `date` (personal data — minimize access). |
| F-02-12 | Sign-up requires acknowledgement of Privacy Policy and Terms (`terms_accepted_at`). |
| F-02-13 | Rate limit: register/login/reset/verify/resend per IP and per email hash or user id (see [09](09-safety-basics.md) for the shared limiter; until 09 ships, implement a Redis sliding window in `platform` and use it here). Phase 2: verify 20/15min per IP; resend 3/15min per IP and per user id. Phase 3: forgot 5/15min per IP and 3/15min per email hash; reset 20/15min per IP; change 5/15min per IP and per user id. |

## Data model

### `users`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | Snowflake |
| `email` | `TEXT` | Unique among non-deleted; store lowercased (no `citext`) |
| `email_verified_at` | `timestamptz NULL` | |
| `password_hash` | `TEXT NULL` | argon2id encoded hash; NULL if Google-only so far |
| `display_name` | `TEXT NOT NULL` | 1–80 chars; profile 03 may split this |
| `date_of_birth` | `DATE NOT NULL` | |
| `terms_accepted_at` | `timestamptz NOT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |
| `updated_at` | `timestamptz NOT NULL` | |
| `deleted_at` | `timestamptz NULL` | soft delete |

Indexes:

- unique `(email) WHERE deleted_at IS NULL` (`users_email_live_key`)
- `(id)` PK

sqlc: `backend/db/queries/auth.sql` generates into `internal/auth/sqlc` (a second sqlc package). Platform `ping.sql` is generated separately so the kernel does not import the `users` table.

### `oauth_accounts`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | |
| `user_id` | `BIGINT NOT NULL` | FK users |
| `provider` | `TEXT NOT NULL` | `google` |
| `provider_user_id` | `TEXT NOT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- unique `(provider, provider_user_id)`
- `(user_id)`

### `auth_tokens`

Used for email verification and password reset.

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | |
| `user_id` | `BIGINT NOT NULL` | FK `users(id)` ON DELETE CASCADE |
| `purpose` | `TEXT NOT NULL` | `verify_email` \| `reset_password` (CHECK) |
| `token_hash` | `BYTEA NOT NULL` | SHA-256 (32 bytes) of the presented secret; never the raw token |
| `expires_at` | `timestamptz NOT NULL` | verify_email: 24h; reset_password: 1h (phase 3) |
| `consumed_at` | `timestamptz NULL` | set on use **or** when a newer token of the same purpose is issued |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- unique `(token_hash)` (`auth_tokens_token_hash_key`)
- unique `(user_id, purpose) WHERE consumed_at IS NULL` (`auth_tokens_unused_purpose_key`) — at most one unused token per purpose per user; serializes concurrent resend

The presented secret is 32 cryptographically random bytes, encoded unpadded base64url (~43 chars) in the email URL. Only the SHA-256 of that exact string is stored. `CHECK (octet_length(token_hash) = 32)`.

Redis:

- `session:{sid}` → JSON, TTL 30d
- `rate:auth:{action}:{ip}` and `rate:auth:{action}:{emailHash}` → counters
- `rate:auth:verify:{ip}` — verify-email
- `rate:auth:resend:{ip}` and `rate:auth:resend:user:{userId}` — resend-verification
- `rate:auth:forgot:{ip}` and `rate:auth:forgot:{emailHash}` — password forgot
- `rate:auth:reset:{ip}` — password reset
- `rate:auth:change:{ip}` and `rate:auth:change:user:{userId}` — password change
- `session_gen:{userId}` — integer generation; `INCR` revokes every existing session for that user (phase 3)

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/v1/auth/register` | public | Create account, set session cookie. Body: `{ email, password, displayName, dateOfBirth, termsAccepted }`. 201 `User`. Errors: 400 `validation_error` / `underage` / `terms_required`, 403 `origin_mismatch`, 409 `email_taken`, 429 `rate_limited`. |
| `POST` | `/v1/auth/login` | public | Password login. Body: `{ email, password }`. 200 `User`. Unknown vs bad password both 401 `invalid_credentials`. |
| `POST` | `/v1/auth/logout` | user | Revoke session, clear cookie. 204. |
| `GET` | `/v1/auth/me` | user | Current user: `{ id, email, emailVerified, displayName }`. DOB is not returned. |
| `GET` | `/v1/auth/google/start` | public | Begin OAuth (phase 4) |
| `GET` | `/v1/auth/google/callback` | public | Finish OAuth, set cookie, redirect to web (phase 4) |
| `POST` | `/v1/auth/verify-email` | public | Body: `{ token }`. 204 on success. Unknown, expired, consumed, and malformed tokens all 400 `invalid_token` (empty token is 400 `validation_error`). |
| `POST` | `/v1/auth/resend-verification` | user | Invalidate unused `verify_email` tokens, issue a new one, enqueue mail. 204 even if already verified (no-op, no email). |
| `POST` | `/v1/auth/password/forgot` | public | Body: `{ email }`. Always **202** for well-formed email (unknown, unverified, verified, deleted). Sends a reset mail **only** if the live user exists **and** `email_verified_at` is set. Origin required. |
| `POST` | `/v1/auth/password/reset` | public | Body: `{ token, newPassword }`. Consumes a `reset_password` token, writes a new argon2id hash, revokes **all** sessions. 204. Unknown/expired/consumed/wrong-purpose tokens are 400 `invalid_token` (empty token is 400 `validation_error`). No session is created; the user signs in afterward. |
| `POST` | `/v1/auth/password/change` | user | Body: `{ currentPassword?, newPassword }`. When `password_hash` is set, `currentPassword` is required and must match (wrong → 401 `invalid_credentials`). When `password_hash` is NULL (Google-only, phase 4), omit `currentPassword` to add a password. Revokes all sessions then issues a new session cookie on this device. 204. |

`User.id` is a Snowflake decimal **string**. Register/login set `Set-Cookie: friendfold_session`. Mutating `/v1/auth/*` require `Origin` matching `WEB_ORIGIN`. Precise schemas live in `backend/api/openapi.yaml`.

CSRF: mutating `/v1/auth/*` that rely on cookies require Origin matching `WEB_ORIGIN`. Login/register send credentials in JSON body (not cookies) then **set** the cookie.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `/signup` | Email, password, display name, DOB, terms checkbox (Google button is phase 4) |
| `/login` | Email, password, link to sign up, **Forgot password?** (phase 3). Google is phase 4. |
| `/forgot-password` | Request reset. Always the same success copy (no enumeration). |
| `/reset-password?token=` | New password + confirm. Missing/invalid token shows an error; success sends the user to `/login`. |
| `/change-password` | Signed-in form: current password + new password. Link from the header when a session exists. Google-only add-password UI waits for phase 4. |
| `/verify-email?token=` | Reads `token` from the query, `POST /v1/auth/verify-email`, then redirects home on 204. Missing/invalid token shows an error; already-signed-in users refresh `GET /me` so the banner clears. |
| `AuthProvider` / layout gate | Redirect anonymous users away from `(main)` once that group exists |
| Header `Sign in` / `Sign out` / `Change password` | Change password is phase 3, signed-in only. |
| `UnverifiedEmailBanner` | Shown on the root chrome when `user && !user.emailVerified`: "Check your email" plus resend. |

Unverified banner on the main chrome: "Check your email" with resend. Unverified users remain allowed in MVP (F-02-07).

## Non-functional and scaling considerations

- Register p95 < 200 ms excluding SMTP (enqueue mail on a bounded worker pool; **do not fail sign-up if Mailpit/SES is down** — log without the token/body and allow resend). Pool size 8; if workers are busy or the process is shutting down, drop the send (resend exists). `cmd/api` waits for in-flight sends on SIGINT/SIGTERM with the HTTP shutdown budget.
- Redis session store is required. Plan for 100M users × 1.5 sessions ≈ 150M keys; value < 512 bytes; eviction is not acceptable for sessions — size Redis accordingly later. TTL keeps the set to DAU × devices, not all-time users.
- argon2id is CPU-heavy: run it in a bounded worker pool so a login flood cannot starve the process; pair with rate limits.
- Sharding key is `users.id`. Email uniqueness becomes a lookup problem after shard — acceptable later with a global email directory service; do not over-design now.

## Security and privacy

- Constant-time compare of password hashes (argon2 library default).
- Tokens in emails are random 32+ bytes, stored hashed; the email has the raw token once.
- Do not log passwords, tokens, or session IDs. Log `user_id` after auth.
- Same error for login miss vs bad password: `401 invalid_credentials`.
- Google: validate `state`, `nonce`/ID token audience, hosted domain not required.
- Personal data: email, DOB, IP, user agent. DOB is not shown on profile. Legal basis: consent + contract. Vietnam PDPD + GDPR access/erasure apply (deletion UI may be phase 5).
- Age 13+: reject sign-up with 400 `underage`. Parental consent for 13–16 in Vietnam is an **open question**; until answered, still allow 13+ and record DOB.

## Dependencies on other features

- Requires: [01](01-project-foundation.md) shipped.
- Enables: 03–09.
- Rate-limit middleware may be introduced here and reused in 09.

## Implementation phases

### Phase 1 — Email/password register, login, logout, me, sessions

**Status:** Shipped  
**PR:** split into five stacked PRs (one concern each); merge in this order:

1. https://github.com/anhtuta/friendfold/pull/19 — OpenAPI register / login / logout / me
2. https://github.com/anhtuta/friendfold/pull/20 — `users` table migration + sqlc
3. https://github.com/anhtuta/friendfold/pull/21 — platform sessions, cookies, Origin check, rate limit
4. https://github.com/anhtuta/friendfold/pull/22 — auth module (argon2id, handlers)
5. https://github.com/anhtuta/friendfold/pull/23 — frontend `/signup`, `/login`, sign-out, `GET /me`

Work:

- Migrations for `users` (password + email columns; no oauth/tokens yet).
- argon2id, Redis sessions, cookie, CORS/Origin check, `auth` module, OpenAPI paths for register/login/logout/me.
- Frontend `/signup`, `/login`, sign-out, `GET /me` on load.

Acceptance:

- [x] Register + login + me + logout round-trip in tests and in the browser.
- [x] Password not stored plaintext; cookie HttpOnly.
- [x] Unknown vs bad password both 401.

Decisions / deviations (phase 1):

- **Verification email is phase 2.** `POST /v1/auth/register` in phase 1 creates the user and session only (`emailVerified: false`). F-02-01's "sends a verification email" waits for phase 2 so this phase stays one concern.
- **Google button and forgot-password link** wait for phases 4 and 3. `/signup` and `/login` are email/password only.
- **Mutating `/v1/auth/*` require `Origin` matching `WEB_ORIGIN`**, including register/login (not only cookie-authenticated writes). Browser `fetch` always sends Origin; curl must pass `-H Origin: …`.
- Email column is `TEXT` stored lowercased (no `citext` extension). Unique `(email) WHERE deleted_at IS NULL`.
- **`COOKIE_SECURE`:** explicit `true`/`false` overrides. When unset, `Secure` follows `WEB_ORIGIN` (`https` → true, `http` → false) so local `go run` works without a `.env`.
- **Client IP** for rate limits is `RemoteAddr` only. `X-Forwarded-For` is not trusted until a reverse-proxy config exists.
- Rate limiter is a Redis sliding window in `internal/platform/ratelimit` (ZSET + Lua). Auth keys: `rate:auth:{action}:{ip}` and `rate:auth:{action}:{emailHash}`. Limits (phase 1): register 5/15min per IP and 3/15min per email hash; login 10/15min per IP and per email hash.
- argon2id PHC encoding `$argon2id$v=19$m=19456,t=2,p=1$…` (19 MiB, time 2, parallelism 1). Hashing runs on a bounded worker pool sized to `GOMAXPROCS`. Rate-limit Redis errors are logged and **fail open**; session Redis errors fail the request. Go request/response structs match the OpenAPI field names (codegen still deferred; health/ready `const` schemas were a poor fit for oapi-codegen in foundation).

### Phase 2 — Email verification

**Status:** Shipped  
**PR:** split into six stacked PRs (one concern each); merged in this order:

1. https://github.com/anhtuta/friendfold/pull/26 — docs (phase-2 breakdown and decisions)
2. https://github.com/anhtuta/friendfold/pull/27 — OpenAPI `POST /v1/auth/verify-email` and `POST /v1/auth/resend-verification`
3. https://github.com/anhtuta/friendfold/pull/28 — `auth_tokens` goose migration + sqlc
4. https://github.com/anhtuta/friendfold/pull/29 — platform SMTP sender + Mailpit config
5. https://github.com/anhtuta/friendfold/pull/30 — auth module: hashed tokens, verify/resend, register enqueues mail
6. https://github.com/anhtuta/friendfold/pull/31 — frontend `/verify-email`, unverified banner, resend

Work:

- `auth_tokens` table (hashed one-time tokens; purpose includes `reset_password` for phase 3 but only `verify_email` is issued here).
- SMTP sender in `internal/platform/mail` (Mailpit on `:1025` / UI `:8025` in Compose). From-address and host from env; no production credentials in git.
- `POST /v1/auth/verify-email` (public) and `POST /v1/auth/resend-verification` (session).
- Register creates the user and session first, then issues a token and enqueues mail. SMTP/queue failure must not undo signup.
- Frontend `/verify-email?token=` and unverified banner with resend.

Acceptance:

- [x] Mailpit round-trip is implemented and **skipped** when SMTP `:1025` is unreachable (this environment had no Docker/Mailpit). Token verifies once; second use 400 `invalid_token`.
- [x] Resend **invalidates previous unused tokens of the same purpose**; the old link is 400.
- [x] Expired token is 400 `invalid_token` (same body as unknown/consumed).
- [x] SMTP failure / full queue does not undo signup (201 + session still happen).
- [x] Mutating verify/resend require Origin matching `WEB_ORIGIN`; resend requires a session.
- [x] Unverified users can still use the product; banner only.
- [x] No raw tokens, passwords, or session IDs in logs (queue tests assert body/recipient are absent).
- [x] `/verify-email` and the unverified banner are covered by frontend tests and dark-theme screenshots.

Decisions / deviations (phase 2):

- **Token TTL:** 24 hours for `verify_email` (password reset stays 1 hour in phase 3). Hardcoded constant, not env.
- **Token encoding:** 32 random bytes → unpadded base64url in the link; store SHA-256 of that presented string as `BYTEA` (32 bytes). Lookup hashes the submitted token; empty token is `validation_error` without a lookup.
- **Consume / resend races:** consume is a single `UPDATE … WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > $now RETURNING`. Resend runs in a transaction: consume unused rows for `(user_id, purpose)`, then insert. The partial unique index `auth_tokens_unused_purpose_key` makes two concurrent resends serialize (second waits, invalidates the first's new row, inserts its own). Verify of the same token: only one UPDATE matches.
- **Indexes vs the original sketch:** unique `(user_id, purpose)` among unused rows replaces the non-unique `(user_id, purpose)` index. Unique `token_hash` is unchanged.
- **Already verified:** resend returns 204 and does not issue a token or send mail. Verify of a still-unused unexpired token still consumes it and is 204 (idempotent `email_verified_at`).
- **HTTP:** both endpoints return **204** on success (no user body; `/me` is the read path). Verify is public (the token is the capability). Resend is cookie-authenticated.
- **Rate limits** (Redis sliding window, 15 min, fail-open like phase 1): verify-email **20/IP**; resend **3/IP** and **3/user id**. Register/login limits unchanged. Verify is not limited per email (the body has no email).
- **SMTP config (local defaults, names in `.env.example`):** `SMTP_HOST=127.0.0.1`, `SMTP_PORT=1025`, `SMTP_USERNAME` / `SMTP_PASSWORD` empty (Mailpit has no auth), `SMTP_FROM=Friendfold <friendfold@localhost>`, `SMTP_STARTTLS=false`. Optional `MAILPIT_URL=http://127.0.0.1:8025` for the round-trip test. Production SES/SMTP is later env, not this phase.
- **Mail workers:** `internal/platform/mail` queue, 8 in-flight sends, 10s per send, drop when saturated or shutting down. Process context (not the HTTP request context) so signup returning does not cancel the send. Shutdown-aware: `cmd/api` closes the queue after HTTP shutdown.
- **Email link:** `{WEB_ORIGIN}/verify-email?token={token}` (web origin, not the API). Plain text only.
- **Unverified users remain allowed.** Banner + resend only; posting/password-reset gates wait for later features.
- **No ADR.** Hashed email tokens in Postgres were already accepted in [0003](../decisions/0003-postgres-redis.md). TTL, rate limits, and SMTP env are recorded here.
- **Out of scope:** password reset, Google OAuth, requiring verified email to use the product, production mail credentials.

### Phase 3 — Password reset and change

**Status:** In progress  
**PR:** split into six stacked PRs (one concern each); merge **after** phase 2 (`main` includes #26–#31). Do not start from phase-2 feature branches. Merge order:

1. https://github.com/anhtuta/friendfold/pull/32 — docs (phase-3 breakdown and decisions)
2. https://github.com/anhtuta/friendfold/pull/33 — OpenAPI `POST /v1/auth/password/forgot`, `/reset`, `/change`
3. https://github.com/anhtuta/friendfold/pull/34 — platform session generation (`session_gen:{userId}`)
4. https://github.com/anhtuta/friendfold/pull/35 — auth module: forgot / reset / change
5. https://github.com/anhtuta/friendfold/pull/36 — frontend `/forgot-password`, `/reset-password`, login link
6. https://github.com/anhtuta/friendfold/pull/37 — frontend `/change-password` and signed-in header link

Work:

- Reuse `auth_tokens.purpose = reset_password` (table and CHECK already shipped in phase 2). No new migration.
- Reuse `internal/platform/mail` + Mailpit. Link is `{WEB_ORIGIN}/reset-password?token=`. Plain text. SMTP/queue failure on forgot still returns 202 (same as register: do not fail the HTTP status; the user can request again).
- `POST /v1/auth/password/forgot` always 202 for a well-formed email. Issue + send **only** when the live user exists and `email_verified_at` is set (F-02-08). Unknown, unverified, and deleted addresses get the same 202 and no mail. A verified Google-only row (`password_hash` NULL) still gets a reset mail so reset can set a password.
- Reset token: 32 random bytes, unpadded base64url, SHA-256 stored, **1 hour** TTL, single use. Issuing a new `reset_password` token invalidates unused tokens of that purpose (existing `IssueToken` tx).
- `POST /v1/auth/password/reset` hashes the new password on the argon2id pool **before** consume, bumps session generation, then in one Postgres transaction consumes the `reset_password` token and writes `password_hash`. Wrong-purpose tokens (`verify_email`) are 400 `invalid_token`. No session cookie is set.
- `POST /v1/auth/password/change` (session): when `password_hash` is set, `currentPassword` is required and compared (wrong → 401 `invalid_credentials`, dummy argon2 when the user row is missing after session). When `password_hash` is NULL, omit `currentPassword` to **add** a password (Google-only path; no OAuth in this phase). Then bump generation, update hash, create a **new** session and set the cookie (this device stays signed in; others die).
- Origin on all three POSTs. No passwords, tokens, or session IDs in logs.

Acceptance:

- [x] Forgot always 202 for eligible, unknown, and unverified addresses; mail is sent only for a live verified user.
- [x] Forgot is rate limited per IP and per email hash; 429 does not encode whether the email exists.
- [x] Reset with a valid token writes a new argon2id hash; the old password fails login.
- [x] Expired, reused, empty, malformed, and `verify_email`-purpose tokens are 400 (`invalid_token` or `validation_error` for empty) with the same public body.
- [x] Reset revokes **all** sessions for that user, including concurrent sessions on other devices; the reset response does not set a session.
- [x] A second forgot invalidates the previous unused `reset_password` token.
- [x] Change requires `currentPassword` when a password is set; wrong current password is 401 `invalid_credentials`.
- [x] Change with NULL `password_hash` and omitted `currentPassword` sets a password (add-password; covered in backend tests without Google).
- [x] Successful change keeps this device signed in (new session cookie) and invalidates every other session.
- [x] If session revoke (`INCR`) fails, the handler does not consume the token or write the new hash.
- [x] `/forgot-password`, `/reset-password`, `/change-password`, and the login forgot link are covered by frontend tests and dark-theme screenshots.
- [x] OpenAPI examples use `example-reset-token` (not high-entropy fakes). Tests mint secrets at runtime.

Decisions / deviations (phase 3):

- **No new tables.** `reset_password` purpose, unique unused `(user_id, purpose)`, and hashed `token_hash` already exist. sqlc gains `UpdatePasswordHash` only.
- **Forgot is 202, not 204.** Matches F-02-08. The body is empty. Well-formedness errors (empty/invalid email, bad JSON) stay 400 `validation_error` — that does not enumerate accounts. Origin mismatch stays 403.
- **Unverified addresses get no mail.** F-02-07 / F-02-08: reset is a verified-email capability. Returning 202 still hides whether the address exists or is unverified.
- **Google-only / NULL password on forgot:** if the user is verified but `password_hash` is NULL, still send a reset (reset **sets** a password). That is the unauthenticated add-password path after OAuth exists. Until phase 4 there is no such row in production; tests may insert one.
- **Token TTL:** 1 hour, hardcoded (`resetTTL`), not env. Same 32-byte unpadded base64url + SHA-256 helper as verify.
- **Session revoke = generation, not a Redis SET of session ids.** Each session record stores `generation` (JSON `gen`) copied from `session_gen:{userId}` (missing key = 0) at `Create`. `Get` / `GetAndTouch` compare to the current generation; mismatch is `ErrNotFound` and the stale key is deleted. `BumpGeneration` is `INCR`. Dead keys expire with the 30-day TTL. This avoids `SCAN` and a secondary index that would leak ids when sessions expire. **No ADR** — Redis sessions were accepted in [0003](../decisions/0003-postgres-redis.md); generation is how F-02-08 is implemented.
- **Order for reset (security over UX):** (1) validate token shape, (2) hash new password on the bounded argon2id pool, (3) `BumpGeneration`, (4) Postgres tx: consume `reset_password` token + `UpdatePasswordHash`. If (3) fails, return 500 and leave the token unused. If (4) fails after (3), the user is signed out everywhere and the token remains usable — they retry. Never consume/write the hash if revoke did not succeed.
- **Order for change:** verify current password (argon2id pool), hash new, `BumpGeneration`, update hash, `Create` a new session, set cookie. If `Create` fails after bump+update, return 500; the user signs in with the new password (all sessions already dead).
- **Change keeps this device.** Reset does **not** sign the user in. OWASP “invalidate existing sessions after password change” is the bump; the new cookie is only for the authenticated change call.
- **Add-password (Google-only):** API supports it now so phase 4 does not change the contract. No Google UI, no `oauth_accounts`. Frontend `/change-password` always shows current-password (every phase-1/2 user has a hash).
- **Rate limits** (15 min, fail-open like phase 1–2): forgot **5/IP** and **3/email hash**; reset **20/IP**; change **5/IP** and **5/user id**. Forgot 429 is the same for every address.
- **Dummy argon2:** change with a missing user after a valid-looking session still `CompareDummy` before 401. Forgot does **not** run argon2 (no password in the body); equalized work is one indexed email lookup plus 202.
- **invalid_token copy:** reset uses “This reset link is invalid or has expired” (verify keeps its own message). Codes stay `invalid_token`.
- **Email copy:** “Reset your Friendfold password” + link + 1 hour expiry + ignore-if-you-did-not-request. No raw token in logs (queue tests already forbid body/recipient).
- **Out of scope:** Google OAuth, requiring verified email to use the product, production mail credentials, listing or browsing other devices’ sessions, 2FA.

### Phase 4 — Google OAuth

**Status:** Not started  
**PR:** —

Work:

- `oauth_accounts`, start + callback, env `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `GOOGLE_REDIRECT_URI`.
- If Google email matches an existing user, **link** (same email, verified by Google) rather than creating a duplicate.

Acceptance:

- [ ] Happy path against Google test client (or recorded HTTP in tests).
- [ ] Linking by verified email does not hijack an unverified local email: only auto-link if the local email is already verified **or** we set it verified from Google's `email_verified`.

### Phase 5 — Account deletion request (minimum viable privacy)

**Status:** Not started  
**PR:** —

Work:

- `POST /v1/auth/account/delete` sets `deleted_at`, revokes sessions, anonymizes email to a tombstone (`deleted+{id}@invalid.local`) so the unique index allows reuse of the real email.
- Settings UI: "Delete account" with confirm.

Acceptance:

- [ ] Deleted user cannot log in.
- [ ] Email can be registered again.
- [ ] Spec and privacy policy text mention the 30-day restoration window **or** immediate deletion — **default: immediate soft-delete, hard-delete job later**.

## Testing plan

- Unit: argon2id hash/verify, token hash (raw secret not stored), session TTL, age gate (12 vs 13), SMTP client against a loopback listener.
- Handler: register validation, login enumeration, CSRF origin reject, verify once then 400, expiry, resend invalidates previous unused token, SMTP/queue failure does not undo signup, resend without session 401, verify/resend origin reject, resend rate limit, forgot always 202 (unknown/unverified/verified), forgot sends only if verified, reset once then 400, reset wrong-purpose 400, reset expires, reset invalidates unused on second forgot, reset old password fails and all sessions die, change requires current when set, change add-password when hash is NULL, change re-issues this session and kills others.
- Frontend: signup validation errors, login error, logout, verify-email missing/invalid/success, unverified banner resend, forgot success copy, reset missing/invalid/success, change current-password error and success.
- Mailpit: optional round-trip (`MAILPIT_URL` / SMTP `:1025`); skip when unreachable. Manual: Compose Mailpit UI on `:8025`.
- Google on a deployed preview if secrets exist (otherwise mock) — phase 4.

## Open questions

- **Parental consent (Vietnam, 13–16):** Default until legal review: allow 13+, store DOB, do not add a parent-email flow in MVP.
- **Session length:** 30 days sliding vs 7 days. Default: **30 days**.
- **Must email be verified before posting?** Default: **no** for MVP, banner only; safety 09 can tighten.
- **Display name vs username at sign-up:** display name here; unique username in feature 03.
