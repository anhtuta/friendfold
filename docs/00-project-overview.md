# Friendfold — project overview

## Vision

Friendfold is a general-audience social network for real people: sign up, keep a profile, add friends, share posts, and see what friends are doing. It is meant to feel familiar to anyone who has used Facebook, without cloning every Facebook product.

It is a real product for real users, designed from day one so the architecture can grow toward **~100 million users**. The first years of traffic will be far smaller; the data model, ID scheme, module boundaries, and feed strategy must not paint us into a corner.

**Tagline:** Into the fold of friends.

**Audience.** Everyone. The first marketing and community push is **Vietnam and Southeast Asia**. The product UI is **English-first**; Vietnamese is planned. Copy, privacy, and data-protection work assume Vietnamese users from the first account.

**Not in the product (now or in the MVP):** a native mobile app, groups, pages, marketplace, stories, DMs / chat, events, a public search of all people, ads. Those, if ever, get their own plans and ADRs.

## MVP scope

A person can:

1. Create an account (email + password or Google), verify email, recover a password, and stay signed in.
2. Set a name, username, avatar, cover photo, and bio, and visit their own and others' profile pages.
3. Send, accept, and decline friend requests; unfriend; block; see a friends list.
4. Publish text and photo posts with privacy (`public`, `friends`, `only me`); edit and delete their posts.
5. Scroll a **newest-first** feed of friends' posts (ranked feed is later).
6. Like posts and comment (including a single level of reply).
7. Get in-app notifications for friend requests, likes, and comments, including realtime delivery while the app is open.
8. Report a post or a user; staff can review reports. Abuse is slowed with rate limits.

Invisible but required: Go API, Postgres, Redis, Next.js app, Docker Compose for local dev, OpenAPI contract, CI, structured logs.

Out of MVP: ranked/For You feed, video, albums, stories, messaging, groups, hashtags, people-you-may-know beyond a trivial stub, push notifications (mobile), full-text search of posts.

Roadmap of the nine build slices, with links, is at the end of this document.

## Functional requirements

These are product requirements for the MVP. Precise fields, endpoints, and phases live in the feature plans.

| ID | Requirement | Plan |
| --- | --- | --- |
| FR-01 | The repo builds locally with Docker Compose (Postgres, Redis, Mailpit), a Go API, a Next.js web app, goose migrations, and GitHub Actions CI. Health endpoints report process liveness. | [01](features/01-project-foundation.md) |
| FR-02 | A visitor can register with email and password or with Google, verify email, log in, log out, and reset a forgotten password. Sessions survive browser restarts until expiry or logout. | [02](features/02-accounts.md) |
| FR-03 | An authenticated user has a profile (display name, unique username, avatar, cover, bio) and a public profile URL. | [03](features/03-profiles.md) |
| FR-04 | Users can send / accept / decline / cancel friend requests, unfriend, block and unblock, and list friends. Blocked users cannot interact. | [04](features/04-friends.md) |
| FR-05 | Users can create, edit, and delete text and photo posts with privacy `public`, `friends`, or `only me`. Viewers only see posts they are allowed to see. | [05](features/05-posts.md) |
| FR-06 | The home feed shows friends' visible posts, newest first, with cursor pagination / infinite scroll. High-follower accounts are planned to use a hybrid fan-out (see NFR and [06](features/06-news-feed.md)). | [06](features/06-news-feed.md) |
| FR-07 | Users can like (or unlike) a post, comment on a post, and reply to a top-level comment. Counts stay consistent. | [07](features/07-likes-and-comments.md) |
| FR-08 | Recipients get in-app notifications for friend requests, likes, and comments, listed in a tray, markable as read, and delivered in realtime while the tab is open. | [08](features/08-notifications.md) |
| FR-09 | Users can report a post or a user. Staff can list, inspect, and resolve reports. Write-heavy and abuse-prone endpoints are rate limited. | [09](features/09-safety-basics.md) |
| FR-10 | HTTP APIs are described in `backend/api/openapi.yaml` before they are implemented. IDs are Snowflake 64-bit values, strings on the wire. | [01](features/01-project-foundation.md), [ADR 0006](decisions/0006-snowflake-ids.md) |

