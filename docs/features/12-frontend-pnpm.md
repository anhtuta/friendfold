# Feature 12: Frontend pnpm

**Status:** In progress (phase 1)  
**Owner:** —  
**Plan version:** 1

> Developer tooling, not an MVP product feature. Independent of 02–09. Does not add user-facing surface.

## Goal

The Next.js app in `frontend/` is installed and run with **pnpm** (Corepack-pinned), not npm or yarn. CI, Make, and local docs match.

## Scope

### In scope

- Replace `frontend/package-lock.json` with `frontend/pnpm-lock.yaml` (import so resolved versions stay as close as possible).
- `packageManager`: `pnpm@12.18.0` (npm `latest` as of 2026-11-16). Corepack.
- Block npm/yarn: `preinstall` script (local, no `only-allow` download), `engines.pnpm` plus npm/yarn sentinels (`please-use-pnpm`), `engine-strict` in `frontend/.npmrc`.
- Frontend CI: `pnpm/action-setup` + `setup-node` cache `pnpm`, `pnpm install --frozen-lockfile`.
- Makefile frontend targets, `docs/deployment/deployment-local.md`, AGENTS.md, implement-task skill, feature 01 commands that still say npm.

### Out of scope

- Multi-package pnpm workspaces / extra JS packages ([ADR 0005](../decisions/0005-monorepo-layout.md)). `frontend/pnpm-workspace.yaml` is pnpm 12 config for the one frontend package only.
- Moving root OpenAPI lint (`npx @redocly/cli`) onto pnpm.
- Root `README.md` (owned separately; it has no npm commands).
- Dockerfiles (none for the frontend today).

## User stories

- As a **developer**, I want one package manager for `frontend/`, so lockfiles and CI do not drift.
- As a **developer**, I want `npm install` in `frontend/` to fail clearly, so I do not commit a second lockfile.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-12-01 | `frontend/package.json` has `packageManager` `pnpm@12.18.0` and `engines.pnpm`. |
| F-12-02 | `frontend/pnpm-lock.yaml` is committed; `package-lock.json` is gone. |
| F-12-03 | `npm install` in `frontend/` exits non-zero with a message that pnpm is required. |
| F-12-04 | Frontend CI uses pnpm with `--frozen-lockfile` and caches from `pnpm-lock.yaml`. |
| F-12-05 | Makefile and local docs use `pnpm`, not `npm install` / `npm ci`. |

## Data model / API / UI

N/A.

## Dependencies

- Requires: [01](01-project-foundation.md) (frontend + CI). Makefile/docs live in [11](11-local-dev-tooling.md); this PR stacks on that branch until 11 merges.
- Enables: consistent local/CI installs for later web work.

## Implementation phases

### Phase 1 — Switch `frontend/` to pnpm

**Status:** In progress  
**PR:** https://github.com/anhtuta/friendfold/pull/40

Work: lockfile import, Corepack pin, npm/yarn guard, CI, Makefile, docs.

Acceptance:

- [x] `pnpm install --frozen-lockfile` in `frontend/`
- [x] `pnpm typecheck`, `pnpm test`, `pnpm lint`, `pnpm build`
- [x] `npm install` in `frontend/` fails with a clear pnpm message
- [x] Make frontend targets dry-run as `pnpm …`

## Testing plan

- Frontend scripts above; `make -n frontend` / `test-frontend` / `lint-frontend` / `typecheck`.
- No UI screenshots.

## Open questions

- **`npx only-allow pnpm` vs a local script:** Default **local `node` script** (`frontend/scripts/only-pnpm.cjs`) so `preinstall` does not hit the network. Same check: `npm_config_user_agent` must start with `pnpm/`.
- **Lifecycle scripts (pnpm 12):** Settings live in `frontend/pnpm-workspace.yaml` (pnpm 12 ignores `package.json` `"pnpm"`). Allow `unrs-resolver` via `allowBuilds`. That file is **config for one package**, not a multi-package workspace ([ADR 0005](../decisions/0005-monorepo-layout.md)).
