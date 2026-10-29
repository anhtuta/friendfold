# Friendfold docs

Index of product, architecture, and build plans. Coding agents: start at [AGENTS.md](../AGENTS.md), then this folder. Implement from the feature plans; do not invent extra product surface.

Root [README.md](../README.md) is a short public intro and is owned by a separate change — do not edit it from docs work.

## Start here

| Doc | What it is |
| --- | --- |
| [00-project-overview.md](00-project-overview.md) | Vision, MVP scope, functional and non-functional requirements, architecture, repo layout, roadmap |

## Feature plans

Build in number order. Template: [_TEMPLATE.md](features/_TEMPLATE.md). One phase per PR; update **Status** when a phase ships.

| # | Feature | Doc | User-visible? |
| --- | --- | --- | --- |
| 01 | Project foundation | [features/01-project-foundation.md](features/01-project-foundation.md) | No |
| 02 | Accounts | [features/02-accounts.md](features/02-accounts.md) | Yes |
| 03 | Profiles | [features/03-profiles.md](features/03-profiles.md) | Yes |
| 04 | Friends | [features/04-friends.md](features/04-friends.md) | Yes |
| 05 | Posts | [features/05-posts.md](features/05-posts.md) | Yes |
| 06 | News feed | [features/06-news-feed.md](features/06-news-feed.md) | Yes |
| 07 | Likes and comments | [features/07-likes-and-comments.md](features/07-likes-and-comments.md) | Yes |
| 08 | Notifications | [features/08-notifications.md](features/08-notifications.md) | Yes |
| 09 | Safety basics | [features/09-safety-basics.md](features/09-safety-basics.md) | Yes (report); admin review is staff-only |

## Decisions

Architecture Decision Records. Write a new one when a choice is expensive to reverse. Next number: **0009**.

| # | Decision | Status |
| --- | --- | --- |
| [0001](decisions/0001-go-backend.md) | Go backend (chi, pgx, sqlc, goose) | Accepted |
| [0002](decisions/0002-nextjs-frontend.md) | Next.js + React + TypeScript web app | Accepted |
| [0003](decisions/0003-postgres-redis.md) | PostgreSQL + Redis | Accepted |
| [0004](decisions/0004-modular-monolith.md) | Modular monolith, one module per feature | Accepted |
| [0005](decisions/0005-monorepo-layout.md) | Monorepo: `frontend/`, `backend/`, `docs/` | Accepted |
| [0006](decisions/0006-snowflake-ids.md) | 64-bit Snowflake IDs (`BIGINT` in DB, strings on the wire) | Accepted |
| [0007](decisions/0007-tailwind-css.md) | Tailwind CSS for the Next.js app | Accepted |
| [0008](decisions/0008-dark-theme.md) | Dark theme as the default look | Accepted |

## Agent rules

| File | Role |
| --- | --- |
| [AGENTS.md](../AGENTS.md) | Canonical workflow for every agent |
| [.cursor/rules/](../.cursor/rules/) | Same policies, scoped for Cursor |

## Conventions used in these docs

- **Status** on a feature: `Not started` → `In progress (phase N)` → `Shipped`.
- **IDs** in JSON examples are decimal strings (`"1234567890123456789"`).
- **HTTP** paths are `/v1/...`. Auth is a session cookie unless noted.
- **Privacy** of a post: `public` | `friends` | `onlyMe`.
- Time is UTC, stored as `timestamptz`.
