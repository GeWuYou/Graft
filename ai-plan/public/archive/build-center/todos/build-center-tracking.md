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
- `phase-1-build-read-api-and-web-workflow` is accepted after the authorized migration repair assigned Build versions `202608040003` and `202608040004`, refreshed Atlas and embedded-registry artifacts, and passed the full cross-boundary validation chain.

## Task Checklist

- [x] Establish moduleapi and Build module Phase 0 server contracts.
- [x] Establish web module registration and Build contract paths.
- [x] Add Build module descriptor, permissions/menu skeleton, and foundation migration.
- [x] Add Build module to the canonical generated registry and validate migration/dependency/permission/menu registration.
- [x] Add Build-owned frozen snapshot, Docker executor runtime identity repair, artifact settlement, and module-owned persistence.
- [x] Add Build read API contracts and Build Jobs list/create/detail web workflow.

## Acceptance Conditions

- Build navigation and route authority are explicit and do not create a Docker submenu.
- No Build-specific worker, queue, log store, or realtime topic is introduced.
- Application and Runtime Target internals remain hidden behind narrow capabilities.
- Changed server/web code passes the repository completion entrypoints for its slice.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["phase-0-contracts", "phase-1-build-backend-foundation", "phase-1-generated-registration", "phase-1-build-execution-foundation", "phase-1-build-read-api-and-web-workflow"],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "phase-1-accepted",
  "stop_reason": null,
  "recovery": {
    "status": "none",
    "resume_target": null,
    "repair_authority": "Build live migration chain",
    "repair_eligible": false
  }
}
```
