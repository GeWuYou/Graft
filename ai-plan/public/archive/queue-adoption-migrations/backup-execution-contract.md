# Backup Execution Contract

## Scope

This topic batch introduces the first product-facing Backup execution entry. It
adopts the existing Task Runtime without changing Task Runtime ownership or
repurposing the Update runner handoff.

The initial public surface is deliberately limited to:

- submit one manually requested platform backup;
- list and read safe backup summaries and their associated Task receipt;
- use the shared Task detail drawer for execution progress, failure, unknown
  state, and manual reconciliation guidance.

Restore, artifact download, artifact browsing, automatic expiry cleanup, and
scheduled backups are out of scope for this batch.

## Authority And Navigation

| Concern | Owner |
| --- | --- |
| backup plan, artifact paths, integrity facts, and side effects | `server/modules/backup` |
| Task, Stage, logs, status, retry, recovery, and realtime | `server/modules/task` |
| public HTTP wire contract | `openapi/**` |
| visible navigation | Platform capability at `/platform/backups` |
| Task drawer projection | `web/src/modules/task` |

The Backup history is a user-managed Platform capability. Its menu, route, and
permissions are owned by `platform-backup`; it is not a Container, Update, or
runtime implementation entry.

## Public Operation

`POST /api/platform/backups` accepts a manual backup request and returns `202
Accepted` with a Task receipt. The request carries only an idempotency key and
the bounded retention choice defined by the canonical OpenAPI schema: `1d`,
`7d`, or `30d`, with `30d` as the default. This policy applies only to
user-requested Backup Tasks; the Update module retains its independent fixed
30-day pre-update backup policy. The request never
accepts host paths, database DSNs, commands, environment values, artifact
references, or restore inputs.

The module creates a frozen `platform.backup.create.v1` Task with stable owner
`platform_backup:manual`. The plan is serial:

1. `platform.backup.create-artifacts.v1` snapshots the module-approved runtime
   configuration and database into a module-owned artifact root.
2. `platform.backup.record-artifacts.v1` verifies the controlled files and
   persists Backup metadata through `BackupService`.

Both Stage inputs contain only a generated operation identity, an opaque
module-owned artifact root reference, the requested retention deadline, and
the requester identity. They do not expose credentials, paths, configuration,
or dump contents through Task APIs or logs.

## Effects, Failure, And Recovery

Backup creation has irreversible external effects once either artifact is
written. Each stage has exactly one attempt and `manual_reconcile` recovery.
No executor automatically retries an interrupted dump or snapshot. A process
crash while a Stage is running becomes `unknown` and its Task becomes
`needs_attention`; the operator must inspect the controlled artifact root and
database state before starting a new backup.

Known executor failures become `failed` with a stable, non-secret failure code.
Task logs may describe the operation phase and sanitized tool error category,
but must not include commands, DSNs, environment values, artifact paths,
configuration content, or dump content.

The record stage may be manually retried only when its frozen artifacts have
been verified and the Task Runtime presents retry capability. The artifact
creation stage has no automatic or UI retry path in this batch.

## Authorization And Audit

`platform-backup.read` authorizes safe Backup history and Task view access.
`platform-backup.create` authorizes the manual create route and the Task owner
authorizer for cancellation or retry decisions. The Task owner authorizer
rechecks these permissions for every Task API operation; possession of a Task
ID is not authorization.

The create request is audited at the Backup module boundary. Task Runtime logs
and events remain the execution authority. A successful receipt is not a
successful Backup: the summary becomes available only after the record stage
has completed.

## Web Projection

The Backup module uses the `list-form-detail` page master. The history list
contains safe summary fields only. The primary action submits the manual Task,
opens the shared Task drawer, retains the current list state, and refreshes the
history only after the observed Task reports success. Failed and
`needs_attention` tasks remain visible in the drawer; the page does not infer
success from `202`.
