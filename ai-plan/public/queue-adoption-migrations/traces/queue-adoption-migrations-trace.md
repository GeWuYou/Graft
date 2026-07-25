# Queue Adoption Migrations Trace

## 2026-07-25 Docker Image Pull Task Adoption

- Created the active topic through Work Intake with the existing Task Runtime and queue-adoption design as authority.
- Migrated the canonical Docker image pull wire contract from NDJSON progress to an accepted Task receipt with an idempotency key.
- The Container image page opens shared Task details and refreshes its query only after the observed Task succeeds.

## Locked Decisions

- Do not retain a parallel NDJSON endpoint or frontend streaming fallback.
- Keep each migration wave below the 80-file local change cap.

## 2026-07-25 Single-Container Lifecycle Task Adoption

- Migrated `start`, `stop`, and `restart` to accepted Task receipts with caller-provided idempotency keys.
- Container remains the business executor owner: lifecycle executors reuse the existing dangerous-action policy, orchestrator policy, audit publication, and Docker runtime boundary.
- Lifecycle Task owner types remain action-specific so Task detail, cancellation, and retry preserve `container.start`, `container.stop`, or `container.restart` authorization instead of widening to a read or generic dangerous-action permission.
- Batch actions and removal remain synchronous and are explicitly deferred because batch partial-result semantics and removal lifecycle behavior need their own bounded authority review.

## 2026-07-25 Container Batch Lifecycle Task Adoption

- Migrated batch `start`, `stop`, and `restart` to one independently submitted lifecycle Task per accepted container.
- Preserved the ordered partial-result contract: each item reports `accepted`, optional `task_id`/initial `status`, or explicit submission failure fields.
- Reused the existing action-specific Container permission and Task owner authorizer; batch `remove` remains synchronous and was not sent through Task Runtime.
- The web Container page observes every accepted Task, opens the shared Task detail drawer for the first item, and exposes the remaining accepted tasks as drawer entries; list refresh follows Task success rather than receipt acceptance.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["docker-image-pull-task-adoption", "container-single-lifecycle-task-adoption", "container-batch-lifecycle-task-adoption"],
  "pending_batches": ["container-batch-removal-or-backup-entry"],
  "current_batch": "container-batch-lifecycle-task-adoption",
  "next_batch": "container-batch-removal-or-backup-entry",
  "closeout_status": "validated"
}
```
