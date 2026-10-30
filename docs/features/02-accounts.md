# Feature 02: Accounts

**Status:** In progress (phase 1)  
**Owner:** —  
**Plan version:** 1

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
| F-02-07 | Verification email contains a single-use token (≥ 32 bytes, stored hashed). `POST /v1/auth/verify-email` marks `email_verified_at`. Unverified users can still use the product in MVP but cannot reset password until verified? **Decision:** they **can** use the product; some future actions may require verified email. Password reset **does** require a verified email, or we send the reset only if the email exists **and** is verified — still no enumeration. |
| F-02-08 | Password reset: request always returns 202; if the email exists and is verified, send a link. Token 1 hour TTL, single use. Completing reset revokes **all** sessions. |
| F-02-09 | Sessions: opaque ID in cookie `friendfold_session`, Redis key `session:{id}` → `{userId, createdAt, userAgent, ip}`, TTL 30 days, sliding refresh at most once per hour on authenticated requests. |
| F-02-10 | Cookie flags: `HttpOnly; Secure; SameSite=Lax; Path=/`. `Secure` off only when `COOKIE_SECURE=false` (local HTTP). |
| F-02-11 | Age gate: date of birth collected at sign-up, must be **13+**. Store `date_of_birth` as `date` (personal data — minimize access). |
| F-02-12 | Sign-up requires acknowledgement of Privacy Policy and Terms (`terms_accepted_at`). |
| F-02-13 | Rate limit: register/login/reset per IP and per email hash (see [09](09-safety-basics.md) for the shared limiter; until 09 ships, implement a Redis sliding window in `platform` and use it here). |

## Data model

### `users`

| Column | Type | Notes |
| --- | --- | --- |
| `id` | `BIGINT PK` | Snowflake |
| `email` | `CITEXT` or `TEXT` | Unique among non-deleted; store lowercased |
| `email_verified_at` | `timestamptz NULL` | |
| `password_hash` | `TEXT NULL` | argon2id encoded hash; NULL if Google-only so far |
| `display_name` | `TEXT NOT NULL` | 1–80 chars; profile 03 may split this |
| `date_of_birth` | `DATE NOT NULL` | |
| `terms_accepted_at` | `timestamptz NOT NULL` | |
| `created_at` | `timestamptz NOT NULL` | |
| `updated_at` | `timestamptz NOT NULL` | |
| `deleted_at` | `timestamptz NULL` | soft delete |

Indexes:

- unique `(email) WHERE deleted_at IS NULL`
- `(id)` PK

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
| `user_id` | `BIGINT NOT NULL` | |
| `purpose` | `TEXT NOT NULL` | `verify_email` \| `reset_password` |
| `token_hash` | `BYTEA NOT NULL` | SHA-256 of the secret |
| `expires_at` | `timestamptz NOT NULL` | |
| `consumed_at` | `timestamptz NULL` | |
| `created_at` | `timestamptz NOT NULL` | |

Indexes:

- `(token_hash)` unique
- `(user_id, purpose)`

Redis:

- `session:{sid}` → JSON, TTL 30d
- `rate:auth:{action}:{ip}` and `rate:auth:{action}:{emailHash}` → counters

