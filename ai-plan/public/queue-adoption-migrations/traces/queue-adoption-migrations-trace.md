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

## 2026-07-25 Container Batch Remove Task Adoption

- Migrated batch `remove` to one independently submitted `container.lifecycle.remove.v1` Task per accepted container; the canonical OpenAPI operation now returns only `202 Accepted` ordered partial submission results.
- The frozen Task input contains both the validated container reference and `force`. The existing `container.remove` authorization remains the HTTP guard and is reused by the action-specific Task owner authorizer for detail, cancellation, and retry.
- Remove keeps `max_attempts=1` and `manual_reconcile`: Docker interruption can leave the external result unknown, so the Task drawer is the authority for failure, `unknown`, and `needs_attention` rather than optimistic list removal.
- The web list opens the first accepted remove Task in the shared drawer, retains selections at receipt time, and refreshes only after observed Task success.

## 2026-07-25 Backup Execution Contract

- Classified Backup history as a `platform-backup` Platform capability with the stable visible route `/platform/backups`; it is neither a runtime nor an Update child entry.
- The initial public operation is manual Backup creation only. It freezes a two-stage, one-attempt Task plan, preserves `manual_reconcile` for interruption ambiguity, and exposes only safe Backup summaries.
- Restore, artifact download/browsing, cleanup, scheduled execution, path/DSN/command inputs, and automatic replay remain excluded. Existing Update runner handoffs remain a narrow Update integration and are not reused as the public Backup execution API.

## 2026-07-26 Backup Task Executor Authority Repair

- Added a forward-only Backup-owned `task_id` association, backfilled only completed runner handoffs, and refreshed the embedded live migration registry.
- The artifact creation Stage remains the sole writer. The record Stage now only verifies the frozen artifacts before persisting metadata, so verification cannot repair snapshots, create directories, or invoke `pg_dump`.
- A unique Task association makes manual record-stage retries return the existing Backup after validating the frozen metadata instead of creating a second record.
- Validated the Backup package, migration comment/version gates, Atlas checksum, generated registry freshness, and the full backend entrypoint. The next batch remains the bounded public Backup surface.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["docker-image-pull-task-adoption", "container-single-lifecycle-task-adoption", "container-batch-lifecycle-task-adoption", "container-batch-removal-task-adoption", "backup-task-executor-authority"],
  "pending_batches": ["backup-public-surface"],
  "current_batch": "backup-task-executor-authority",
  "next_batch": "backup-public-surface",
  "closeout_status": "executor-authority-validated"
}
```
