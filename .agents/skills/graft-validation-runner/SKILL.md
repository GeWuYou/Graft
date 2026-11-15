---
name: graft-validation-runner
description: "Select the smallest correct Graft validation scope and report completion evidence or concrete gaps."
---

# Graft Validation Runner

Use this skill to choose the correct phase and validation scope for `Graft` without inventing a second validation truth.

## Preconditions

1. Ensure the current turn already has the startup receipt required by the root [AGENTS.md](../../../AGENTS.md).
2. Use the already-read `.ai/environment/tools.ai.yaml` before choosing runtimes or package managers; refresh only when capabilities may have changed.
3. Treat root [AGENTS.md](../../../AGENTS.md) plus repository entrypoints as validation truth:
   - `cd server && go run ./cmd/graft validate backend`
   - `cd web && bun run check`

README files, tracking docs, skills, and CI workflows may point to these entrypoints or to narrower execution slices,
but they must not redefine acceptance criteria, command order, or local-vs-CI environment rules.

Reuse passing results for unchanged code and unchanged relevant dependencies. Skill transitions alone do not justify
rerunning checks; hooks, CI, and phase-specific preparation and full CI entrypoints remain effective.

## Validation Workflow

1. Record the verification classification before selecting commands:
   - `agent-only`, `human-acceptance-required`, `browser-automation-required`, or `mixed`
   - record `browser_status` as `not-needed`, `authorized-local`, or `ci-contract`
   - do not select local browser interaction unless the task has explicit user/developer authorization and the
     [web/AGENTS.md](../../../web/AGENTS.md) conditions are met
2. Classify the touched area:
   - `server`
   - `web`
   - `cross-boundary`
   - `docs or automation`
3. For `server` work:
   - for commit preparation, use `just check-local-plan` / `just check-local` with explicit consumer tests
   - for full CI or explicit full validation, use `cd server && go run ./cmd/graft validate backend`
   - for intermediate iteration, prefer the smallest `go test` or `go build` scope that covers the touched code
   - widen to `go test ./...` or `go build ./...` when shared abstractions, lifecycle code, plugin wiring, or dependency resolution changed
   - if you use `graft validate backend --stage ...`, report it as an execution-layer slice of the unified backend entrypoint, not as a second validation contract
4. For `web` work:
   - for commit preparation, select changed files and consumer tests through the same local planner
   - for full CI or explicit full validation, use `cd web && bun run check`
   - for intermediate iteration, prefer the smallest direct command that proves the changed TypeScript, route, page, style, or test surface
   - use .ai/environment/tools.ai.yaml to confirm the repository current Bun toolchain instead of inventing a second local rule
5. For `cross-boundary` work:
   - select consumers on both sides when contracts, menus, routes, permissions, lifecycle semantics, or shared validation entrypoints changed; full CI covers both domains
6. For docs or automation work:
   - validate the referenced entrypoints, environment rules, and workflow/doc wording instead of pretending the slice has runtime coverage
   - prefer structural checks such as `rg`, `sed`, `git diff`, CLI help output, or YAML parsing
   - if a workflow keeps split jobs or stage flags, verify the wording makes them execution-layer decomposition rather than a second truth
   - if no stronger real validation exists, report the exact limitation instead of pretending the area was fully validated
7. When the touched slice includes live server schema or migration files:
   - run `python3 scripts/validate_sql_migrations.py --changed --base-ref <merge-base>` to check live migration SQL comments and versions
   - apply the same check to core-owned handwritten migration directories such as `server/internal/httpx/migrations/**`,
     not only plugin Ent paths
   - keep the removed historical Ent/manual replay chain non-authoritative; do not recreate it as validation fallback

## Reporting Rules

* state the exact command you ran
* state whether each command was a full repository entrypoint, a focused direct validation, or an execution-stage slice
* if you could not run the expected validation, say why
* keep validation claims proportional to the repository's current maturity
* when human judgment remains, report `human_acceptance: awaiting_human_acceptance` and provide the concise acceptance
  contract defined by `AI代码生成与Review规范.md`; do not substitute browser artifacts for that contract

Phase timing and worktree/index evidence are defined in [development validation](../../../docs/development/validation.md).
