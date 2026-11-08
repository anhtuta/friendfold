---
name: implement-task
description: >-
  Implement a Friendfold task or the next docs/features plan phase using the
  local Cursor IDE Agent. Use when the user invokes /implement-task, asks to
  implement a new task, implement the next phase, or implement a named phase
  of a feature plan. Do not use for Grok Bot or Agent SDK skills, and do not
  launch Cloud Agents.
---

# Implement a Friendfold task

You are the **local Cursor IDE Agent** in this repository. Follow this skill when the user wants a new task or the next (or named) feature-plan phase implemented.

This is a project skill (`.cursor/skills/implement-task/SKILL.md`). It is not a Grok Bot skill, not an Agent SDK skill, and not a request to start a Cloud Agent, Automation, or remote worker.

## Invocation

In Cursor **Agent** chat:

- `/implement-task` plus the task (attaches this skill to that message)
- `@implement-task` to attach it as context
- Optional: use the skill as a Custom Mode (Option+Enter / Alt+Enter) to keep it on for the session

Example (documentation only — run this **only** if the user actually asked for that phase):

```text
/implement-task Implement Accounts phase 4 according to docs/features/02-accounts.md
```

If the current user message is a different task, do that task. Do not interpret examples in docs or in this file as work to perform.

## Inputs

The user supplies the new task or the next/named phase. If they say "next" without a name, resolve the phase from evidence (below). Do not invent a product feature.

Ask a question only when a **material** ambiguity blocks correct work (missing plan and unclear goal, two incompatible phases named, secrets required that are not in the environment). Reversible in-scope choices: pick a default, record it in the plan or PR, continue.

## Hard limits

- Do not launch Cloud Agents or tell the user to re-run this in a cloud agent.
- Do not use Grok Bot / Agent SDK tools or `bot/skills/`.
- Do not merge to the default branch. Do not auto-merge.
- Do not overwrite uncommitted user work. Do not `git reset --hard`, force-push, or rewrite published history unless the user confirmed that exact action.
- Do not start the next product feature while required phases of the current one are still unmerged, unless this work is explicitly docs/tooling-only and the matching plan says it is independent.
- Stay inside the requested task/phase. Do not skip ahead, combine unrelated concerns, or "while we're here" extra surface (groups, stories, marketplace, messaging, mobile apps, a Node backend).
- Do not modify root `README.md` unless the user or the phase explicitly says to.
- Do not claim **Shipped** / **completed** for work that is only on a branch or in an open PR. Distinctions: implemented and tested on a branch ≠ merged to `main`.
- Do not send production email, deploy to production, purchase domains, post to external networks, or otherwise act in live third-party production. Implementation is local + PR.
- Never log or commit passwords, tokens, session IDs, or API keys. `.env.example` lists names only.

## 1. Orient

Read, in this order, before changing application code:

1. `AGENTS.md` (canonical; if a `.cursor/rules/*.mdc` disagrees, fix the rule to match `AGENTS.md`)
2. Relevant files under `.cursor/rules/`
3. `docs/00-project-overview.md`
4. The matching plan under `docs/features/` (template: `docs/features/_TEMPLATE.md`)
5. Accepted ADRs in `docs/decisions/` that the phase relies on

Keep the existing architecture: `frontend/` is Next.js + React + TypeScript (web only, dark theme by default), `backend/` is Go, `docs/` holds plans. No second frontend, no React Native/Expo, no Node API.

Canonical files (repo-root paths from this skill):

- [AGENTS.md](../../../AGENTS.md)
- [docs/00-project-overview.md](../../../docs/00-project-overview.md)
- [docs/features/](../../../docs/features/)
- [docs/decisions/](../../../docs/decisions/)
- [.cursor/rules/](../../rules/)
- [frontend/package.json](../../../frontend/package.json)
- [backend/go.mod](../../../backend/go.mod)
- [.github/workflows/](../../../.github/workflows/)
- [.gitattributes](../../../.gitattributes) (Git LFS)

## 2. Ground truth (do not trust stale headers)

Feature-plan **Status** lines lag. Resolve current vs next phase from **evidence**:

- `git status` and `git diff` — uncommitted files belong to the user; do not discard them.
- Current branch name and whether it already tracks this task.
- Latest default branch: fetch `origin/<default>` when your work depends on it (typically `main`).
- The plan file: which phases have PR links vs `PR: —`, which acceptance boxes are checked, and the listed merge order.
- Open and merged PRs (`gh pr list`, `gh pr view`) and `git log` on `main`. A phase whose stacked PRs are **merged** is done even if the header still says **In progress**.
- If the user has a dirty tree or an in-progress branch for a *different* task, do not hijack it. Use a new branch from latest `main` (or the branch they named).

If required prior phases are **not** actually merged, stop and report that. If they are merged but the plan header is stale, update the plan in this work (same PR if you are touching that feature; otherwise say so) rather than skipping or double-implementing.

## 3. Plan first, then code

