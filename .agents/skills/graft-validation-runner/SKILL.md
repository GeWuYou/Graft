---
name: graft-validation-runner
description: Choose and run the smallest correct validation for Graft work. Use when the task touches server, web, automation, or cross-boundary contracts and Codex needs to decide which current validation commands are justified, which ones are not yet possible, and how to report validation gaps honestly.
---

# Graft Validation Runner

Use this skill to choose the correct validation scope for `Graft` without inventing a second validation truth.

## Preconditions

1. Ensure the current turn already has the startup receipt required by the root `AGENTS.md`.
2. Read `.ai/environment/tools.ai.yaml` before choosing runtimes or package managers.
3. Treat root `AGENTS.md` plus repository entrypoints as validation truth:
   - `cd server && go run ./cmd/graft validate backend`
   - `cd web && bun run check`

README files, tracking docs, skills, and CI workflows may point to these entrypoints or to narrower execution slices,
but they must not redefine acceptance criteria, command order, or local-vs-CI environment rules.

## Validation Workflow

1. Classify the touched area:
   - `server`
   - `web`
   - `cross-boundary`
   - `docs or automation`
2. For `server` work:
   - for completion-state work, prefer `cd server && go run ./cmd/graft validate backend`
   - for intermediate iteration, prefer the smallest `go test` or `go build` scope that covers the touched code
   - widen to `go test ./...` or `go build ./...` when shared abstractions, lifecycle code, plugin wiring, or dependency resolution changed
   - if you use `graft validate backend --stage ...`, report it as an execution-layer slice of the unified backend entrypoint, not as a second validation contract
3. For `web` work:
   - for completion-state work, prefer `cd web && bun run check`
   - for intermediate iteration, prefer the smallest direct command that proves the changed TypeScript, route, page, style, or test surface
   - use .ai/environment/tools.ai.yaml to confirm the repository current Bun toolchain instead of inventing a second local rule
4. For `cross-boundary` work:
   - validate both sides when contracts, menus, routes, permissions, lifecycle semantics, or shared validation entrypoints changed
5. For docs or automation work:
   - validate the referenced entrypoints, environment rules, and workflow/doc wording instead of pretending the slice has runtime coverage
   - prefer structural checks such as `rg`, `sed`, `git diff`, CLI help output, or YAML parsing
   - if a workflow keeps split jobs or stage flags, verify the wording makes them execution-layer decomposition rather than a second truth
   - if no stronger real validation exists, report the exact limitation instead of pretending the area was fully validated
6. When the touched slice includes live server schema or migration files:
   - run `python3 scripts/validate_sql_migrations.py` to check live migration SQL comments and versions
   - apply the same check to core-owned handwritten migration directories such as `server/internal/httpx/migrations/**`,
     not only plugin Ent paths
   - keep `server/internal/ent/migrate/migrations/**` excluded unless the task explicitly targets legacy/manual replay migrations

## Reporting Rules

* state the exact command you ran
* state whether each command was a full repository entrypoint, a focused direct validation, or an execution-stage slice
* if you could not run the expected validation, say why
* keep validation claims proportional to the repository's current maturity
