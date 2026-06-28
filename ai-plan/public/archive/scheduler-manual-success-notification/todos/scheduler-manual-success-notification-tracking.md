# Scheduler Manual Success Notification Tracking

## Topic

- Topic: `scheduler-manual-success-notification`
- Status: `archive-ready`
- Goal: deliver an unread in-app notification to the current user when a scheduled task is manually run through the API and succeeds.
- Current branch: `feat/scheduler-manual-success-notification`
- Recovery source: archived parent topic in `ai-plan/public/archive/scheduler-manual-success-notification`

## Authority

- Governance source:
  - root `AGENTS.md`
  - `server/AGENTS.md` for implementation
- Design source:
  - `ai-plan/design/domains/notification/通知中心设计.md`
- Stable cross-module port:
  - `server/internal/moduleapi.NotificationPublisher`
- Scheduler runtime boundary:
  - `server/internal/scheduler` owns domain runtime and notifier interfaces.
  - `server/modules/scheduler` adapts runtime notifications to `moduleapi.NotificationPublisher`.

## Scope

Owned implementation scope:

- `server/internal/scheduler/**`
- `server/modules/scheduler/**`
- focused tests under `server/modules/notification/**` and `server/modules/system-config/**` only if required to prove config or dedupe behavior
- `ai-plan/public/scheduler-manual-success-notification/**`

Non-scope:

- no generic EventBus
- no Notification Center refactor
- no scheduler runtime redesign
- no web changes unless proven necessary
- no OpenAPI changes expected
- no compatibility or legacy notification config key

## Current Design Decisions

- Runtime receives scheduler-domain trigger information, not web/auth objects.
- Recommended trigger model:

```go
type RunTrigger struct {
    Type          TriggerType
    TriggerUserID uint64
}
```

- `TriggerUserID == 0` means no user-bound actor. It must not be treated as broadcast.
- Runtime exposes scheduler-domain hooks:
  - `RunSuccessNotifier.NotifyRunSucceeded(ctx, run, trigger)`
  - `RunFailureNotifier.NotifyRunFailed(ctx, run)`
- Scheduler module adapter publishes `task_succeeded` through `moduleapi.NotificationPublisher`.
- Success notification target is `TargetUser(current_user_id)` only.
- Failure notification continues to use the existing permission-target fan-out behavior.
- Notification config stays authoritative inside notification module:
  - `notification.enabled`
  - `notification.delivery.in_app.enabled`
  - `notification.source.scheduled_task_success.enabled`

## Completed Batches

### Batch 1: Runtime Boundary

- Completed in `87e197c5`.
- Added `RunTrigger`, `RunSuccessNotifier`, and `RunOnceWithTrigger`.
- Preserved failure notifier behavior.
- Added focused runtime tests for manual success, cron success skip, empty-user trigger preservation, and existing failure notification.

### Batch 2: Scheduler Module Adapter

- Completed in `ed331a70`.
- Manual run API handler passes current request user ID through scheduler-domain `RunTrigger`.
- Scheduler module registers `SetRunSuccessNotifier(...)` consistently with existing failure notifier wiring.
- Success adapter publishes event type `task_succeeded`, severity `info`, category `TASK`, source module `scheduler`, navigation kind `SCHEDULER_RUN`, and `target_type=USER`.
- Payload contains `run_id`, `task_key`, and `job_key`.
- Dedupe key is `scheduler:run_succeeded:<run_id>`.
- Added message keys:
  - `scheduledTask.notification.runSucceeded.title`
  - `scheduledTask.notification.runSucceeded.message`
- Publish errors are logged and do not change run status or API response.

### Batch 3: Notification Config And Dedupe Proof

- Completed in `35430d26`.
- Added focused publisher test proving scheduler `task_succeeded` uses `notification.source.scheduled_task_success.enabled`.
- Confirmed disabled source skips persistence and enabled source persists through the existing publisher path.
- Existing key-based dedupe remains sufficient; no dedupe subsystem was added.

### Batch 4: Archive Readiness

- Completed in this closeout.
- Required focused backend tests and backend lint passed.
- Dependency graph confirmed:

```text
scheduler runtime
  -> notifier interfaces
  -> scheduler module adapter
  -> moduleapi.NotificationPublisher
  -> notification module
```

- No circular dependency was introduced:
  - runtime has no notification dependency
  - scheduler module does not import notification internals
  - notification module does not import scheduler module
- Trace updated with validation results and residual risks.

## Required Validation

Implementation validation:

```bash
cd server && go test ./internal/scheduler ./modules/scheduler ./modules/notification ./modules/system-config
cd server && go run ./cmd/graft validate backend --stage lint
```

Run `cd web && bun run check` only if implementation changes OpenAPI, frontend generated types, or web visible copy.

## Current Risks

- The implementation did not require frontend or OpenAPI changes.
- Existing dedupe remains key-based through notification event uniqueness and delivery uniqueness.
- The focused validation passed; broader full backend validation beyond lint was not required by this topic.
