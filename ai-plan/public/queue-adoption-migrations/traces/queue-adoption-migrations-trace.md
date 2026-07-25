# Queue Adoption Migrations Trace

## 2026-07-25 Docker Image Pull Task Adoption

- Created the active topic through Work Intake with the existing Task Runtime and queue-adoption design as authority.
- Migrated the canonical Docker image pull wire contract from NDJSON progress to an accepted Task receipt with an idempotency key.
- The Container image page opens shared Task details and refreshes its query only after the observed Task succeeds.

## Locked Decisions

- Do not retain a parallel NDJSON endpoint or frontend streaming fallback.
- Keep each migration wave below the 80-file local change cap.

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