## Non-functional requirements

### Scalability (path to ~100M users)

The MVP runs as **one Go API process** (horizontally replicated) plus **one Next.js deployment**, **Postgres**, and **Redis**. 100M users will not run on that topology, but the following must be true from the first schema:

- **User-keyed data.** Rows that belong to a user carry `user_id` (or `author_id` / `recipient_id`) as a first-class column. Future sharding (Citus or a distributed Postgres-compatible store) is by user ID. Avoid schema designs that require global secondary lookups without an index that can later become a distributed index.
- **IDs are not sequences.** Snowflake IDs are generated in the app, roughly time-ordered, and do not rely on a single Postgres sequence. See [ADR 0006](decisions/0006-snowflake-ids.md).
- **Modules can become services.** Each feature is a Go package under `backend/internal/<feature>` with a service interface. Cross-module calls stay behind those interfaces so they can become gRPC later ([ADR 0004](decisions/0004-modular-monolith.md)).
- **Feed is the hard part.** Chronological friends feed for MVP. **Fan-out-on-write** into per-user Redis timelines for ordinary friend graphs; **fan-out-on-read** (merge at read time) for accounts above a follower/friend threshold so a celebrity post does not write millions of Redis keys. Ranked feed is later and will consume the same timeline + candidate sources. Detail: [06](features/06-news-feed.md).
- **Read replicas** for Postgres as soon as read traffic justifies them; the API must already separate read/write connection pools.
- **Later, not MVP:** Kafka (domain events: `post.created`, `friend.accepted`, …), a WebSocket gateway process, OpenSearch, object storage + CDN for media, Kubernetes (EKS) in **ap-southeast-1 (Singapore)** via Terraform, Cloudflare in front.

Capacity planning targets (order of magnitude, not a promise of day-one hardware): 100M registered users, ~10% DAU, ~20 posts + 100 likes + 20 comments per DAU as a stress shape. Design indexes and Redis key TTLs for that shape; do not provision for it now.

### Performance

Measured at the Go API edge, cached where the plan says to cache, p95 over a 5-minute window in production once we have traffic. Local/dev is exempt but must not be accidentally quadratic.

| Surface | Target (p95) | Notes |
| --- | --- | --- |
| `GET /v1/health` | 20 ms | liveness, no DB |
| Session-authenticated reads that hit Redis (timeline page, session lookup) | 50 ms | |
| Typical Postgres-backed read (profile, single post, friends page) | 100 ms | |
| Home feed first page | 150 ms | hybrid fan-out; see feature 06 |
| Writes (post, like, comment, friend accept) | 200 ms | excluding media upload |
| Image upload (complete, excluding client transfer) | 1 s | transcode may be async; API returns an accepted job or a ready URL per phase plan |
| Next.js first contentful paint for logged-in home, Singapore region | 1.5 s | CDN + SSR; budget, not a lab number |

Pagination is **cursor-based** everywhere. Default page size 20, max 50. No `OFFSET` on unbounded lists.

Hot counters (like count, comment count, friend count, unread notification count) are cached in Redis and reconciled from Postgres.

### Availability

- MVP: **99.9%** monthly for the API and web app, excluding planned maintenance windows announced in status notes.
- Later (multi-AZ EKS): **99.95%**.
- Postgres: automated backups, PITR; Redis: persistence for sessions is required or sessions must be reconstructible; treat Redis as required for sessions and rate limits, optional for cache (cache miss falls back to DB).
- Deploys are rolling. Migrations are expand/contract so old and new API versions can run together.

### Security

