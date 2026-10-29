# ADR 0002: Next.js frontend

- **Status:** Accepted
- **Date:** 2026-10-29
- **Tags:** frontend, web, typescript

## Context

Users need a web app: auth pages, profile, friends, composer, feed, notifications, and a small admin review UI. There is **no native mobile app** in this phase of the product. SEO for public profiles is nice-to-have; logged-in feed performance and a sane TypeScript model matter more.

Candidates: Next.js (App Router), Remix, a Vite SPA, and (rejected for now) React Native.

## Decision

Build one **Next.js + React + TypeScript** web application in `frontend/`, App Router, React Server Components by default.

- Talk to the Go API over HTTPS using types generated from `backend/api/openapi.yaml`.
- Session cookie is set by the Go API; the Next.js app is a confidential browser client, not a BFF that re-implements business rules. A thin Next.js route handler is allowed only as a same-origin proxy if we need to hide the API host; it must not contain domain logic.
- No React Native, Expo, or separate admin SPA. Staff tools are routes in this app, gated by role.

## Consequences

- One frontend toolchain, first-class TypeScript, good SSR for public profiles later.
- We must keep `"use client"` surfaces small so the feed's first paint can still be server-rendered.
- i18n (English now, Vietnamese later) should be introduced as a library (`next-intl` or equivalent) rather than ad-hoc strings — tracked in [feature 01](../features/01-project-foundation.md).
- A mobile app, if it happens, will consume the same OpenAPI; this ADR does not authorize building it.

## Alternatives considered

- **Vite SPA.** Simpler hosting, worse first paint and no server-side session-aware render for the feed.
- **Remix.** Viable; Next.js is the team's default and has broader agent/codegen familiarity.
- **Server-only HTML (templ / Go templates).** Rejected; we want a React component model for the feed and realtime UI.
