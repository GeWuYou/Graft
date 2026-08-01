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

- Topic bootstrap and design authority are being established.
- Workers are split by server and web ownership.
- Next step: integrate bounded Phase 0 results before starting persistence/API work.

## Task Checklist

- [ ] Establish moduleapi and Build module Phase 0 server contracts.
- [ ] Establish web module registration and Build contract paths.
- [ ] Add persistence, API, Docker executor, and task stage in a later bounded batch.
- [ ] Add generated projections and end-to-end validation.

## Acceptance Conditions

- Build navigation and route authority are explicit and do not create a Docker submenu.
- No Build-specific worker, queue, log store, or realtime topic is introduced.
- Application and Runtime Target internals remain hidden behind narrow capabilities.
- Changed server/web code passes the repository completion entrypoints for its slice.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [],
  "pending_batches": ["phase-0-server-contracts", "phase-0-web-contracts", "phase-0-integration"],
  "current_batch": "phase-0-server-contracts-and-web-contracts",
  "next_batch": "phase-0-integration",
  "closeout_status": "in-progress"
}
```
