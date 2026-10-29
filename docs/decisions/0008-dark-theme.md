# ADR 0008: Dark theme as the default look

- **Status:** Accepted
- **Date:** 2026-10-29
- **Tags:** frontend, css, accessibility, web

## Context

The placeholder home and `/health` pages shipped with Tailwind's default light zinc text on a white page. Feature 01 already requires [WCAG 2.2 Level AA](../00-project-overview.md). Later screens (auth, profile, feed) need a shared look they can inherit instead of each page picking its own `zinc-*` scale.

## Decision

The Friendfold web app is **dark by default**.

- Theme tokens live in `frontend/app/globals.css` as Tailwind `@theme` CSS variables: `background`, `foreground`, `muted`, `subtle`, `accent`.
- `html` sets `color-scheme: dark`. `body` paints `background` / `foreground` so new routes inherit the look even before they choose utilities.
- Pages use those token utilities (`bg-background`, `text-foreground`, `text-muted`, `text-subtle`, `text-accent`), not ad-hoc `zinc-*` palettes.
- Contrast of text tokens on `background` meets WCAG 2.2 AA (4.5:1 for normal text). A unit test reads the CSS tokens and checks the ratios.
- No user-facing light/dark toggle in this change. A later light theme is a token swap (and a new ADR if it becomes a product setting).

Confirmed against [feature 01](../features/01-project-foundation.md) `/` and `/health`. Styling method remains Tailwind ([ADR 0007](0007-tailwind-css.md)).

## Consequences

- Future product screens start from a dark canvas and the same type/color roles.
- Agents should not introduce a second light palette beside these tokens.
- OS `prefers-color-scheme: light` does not switch the product look.

## Alternatives considered

- **Stay light (the placeholder).** Rejected; the product look is dark.
- **Follow `prefers-color-scheme`.** Extra tokens and no product control; revisit if we add a user setting.
- **Component-library theme (shadcn dark).** Out of scope; [ADR 0007](0007-tailwind-css.md) deferred kits.
