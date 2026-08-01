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

- `phase-1-build-execution-foundation` is complete in `e6d0f5c4` after one controller-managed recovery repair.
- The accepted scope freezes the Project-authorized workspace/source/runtime context onto Build jobs, makes the Task executor consume that persisted Build-owned snapshot without a request actor, and settles Docker image evidence onto Build artifacts.
- Remaining Phase 1 scope is deliberately separate: Build-owned read API contracts and the Build Jobs web workflow. It must not create independent Task execution/log/realtime authority or expose Container/Project internals.

## Task Checklist

- [x] Establish moduleapi and Build module Phase 0 server contracts.
- [x] Establish web module registration and Build contract paths.
- [x] Add Build module descriptor, permissions/menu skeleton, and foundation migration.
- [x] Add Build module to the canonical generated registry and validate migration/dependency/permission/menu registration.
- [x] Add Build-owned frozen snapshot, Docker executor runtime identity repair, artifact settlement, and module-owned persistence.
- [ ] Add Build read API contracts and Build Jobs list/create/detail web workflow.

## Acceptance Conditions

- Build navigation and route authority are explicit and do not create a Docker submenu.
- No Build-specific worker, queue, log store, or realtime topic is introduced.
- Application and Runtime Target internals remain hidden behind narrow capabilities.
- Changed server/web code passes the repository completion entrypoints for its slice.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["phase-0-contracts", "phase-1-build-backend-foundation", "phase-1-generated-registration", "phase-1-build-execution-foundation"],
  "pending_batches": ["phase-1-build-read-api-and-web-workflow"],
  "current_batch": "phase-1-build-read-api-and-web-workflow",
  "next_batch": null,
  "closeout_status": "active",
  "stop_reason": null,
  "recovery": {
    "status": "complete",
    "resume_target": null,
    "repair_authority": "explicitly approved and consumed",
    "repair_eligible": false
  }
}
```
