---
name: graft-validation-runner
description: Choose and run the smallest correct validation for Graft work. Use when the task touches server, web, automation, or cross-boundary contracts and Codex needs to decide which current validation commands are justified, which ones are not yet possible, and how to report validation gaps honestly.
---

# Graft Validation Runner

Use this skill to choose the correct validation scope for `Graft`.

## Validation Workflow

1. Classify the touched area:
   - `server`
   - `web`
   - `cross-boundary`
   - `docs or automation`
2. For `server` work:
   - if `server/go.mod` exists, prefer the smallest `go test` or `go build` scope that covers the touched code
   - widen to `go test ./...` or `go build ./...` when shared abstractions, lifecycle code, or plugin wiring changes
3. For `web` work:
   - if only package presence exists, keep validation at the current smoke-install level
   - once real scripts exist, prefer install + typecheck + build
4. For `cross-boundary` work:
   - validate both sides when contracts, menus, routes, or permissions changed
5. For docs or automation work:
   - run the smallest structural check available
   - if no real runtime validation exists, report the exact limitation instead of pretending the area was fully validated

## Reporting Rules

* state the exact command you ran
* if you could not run the expected validation, say why
* keep validation claims proportional to the repository's current maturity
