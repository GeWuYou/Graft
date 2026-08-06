# Platform Availability And Capability Health

## Current Status Summary

- Topic objective: implement unified platform reachability and capability health control planes.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running feature requiring design, roadmap, ADR, and loop execution.
- Canonical authority: `server/internal/contract`, `server/internal/moduleapi`, `openapi/**`, and the web shell.
- Completed so far: Phases 0-4 implementation and focused validation.
- Remaining: Phase 5 recovery UX and diagnostics acceptance.

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

- Phase 4 capability snapshot API and dashboard projection are implemented and validated.
- Risk: preserve generated OpenAPI freshness and keep capability observations separate from module resource truth.
- Next step: Phase 5 recovery UX and diagnostics.

## Work Intake

- This topic was created through `Work Intake`.
- Full Work Contract is in the tracking file.

## Pending Batch Direction

- Implement Phase 0 server contracts, registry, coordinator, and focused tests.
- Then integrate the browser availability gate in a separate round.

## Validation Targets

```bash
git diff --check
graft validate backend --stage lint
```

## Loop Entry

- Preferred entry: `ai-plan/public/platform-availability-capability-health/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