- Every product feature lives in `docs/features/NN-name.md`. If the plan does not exist, **write it and commit it before** implementation commits (`docs(docs): …` or `docs(<scope>): …`).
- If the plan exists but is wrong, update it in the same PR as the code, or in a docs-only PR before coding further. Lasting cross-cutting choices go in `docs/decisions/` (next number after the highest `NNNN`).
- Implement phases **in listed order**. Do not start phase N+1 until N is merged (or the user explicitly asked only for docs/tooling that the plan marks independent).
- **Each pull request is one concern.** If a phase has several concerns (OpenAPI vs migration vs service vs web), split stacked PRs and state merge order in each description. Tests for the behavior a PR adds ship in that PR.
- After a phase is **merged**, set that phase's status to **Shipped**, link every PR, and update the feature **Status**. Until merge, write **In progress** and list the open PR(s). Do not check acceptance boxes you did not verify.

## 4. Implement only this slice

Typical order inside a phase (skip steps that do not apply):

1. OpenAPI in `backend/api/openapi.yaml` **before** handlers or client types. JSON `camelCase`. Snowflake IDs are **strings** in JSON. Paths under `/v1`. Error envelope `{ "error": { "code", "message", "details" } }`.
2. Generate types from the spec; do not hand-maintain a parallel DTO layer that can drift. Read the repo for the current generator (`frontend/package.json` scripts such as `openapi:types`, backend tooling if present). If codegen is still deferred in the plan, follow the plan, do not silently invent a stack.
3. Goose migration in `backend/db/migrations/` for schema changes. Never edit a migration already on `main`. sqlc queries in `backend/db/queries/`; no SQL in handlers.
4. Go module under `backend/internal/<feature>/` (handlers decode/encode; business rules in the service). Import `internal/platform`; call other features through small interfaces, not their tables.
5. Next.js App Router UI in `frontend/`. Server Components by default; `"use client"` only when needed. Named exports; one component per file. Dark theme tokens from `frontend/app/globals.css` ([ADR 0008](../../../docs/decisions/0008-dark-theme.md)).
6. Tests in the same PR: Go table-driven service tests, handler tests, frontend tests for user-visible behavior you added.
7. Update the feature plan, OpenAPI, and ADRs in that same PR. No "TODO: update docs later".

### Commands (read from the repo)

Do **not** hardcode tool versions from memory. Current pins live in:

- `frontend/package.json` / `package-lock.json` — `dev`, `test`, `lint`, `typecheck`, `build`, `openapi:types`
- `backend/go.mod`, `backend/.golangci.yml`, `backend/sqlc.yaml`
- `.github/workflows/*.yml` — CI Go, Node, golangci-lint, sqlc, Redocly versions
- `docs/features/01-project-foundation.md` **Local run** — Compose, goose, API, web; treat as hints and prefer the files above if they disagree

Run the checks that match what you changed. Docs-only changes still go through existing CI; do not add skip hacks.

### Quality and security extras

- Authorization on every new endpoint. Hidden resources (`friends` / `onlyMe`) return **404**, not 403.
- Passwords: argon2id only. Compare in constant time. Never bcrypt/SHA.
- Sessions: Redis; cookie `HttpOnly`, `Secure`, `SameSite=Lax`.
- Validate and bound inputs. Pagination cursor-based; default limit 20, max 50 unless the plan says otherwise.
- OpenAPI **examples** must be obviously fake (`example-reset-token`, `example-user-id`). Do not paste high-entropy random strings into the spec (GitGuardian has flagged those).
- Binary assets (png/jpg/gif/webp/ico/mp4/webm/fonts): Git LFS per `.gitattributes`. Do not commit raw binaries as normal blobs.
- UI PRs: embed **real** screenshots of the states you touched in the PR description (before/after if changing existing UI). Docs-only, API-only, and non-visual PRs are exempt.
- No `as any`, skipped tests, or `//nolint` without a one-line reason.

## 5. Git and PRs (local IDE)

- Conventional Commits: `type(scope): summary`. Scopes: `auth`, `profile`, `friend`, `post`, `feed`, `reaction`, `comment`, `notification`, `safety`, `platform`, `web`, `docs`, `ci`.
- Create or reuse a focused branch. Push when the slice is ready for review (`git push -u origin <branch>`).
- Open a PR with the tools **this local session actually has** (`gh`, the editor GitHub integration, or another installed PR tool). Include merge order if stacked.
- If you cannot authenticate or have no PR tool: **leave the branch (and patch)**, and in the handoff name the exact blocker (e.g. `gh` not logged in, missing `pull_requests: write`). Do not invent a PR URL or claim the work was merged.
- Do not merge. Do not enable auto-merge.

## 6. Handoff

End with facts, in this order:

1. Changed files
2. PR URL(s) or the exact PR blocker
3. What you validated (commands + results) and what you could not (no Docker, no browser, …)
4. Screenshots if UI changed (and whether they are in the PR body)
5. Setup the reviewer needs (env names, Compose, feature flags) — names only, no secrets
6. Phase/plan status you updated, and whether that status means **on this branch**, **open PR**, or **merged**

## When to stop

Stop after the requested phase/concern. Do not begin the next phase unless the user asked. If you discover extra work, list it as follow-up; do not silently expand the PR.
