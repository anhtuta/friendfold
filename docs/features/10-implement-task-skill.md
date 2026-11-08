# Feature 10: Cursor IDE `implement-task` skill

**Status:** In progress (phase 1)  
**Owner:** —  
**Plan version:** 1

> Agent tooling, not an MVP product feature. Independent of features 02–09: this plan does not skip the product roadmap, does not block accounts/profiles/…, and does not add user-facing surface. Number `10` is docs/tooling order only.

## Goal

A person working in **Cursor IDE** can invoke a repository-local Agent Skill so the **local** IDE agent implements a new Friendfold task (or the next specified plan phase) using this repo's existing plan-first workflow. They should not need to paste `AGENTS.md` into chat every time.

## Scope

### In scope

- One project-level Cursor Agent Skill at `.cursor/skills/implement-task/SKILL.md` (folder name = skill `name`).
- Explicit invocation `/implement-task` in Agent chat, plus a `description` so Agent can also attach the skill when the user asks to implement a task or phase.
- Instructions for a **local Cursor IDE agent** (read plans/rules, ground truth from git/PRs, one concern per PR, stay in the requested phase).
- Minimal discoverability in existing docs (`AGENTS.md` and this docs index). One usage example that is documentation only.

### Out of scope

- Application code, schema, OpenAPI, Docker, CI workflow changes, frontend/backend behavior.
- Implementing any product phase (including Accounts phase 4). The usage example must not be executed as part of this work.
- Grok Bot / Agent SDK skills (`bot/skills/`, `~/.cursor/skills-cursor/`, `@grok/agent-sdk`).
- Instructions to launch Cloud Agents, Automations, or self-hosted workers.
- A new always-on Cursor rule that duplicates the skill.
- Root `README.md` (owned separately).
- An ADR (skill path is Cursor's documented project layout, not an architecture choice).

## User stories

- As a **developer in Cursor IDE**, I want to type `/implement-task` and name a task or phase, so the local agent follows Friendfold's plan-first workflow without me restating it.
- As a **developer**, I want the skill in the repo, so every clone gets it without installing a personal/global skill.

## Functional requirements

| ID | Requirement |
| --- | --- |
| F-10-01 | Skill directory is `.cursor/skills/implement-task/` with `SKILL.md`. |
| F-10-02 | YAML frontmatter has required `name: implement-task` (matches folder) and a `description` that states what the skill does and when to use it. |
| F-10-03 | Humans invoke it with `/implement-task` in Cursor Agent chat (also `@implement-task` to attach). |
| F-10-04 | The skill targets the local IDE agent: no Cloud Agent launch, no Grok Bot tools. |
| F-10-05 | The skill tells the agent to read `AGENTS.md`, `.cursor/rules/`, the relevant feature plan, and accepted ADRs before coding. |
| F-10-06 | The skill tells the agent to resolve the current/next phase from git + merged/open PRs, not from stale **Status** headers alone. |
| F-10-07 | Existing docs link to the skill once; they do not copy the full procedure. |

## Data model

N/A — no tables or Redis keys.

## API endpoints

N/A — no HTTP API.

## Frontend pages / components

N/A — no UI.

## Non-functional and scaling considerations

N/A for product traffic. Keep `SKILL.md` focused (Cursor/Agent Skills guidance: instructions load when the skill is used; prefer pointing at `AGENTS.md` over duplicating it).

## Security and privacy

- Skill text must forbid committing or logging secrets, and require obviously fake OpenAPI examples (GitGuardian has flagged high-entropy spec placeholders before).
- No credentials in the skill files.

## Dependencies on other features

- Requires: nothing from 02–09. Feature 01 docs/rules already exist; the skill reads them.
- Enables: faster, consistent local implementation of later product phases. Does not replace those plans.

## Implementation phases

Each phase is one PR. Do not start phase N+1 until N is shipped.

### Phase 1 — Add the project skill and a short pointer in docs

**Status:** In progress  
**PR:** https://github.com/anhtuta/friendfold/pull/38 (open; not merged)

Work:

- Add `.cursor/skills/implement-task/SKILL.md` per current Cursor Agent Skills docs (project path `.cursor/skills/<name>/SKILL.md`, required frontmatter `name` + `description`).
- Point to it from `AGENTS.md` and `docs/README.md` with a usage example that is **not** an instruction to run Accounts phase 4 in this PR.
- Update this plan's phase status and PR link when the skill lands.

Acceptance:

- [x] Skill path and `name` are `implement-task`; frontmatter parses; `name` matches the folder. (Verified on this branch with `skills_ref.validate`; not merged.)
- [x] Invocation documented as `/implement-task` (and `@implement-task`).
- [x] Skill body covers local-IDE workflow: orient, ground truth, plan-first commit, one concern per PR, no auto-merge, no overwriting uncommitted user work, report a PR blocker instead of inventing success.
- [x] `AGENTS.md` and `docs/README.md` each have a short pointer (no duplicated playbook).
- [x] No `frontend/` / `backend/` / CI / OpenAPI / migration changes.
- [x] Markdown links from the new/changed docs resolve.

Decisions / deviations (phase 1):

- Verified against Cursor docs (2026-11-08): [Agent Skills](https://cursor.com/docs/skills) and [Skills help](https://cursor.com/help/customization/skills). Project skills live in `.cursor/skills/<name>/SKILL.md`; required frontmatter is `name` (must match the folder) and `description`; humans invoke with `/skill-name` or `@skill-name`. Built-in skills do not include `implement-task`.
- Omitted `disable-model-invocation` so Agent can also attach the skill when the user asks to implement a phase without the slash.
- Did **not** copy the skill into `.agents/skills/` or `~/.cursor/skills/`.
- Usage example in docs (`/implement-task Implement Accounts phase 4 …`) is documentation only; this PR does not implement Accounts phase 4.

## Testing plan

- Docs-only: parse YAML frontmatter; `name` equals parent folder; `description` non-empty and ≤1024 characters.
- Confirm relative/repo-root links in the skill, this plan, `AGENTS.md`, and `docs/README.md` point at files that exist.
- Confirm no application paths are modified.
- Existing GitHub Actions still run on the PR (path filters were never added); they should stay green without new jobs.
- No UI screenshots (no user-visible UI).

## Open questions

- **`.cursor/skills/` vs `.agents/skills/`:** Default **`.cursor/skills/`** — Cursor documents both as project-level; this repo already uses `.cursor/rules/`, so keep skills next to rules. Do not duplicate into `.agents/skills/` or `~/.cursor/skills/`.
- **Auto-invoke vs slash-only:** Default **allow model invocation** (omit `disable-model-invocation`). Users can still type `/implement-task`. Slash-only would hide the skill when someone says "implement the next phase" without the slash.
