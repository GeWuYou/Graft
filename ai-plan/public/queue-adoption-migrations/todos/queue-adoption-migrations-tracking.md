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
- Guardrail: compare with `origin/main` and stop or split before 80 changed files.
- Next step: complete cross-boundary validation and choose the next Container candidate.

## Task Checklist

- [ ] Validate and close Batch 1 Docker image pull Task adoption.
- [ ] Migrate one bounded Container lifecycle or batch action.
- [ ] Assess Backup after a product execution entry exists.

## Acceptance Conditions

- Accepted operations return a Task receipt and use stable Task logs and state as execution truth.
- No legacy streaming or compatibility path remains for a migrated operation.
- Every batch stays within the configured file-change cap.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [],
  "pending_batches": ["docker-image-pull-task-adoption", "container-lifecycle-or-batch-action"],
  "current_batch": "docker-image-pull-task-adoption",
  "next_batch": "container-lifecycle-or-batch-action",
  "closeout_status": "in-progress"
}
```
