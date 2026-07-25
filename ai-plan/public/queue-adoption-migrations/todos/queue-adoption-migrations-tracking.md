# Queue Adoption Migrations Tracking

## Topic

Queue Adoption Migrations

## Scope

Migrate selected high-value, request-blocking business operations to the existing Task Runtime in bounded cross-boundary batches.

## Repository Truth

- `AGENTS.md`
- `openapi/**`
- `server/modules/task/**`
- `server/modules/container/**`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Task Runtime, Container module, and OpenAPI own the queue-adoption migration boundary.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - topic
closeout:
  archive: false
  lessons_review: true
```

## Current Recovery Point

- Batch 1: Docker image pull submits `container.docker-image-pull.v1` and opens Task details.
- Batch 2: container start/stop/restart submit `container.lifecycle.{action}.v1`, retain the existing action-specific permission boundary, and open shared Task details.
- Batch 3: container batch start/stop/restart submit one independent `container.lifecycle.{action}.v1` Task per accepted item; the HTTP response remains an ordered partial-result projection with explicit `accepted`, `task_id`, `status`, and failure fields. Batch remove remains synchronous.
- Guardrail: compare with `origin/main` and stop or split before 80 changed files.
- Next step: assess Container batch removal as the next bounded migration candidate.

## Task Checklist

- [x] Validate and close Batch 1 Docker image pull Task adoption.
- [x] Migrate single-container start/stop/restart to Tasks.
- [x] Migrate Container batch start/stop/restart to independent Tasks while preserving partial-result semantics.
- [ ] Assess Container batch removal as the next bounded migration.
- [ ] Assess Backup after a product execution entry exists.

## Acceptance Conditions

- Accepted operations return a Task receipt and use stable Task logs and state as execution truth.
- No legacy streaming or compatibility path remains for a migrated operation.
- Every batch stays within the configured file-change cap.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["docker-image-pull-task-adoption", "container-single-lifecycle-task-adoption"],
  "pending_batches": ["container-batch-actions-or-removal"],
  "current_batch": "container-single-lifecycle-task-adoption",
  "next_batch": "container-batch-actions-or-removal",
  "closeout_status": "validation-pending"
}
```
