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
implementation:
  includes:
    - server/modules/rbac/migrations/202608070001_platform_capability_permission.sql
validation:
  required:
    - python3 scripts/validate_sql_migrations.py --paths server/modules/rbac/migrations/202608070001_platform_capability_permission.sql
    - python3 scripts/check_migration_versions.py --mode all
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Phase 0 through Phase 5 are implemented. Focused validation passes; full `bun run check` remains blocked by the
  pre-existing missing Monaco Dockerfile language import.
- `server/modules/rbac/migrations/202608070001_platform_capability_permission.sql` is in scope and its SQL-comment
  and default-chain migration-version gates are required validation evidence.
- Next step: resolve or explicitly adjudicate the full web-check blocker before archive readiness and worktree closeout.

## Task Checklist

- [x] Phase 0: contracts, registry, coordinator, and tests
- [x] Phase 1: browser availability baseline and router takeover
- [x] Phase 2: Axios and TanStack Query gating
- [x] Phase 3: WebSocket/SSE pause and recovery
- [x] Phase 4: capability API, providers, and dashboard
- [x] Phase 5: recovering UX and diagnostics
- [ ] Complete cross-boundary acceptance: full `bun run check` passes or its blocker is explicitly adjudicated

## Acceptance Conditions

- Platform and capability authorities are single-source and independently testable.
- Unavailable platform stops business traffic, retries, polling, and realtime reconnects.
- Capability statuses and diagnostics are projected through one typed API without duplicating module resource truth.
- Backend and frontend authoritative validation passes for every affected phase.
- The RBAC capability-permission migration passes SQL-comment and default-chain version validation.
- Complete cross-boundary acceptance is gated on a passing or explicitly adjudicated `bun run check`; focused checks do
  not substitute for that gate.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["phase-0-server-foundation", "phase-1-web-availability", "phase-2-query-gating", "phase-3-realtime", "phase-4-capability-projection", "phase-5-recovery-diagnostics"],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archive-check-blocked-by-web-check"
}
```
