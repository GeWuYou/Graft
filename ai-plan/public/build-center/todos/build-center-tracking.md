# Docker Build Center Tracking

## Topic

Docker Build Center

## Scope

Cross-boundary Build domain authority, first Docker executor, persistence, task integration, API, and web workflow.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/architecture/docker-build-center.md`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Build owns jobs and artifacts; Task owns execution; Container owns Docker execution.
requires:
  design: true
  topic: true
  roadmap: true
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - ai-plan/public/build-center
    - ai-plan/design/architecture/docker-build-center.md
    - ai-plan/roadmap/build-center.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Topic bootstrap and Phase 0 contracts are complete.
- Phase 1 backend foundation and generated module registration are committed.
- Current step: implement the Build API and Task submission boundary; Docker executor and web workflow remain later batches.

## Task Checklist

- [x] Establish moduleapi and Build module Phase 0 server contracts.
- [x] Establish web module registration and Build contract paths.
- [x] Add Build module descriptor, permissions/menu skeleton, and foundation migration.
- [x] Add Build module to the canonical generated registry and validate migration/dependency/permission/menu registration.
- [ ] Add persistence, API, Docker executor, and task stage in later bounded batches.

## Acceptance Conditions

- Build navigation and route authority are explicit and do not create a Docker submenu.
- No Build-specific worker, queue, log store, or realtime topic is introduced.
- Application and Runtime Target internals remain hidden behind narrow capabilities.
- Changed server/web code passes the repository completion entrypoints for its slice.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["phase-0-contracts", "phase-1-build-backend-foundation", "phase-1-generated-registration"],
  "pending_batches": ["phase-1-build-api-and-task-submission", "phase-1-docker-executor-and-web-workflow"],
  "current_batch": "phase-1-build-api-and-task-submission",
  "next_batch": "phase-1-docker-executor-and-web-workflow",
  "closeout_status": "in-progress"
}
```
