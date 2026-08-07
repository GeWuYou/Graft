# Platform Availability And Capability Health

## Current Status Summary

- Topic objective: implement unified platform reachability and capability health control planes.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running feature requiring design, roadmap, ADR, and loop execution.
- Canonical authority: `server/internal/contract`, `server/internal/moduleapi`, `openapi/**`, and the web shell.
- Completed so far: Phases 0-5 implementation and focused validation, including the RBAC permission migration for the
  capability snapshot API.
- Remaining: archive-readiness validation. Full `bun run check` is still blocked by the pre-existing missing Monaco
  Dockerfile language import, so cross-boundary acceptance is not yet claimed.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: server typed contracts and OpenAPI own shared semantics; web shell owns browser availability projection.

## Owned Scope

- platform availability and capability health contracts
- server and web integration required by those contracts
- topic recovery and staged validation

Out of scope:

- runtime plugin discovery or hot loading
- replacing module-owned resource and operation APIs

## Locked Decisions

1. `PlatformAvailabilityStore` and `CapabilityCoordinator` are the only state authorities.
2. Capability impact is `platform`, `feature`, or `advisory`; descriptors also have a standard category.

## Phase Plan

- Phase 0: architecture foundation and typed contracts
- Phase 1: availability baseline and router takeover
- Phase 2: Axios and TanStack Query pause/resume
- Phase 3: WebSocket/SSE lifecycle integration
- Phase 4: capability snapshot API and dashboard projection
- Phase 5: recovery UX and diagnostics

## Current Recovery Point

- Phases 0-5 are implemented. The focused backend, OpenAPI, and frontend validations recorded in the trace passed.
- The RBAC migration is part of the delivered scope and must retain both SQL-comment and default-chain version
  validation evidence.
- Risk: `bun run check` remains blocked by the existing Monaco import failure; do not archive or claim complete
  cross-boundary acceptance until that gate is resolved or explicitly adjudicated.
- Next step: reproduce and resolve or formally adjudicate the full web-check blocker, then perform archive readiness.

## Work Intake

- This topic was created through `Work Intake`.
- Full Work Contract is in the tracking file.

## Closeout Direction

- No implementation batch is pending.
- Preserve the Phase 0-5 evidence while the full web-check blocker is resolved or explicitly adjudicated.
- Perform archive readiness only after the acceptance condition for `bun run check` is satisfied.

## Validation Targets

```bash
git diff --check
graft validate backend --stage lint
python3 scripts/validate_sql_migrations.py --paths server/modules/rbac/migrations/202608070001_platform_capability_permission.sql
python3 scripts/check_migration_versions.py --mode all
cd web && bun run check
```

## Loop Entry

- Preferred entry: `ai-plan/public/platform-availability-capability-health/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