## API endpoints

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/v1/auth/register` | public | Create account, set session cookie. Body: `{ email, password, displayName, dateOfBirth, termsAccepted }`. 201 `User`. Errors: 400 `validation_error` / `underage` / `terms_required`, 403 `origin_mismatch`, 409 `email_taken`, 429 `rate_limited`. |
| `POST` | `/v1/auth/login` | public | Password login. Body: `{ email, password }`. 200 `User`. Unknown vs bad password both 401 `invalid_credentials`. |
| `POST` | `/v1/auth/logout` | user | Revoke session, clear cookie. 204. |
| `GET` | `/v1/auth/me` | user | Current user: `{ id, email, emailVerified, displayName }`. DOB is not returned. |
| `GET` | `/v1/auth/google/start` | public | Begin OAuth (phase 4) |
| `GET` | `/v1/auth/google/callback` | public | Finish OAuth, set cookie, redirect to web (phase 4) |
| `POST` | `/v1/auth/verify-email` | public | Body: `{ token }` (phase 2) |
| `POST` | `/v1/auth/resend-verification` | user | Send a new verify email (phase 2) |
| `POST` | `/v1/auth/password/forgot` | public | Always 202 (phase 3) |
| `POST` | `/v1/auth/password/reset` | public | Body: `{ token, newPassword }` (phase 3) |
| `POST` | `/v1/auth/password/change` | user | Body: `{ currentPassword?, newPassword }` (phase 3) |

`User.id` is a Snowflake decimal **string**. Register/login set `Set-Cookie: friendfold_session`. Mutating `/v1/auth/*` require `Origin` matching `WEB_ORIGIN`. Precise schemas live in `backend/api/openapi.yaml`.

CSRF: mutating `/v1/auth/*` that rely on cookies require Origin matching `WEB_ORIGIN`. Login/register send credentials in JSON body (not cookies) then **set** the cookie.

## Frontend pages / components

| Route or component | Responsibility |
| --- | --- |
| `/signup` | Email, password, display name, DOB, terms checkbox, Google button |
| `/login` | Email, password, Google button, link to forgot |
| `/forgot-password` | Request reset |
| `/reset-password?token=` | New password |
| `/verify-email?token=` | Calls verify API, then redirects |
| `AuthProvider` / layout gate | Redirect anonymous users away from `(main)` once that group exists |
| Header `Sign in` / `Sign out` | |

Unverified banner on the main chrome: "Check your email" with resend.

## Non-functional and scaling considerations

- Register p95 < 200 ms excluding SMTP (send mail asynchronously via a goroutine or a later queue; **do not fail sign-up if Mailpit/SES is down** — log and allow resend).
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

**Status:** In progress  
**PR:** split into five stacked PRs (one concern each); merge in this order:

1. OpenAPI register / login / logout / me (this PR)
2. `users` table migration + sqlc
3. Platform sessions, cookies, Origin check, rate limit
4. Auth module (argon2id, handlers)
5. Frontend `/signup`, `/login`, sign-out, `GET /me`

Work:

- Migrations for `users` (password + email columns; no oauth/tokens yet).
- argon2id, Redis sessions, cookie, CORS/Origin check, `auth` module, OpenAPI paths for register/login/logout/me.
- Frontend `/signup`, `/login`, sign-out, `GET /me` on load.

Acceptance:

- [ ] Register + login + me + logout round-trip in tests and in the browser.
- [ ] Password not stored plaintext; cookie HttpOnly.
- [ ] Unknown vs bad password both 401.

Decisions / deviations (phase 1):

- **Verification email is phase 2.** `POST /v1/auth/register` in phase 1 creates the user and session only (`emailVerified: false`). F-02-01's "sends a verification email" waits for phase 2 so this phase stays one concern.
- **Google button and forgot-password link** wait for phases 4 and 3. `/signup` and `/login` are email/password only.
- **Mutating `/v1/auth/*` require `Origin` matching `WEB_ORIGIN`**, including register/login (not only cookie-authenticated writes). Browser `fetch` always sends Origin; curl must pass `-H Origin: …`.
- Email column is `TEXT` stored lowercased (no `citext` extension). Unique `(email) WHERE deleted_at IS NULL`.

### Phase 2 — Email verification

**Status:** Not started  
**PR:** —

Work:

- `auth_tokens`, SMTP sender (Mailpit locally), verify + resend endpoints, `/verify-email` page, unverified banner.

Acceptance:

- [ ] Mailpit shows the message; token verifies once; second use 400.
- [ ] Resend invalidates or coexist with previous tokens (pick one: **invalidate previous unused tokens of the same purpose**).

### Phase 3 — Password reset and change

**Status:** Not started  
**PR:** —

Work:

- forgot/reset/change endpoints and pages; reset revokes sessions.

Acceptance:

- [ ] Forgot always 202.
- [ ] Reset with valid token sets a new hash and invalidates sessions.
- [ ] Change password requires current password when one is set.

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

- Unit: argon2id hash/verify, token hash, session TTL, age gate (12 vs 13).
- Handler: register validation, login enumeration, CSRF origin reject, reset consumes token.
- Frontend: signup validation errors, login error, logout.
- Manual: Mailpit round-trip, Google on a deployed preview if secrets exist (otherwise mock).

## Open questions

- **Parental consent (Vietnam, 13–16):** Default until legal review: allow 13+, store DOB, do not add a parent-email flow in MVP.
- **Session length:** 30 days sliding vs 7 days. Default: **30 days**.
- **Must email be verified before posting?** Default: **no** for MVP, banner only; safety 09 can tighten.
- **Display name vs username at sign-up:** display name here; unique username in feature 03.
