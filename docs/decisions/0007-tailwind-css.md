# ADR 0007: Tailwind CSS

- **Status:** Accepted
- **Date:** 2026-10-29
- **Tags:** frontend, css, web

## Context

Feature 01 needs one styling method for the Next.js app so later pages (auth, profile, feed) do not mix approaches. The plan left **Tailwind vs CSS modules** open, with Tailwind as the default if we must proceed.

We also need to stay inside the foundation scope: pick a consistent method, not a component library.

## Decision

Style the web app with **Tailwind CSS** (the version `create-next-app` ships for this Next.js major).

- Utility classes in TSX. Shared tokens (colors, type scale) live in the Tailwind theme / CSS entry, not in a second design system.
- Global CSS is only the Tailwind import plus a small reset/base. No CSS-in-JS (styled-components, Emotion, Stitches).
- CSS modules are allowed only for a one-off that Tailwind cannot express; that exception needs a comment. Do not start a parallel module stylesheet tree.
- No UI kit (shadcn, MUI, Chakra, Radix themes) in foundation. A kit would be a new ADR.

Confirmed in [feature 01 phase 3](../features/01-project-foundation.md).

## Consequences

- Agents and humans share one class vocabulary; the placeholder pages and later product UI look like one product.
- Bundle size depends on Tailwind's compiler (unused utilities are dropped). Keep class strings static enough for the compiler to see them.
- Visual change later is mostly theme tokens, not a rewrite of component CSS files.
- Vietnamese UI (later) does not require a different CSS approach.

## Alternatives considered

- **CSS modules.** Fine isolation, more files, no shared scale unless we invent one. Rejected so we do not grow an ad-hoc token system beside the components.
- **Plain global CSS.** Too easy to collide; no constraint on naming.
- **CSS-in-JS.** Extra runtime and a second mental model; not needed for Server Components-first pages.
- **Component library now.** Out of scope for feature 01; would lock visual language before the product UI exists.
