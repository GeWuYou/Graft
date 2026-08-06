# Platform Availability And Capability Health Tracking

## Topic

Platform Availability And Capability Health

## Scope

Cross-boundary implementation of platform reachability, capability health contracts, server projection, and web shell,
query, and realtime integration.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `openapi/**`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: server typed contracts and OpenAPI own shared semantics; web shell owns browser availability projection.
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - topic
    - topic design
    - topic roadmap
    - topic ADR
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Phase 0 through Phase 5 are implemented; focused validation passes and the full web check retains one unrelated Monaco dependency blocker.
- No schema or migration work is planned for the initial phases.
- Next step: archive readiness review and worktree closeout.

## Task Checklist

- [x] Phase 0: contracts, registry, coordinator, and tests
- [x] Phase 1: browser availability baseline and router takeover
- [x] Phase 2: Axios and TanStack Query gating
- [x] Phase 3: WebSocket/SSE pause and recovery
- [x] Phase 4: capability API, providers, and dashboard
- [x] Phase 5: recovering UX, diagnostics, and full acceptance

## Acceptance Conditions

- Platform and capability authorities are single-source and independently testable.
- Unavailable platform stops business traffic, retries, polling, and realtime reconnects.
- Capability statuses and diagnostics are projected through one typed API without duplicating module resource truth.
- Backend and frontend authoritative validation passes for every affected phase.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["phase-0-server-foundation", "phase-1-web-availability", "phase-2-query-gating", "phase-3-realtime", "phase-4-capability-projection", "phase-5-recovery-diagnostics"],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archive-check"
}
```