- TLS everywhere outside local Compose.
- Argon2id password hashes; Google OAuth for federated login.
- HttpOnly, Secure, SameSite=Lax session cookies; server-side session store in Redis; CSRF protection on cookie-authenticated mutating requests (Origin/Referer check plus SameSite).
- Authorization on every endpoint; privacy-hidden resources return 404.
- Secrets only in the environment; never in git.
- Rate limits on register, login, password reset, friend request, post create, report, and any unauthenticated endpoint.
- Uploads: MIME allowlist, size cap, metadata strip, store off the API host.
- Dependency scanning in CI once foundation lands.
- Admin routes require an `admin` role; there is no hidden backdoor user.

### Privacy and data protection

Friendfold processes personal data of Vietnamese users from day one. Comply with:

- **Vietnam Law on Personal Data Protection** No. 91/2025/QH15 (in force 1 January 2026), including **Article 29** duties for social-network providers.
- **Decree 13/2023/NĐ-CP** on personal data protection, still relevant for consent mechanics and the 72-hour clocks on several data-subject requests.
- **GDPR-style user rights** for anyone using the product, so we do not run a second, weaker privacy stack for non-Vietnamese users.

Concrete product obligations (MVP unless marked later):

| Obligation | Product response |
| --- | --- |
| Be informed; public privacy policy | Privacy policy and cookie notice pages; sign-up does not complete without acknowledgement |
| Consent is opt-in; silence is not consent; purposes are separable | Separate consents for account processing vs optional analytics/marketing (marketing is out of MVP) |
| Access, correction, deletion, copy of one's data | Settings: view/edit profile; later export; delete account (tombstone + scheduled hard delete) — account deletion can land as a phase of [02](features/02-accounts.md) / [03](features/03-profiles.md) if not in the first auth PR |
| Withdraw consent; restrict processing; object to advertising | Restriction/object APIs and 72-hour SLA (staff process is acceptable in MVP; automate later) |
| Cookie refusal; do-not-track or consent-based tracking | No third-party tracking pixels in MVP. First-party session cookies are necessary; document them. If analytics is added, it is opt-in |
| Do not require ID-card photos to verify an account (Law 91/2025 Art. 29) | Email + Google only. Never ask for CCCD/CMND/passport images |
| Protect Vietnamese personal data on cross-border transfer | First region is **Singapore**. Complete a transfer impact assessment before production; store a record of it. Minimize data sent to Google (OAuth) and to object storage |
| Breach notice to authorities within 72 hours when required | Runbook in ops docs (later); structured logs and access logs to support investigation |
| Sensitive personal data | We do not collect health, religion, biometrics, or precise location in MVP. Avatars are not used as biometric templates |
| Minors | Default **13+**. Age gate at sign-up. Parental-consent flow is an open question in [02](features/02-accounts.md); do not silently ship under-16 accounts in Vietnam without a decision |

Data minimization: collect email, name, username, optional bio/photos, friendship graph, posts, reactions, comments, reports, session metadata (IP, user agent, timestamps). Do not collect phone number, government ID, or contacts upload in MVP.

### Observability

- Structured JSON logs from the API (`request_id`, `user_id`, `route`, `status`, `latency_ms`). No PII in log fields beyond user id and (on auth failures) a redacted email hash if needed for abuse.
- Request IDs generated at the edge, returned as `X-Request-Id`, propagated into logs.
- OpenTelemetry traces and metrics when we run on EKS; MVP can expose a Prometheus `/metrics` internally and wrap chi with a tracing middleware stub.
- Next.js: server error reporting once a backend is chosen (open question in foundation).
- SLOs follow the performance table; alert later, log always.

### Accessibility

New pages target **WCAG 2.2 Level AA**: labeled inputs, keyboard operation, visible focus, sufficient contrast, `alt` on images the user did not author (user-authored images use the post text as accessible name when no alt is provided). Do not ship icon-only buttons without an accessible name. The product UI is **dark by default** ([ADR 0008](decisions/0008-dark-theme.md)); text tokens on the page background meet AA.

### Internationalization

