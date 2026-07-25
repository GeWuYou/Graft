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
- Batch 3: container batch start/stop/restart submit one independent `container.lifecycle.{action}.v1` Task per accepted item; the HTTP response remains an ordered partial-result projection with explicit `accepted`, `task_id`, `status`, and failure fields.
- Batch 4: container batch remove submits one independent `container.lifecycle.remove.v1` Task per accepted item. The frozen plan retains `force`; the existing `container.remove` route and Task owner authorizer both enforce remove-specific authorization. Docker interruption resolves through Task Runtime `unknown` / `needs_attention` and manual reconciliation, never automatic replay.
- Guardrail: compare with `origin/main` and stop or split before 80 changed files.
- Backup execution contract: the first public entry is a Platform capability at
  `/platform/backups`; it submits a frozen, manual-reconcile Backup Task and
  excludes restore, artifact download, scheduled backups, and cleanup.
- Next step: implement the bounded Backup artifact writer and public Task-backed
  create/list/detail contract from `backup-execution-contract.md`.

## Task Checklist

- [x] Validate and close Batch 1 Docker image pull Task adoption.
- [x] Migrate single-container start/stop/restart to Tasks.
- [x] Migrate Container batch start/stop/restart to independent Tasks while preserving partial-result semantics.
- [x] Migrate Container batch remove to independent Tasks while preserving ordered partial-result and manual-reconcile semantics.
- [x] Define the first Backup product execution entry and Task contract.
- [ ] Implement the bounded Backup artifact writer and Task executor.
- [ ] Add the canonical Backup OpenAPI contract, Platform navigation, and web
  history page after the backend contract is executable.

## Acceptance Conditions

- Accepted operations return a Task receipt and use stable Task logs and state as execution truth.
- No legacy streaming or compatibility path remains for a migrated operation.
- Every batch stays within the configured file-change cap.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["docker-image-pull-task-adoption", "container-single-lifecycle-task-adoption", "container-batch-lifecycle-task-adoption", "container-batch-removal-task-adoption"],
  "pending_batches": ["backup-task-executor", "backup-public-surface"],
  "current_batch": "backup-execution-contract",
  "next_batch": "backup-task-executor",
  "closeout_status": "contract-defined"
}
```