- UI language: English. Prepare string extraction so Vietnamese can be added without rewriting components (`next-intl` or equivalent, decided in foundation).
- `lang` on `<html>` matches the active locale. Dates/times use the user's locale and timezone (profile field later; until then, browser locale).
- User-generated content is stored and shown as authored; no mandatory translation.

### Maintainability

- Plan doc per feature; one phase per PR; ADRs for lasting choices; OpenAPI before code; migrations for every schema change.
- golangci-lint + `tsc --noEmit` + tests in CI.
- Module boundaries as in [ADR 0004](decisions/0004-modular-monolith.md). No shared "utils" dumping ground; put helpers in `internal/platform`.
- Conventional Commits.

## High-level architecture

MVP (local and first production):

```
                    ┌─────────────────────────────────────────┐
  Browser           │ Cloudflare (later; TLS + DDoS)          │
    │               └─────────────────────────────────────────┘
    ▼
 Next.js (App Router)          Go API (chi, modular monolith)
    │  SSR / RSC                    │
    │  cookie session               │  /v1/*  from OpenAPI
    └─────────────HTTPS─────────────┤
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
               PostgreSQL         Redis          Mailpit (dev)
               (primary;          sessions,      SMTP catcher
                replicas later)   cache,         (prod: SES or
                                  counters,      equivalent)
                                  rate limits,
                                  timelines
```

Media (from posts/profiles): object storage (MinIO in dev; S3 or Cloudflare R2 in prod) + CDN. The API never serves raw user files from its own disk in production.

Later topology (not MVP, do not build now):

- Kafka (or equivalent) between modules once they split.
- WebSocket gateway for realtime fan-out, subscribed to notification events.
- OpenSearch for people/post search.
- EKS in **ap-southeast-1**, Terraform, OpenTelemetry collector.

ID generation happens in each API process (worker ID from config). See [ADR 0006](decisions/0006-snowflake-ids.md).

## Planned repo layout

Do not create `frontend/` or `backend/` until [feature 01](features/01-project-foundation.md) is in implementation. Docs and agent rules live at the repo root now.

```
.
├── AGENTS.md
├── README.md                 # public intro — owned separately; do not edit from this workstream
├── docs/                     # this tree
├── .cursor/rules/            # Cursor project rules (mirror AGENTS.md)
├── frontend/                 # Next.js App Router (created in 01)
└── backend/                  # Go module github.com/anhtuta/friendfold/backend
    ├── cmd/api/              # process entrypoint, router wiring
    ├── internal/
    │   ├── platform/         # config, db, redis, http, ids, log
    │   ├── auth/
    │   ├── profile/
    │   ├── friend/
    │   ├── post/
    │   ├── feed/
    │   ├── reaction/
    │   ├── comment/
    │   ├── notification/
    │   └── safety/
    ├── api/openapi.yaml
    ├── db/migrations/        # goose
    └── db/queries/           # sqlc
```

Local Docker Compose (created in 01): Postgres, Redis, Mailpit. CI: GitHub Actions.

## Roadmap

Ship in this order. Each item is a feature plan; implement its phases in order, one PR at a time.

1. [Project foundation](features/01-project-foundation.md) — API skeleton, migrations, Next.js, Compose, CI.
2. [Accounts](features/02-accounts.md) — email/password, Google, verification, reset, sessions.
3. [Profiles](features/03-profiles.md) — name, avatar, cover, bio, profile page.
4. [Friends](features/04-friends.md) — requests, unfriend, block, friends list.
5. [Posts](features/05-posts.md) — text/photo, privacy, edit/delete.
6. [News feed](features/06-news-feed.md) — chronological friends feed, infinite scroll, hybrid fan-out.
7. [Likes and comments](features/07-likes-and-comments.md) — reactions, comments, replies.
8. [Notifications](features/08-notifications.md) — in-app + realtime.
9. [Safety basics](features/09-safety-basics.md) — reports, admin review, rate limits.

After 09, the product is an MVP. Ranked feed, search, messaging, Vietnamese UI, and Kubernetes are later programs, each with new plans and ADRs.
