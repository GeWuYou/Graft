# Scheduler Manual Success Notification Trace

## 2026-06-11 architecture refinement recorded

- Updated `ai-plan/design/domains/notification/通知中心设计.md` with the scheduler notification boundary:
  - scheduler runtime stays domain-only
  - API handler may read current user context, but runtime receives `RunTrigger`
  - runtime uses `RunSuccessNotifier` and `RunFailureNotifier`
  - scheduler module adapter is responsible for publishing through `moduleapi.NotificationPublisher`
- Recorded the intended dependency direction:

```text
scheduler runtime
  -> notifier interfaces
  -> scheduler module adapter
  -> moduleapi.NotificationPublisher
  -> notification module
```

- Recorded that this task must not introduce EventBus, redesign scheduler runtime, or refactor Notification Center.
- Recorded dedupe semantics as key-based notification event uniqueness and delivery uniqueness rather than time-window dedupe.
- Recorded notification permission guidance:
  - do not add one receive permission per notification type
  - use RBAC eligibility, target, and future preference/policy layers
  - manual task success is personal feedback and should target only the triggering user

## Next Implementation Recovery

- Topic is archive-ready after the implementation loop.
- No next implementation batch remains for this topic.

## 2026-06-11 implementation loop completed

- Batch 1 committed `87e197c5 feat(scheduler): add manual run success trigger boundary`.
  - Added scheduler-domain `RunTrigger`, `RunSuccessNotifier`, and `RunOnceWithTrigger`.
  - Manual success notifies through runtime hook; cron success does not notify.
  - Existing failure notifier behavior remains intact.
- Batch 2 committed `ed331a70 feat(scheduler): publish manual run success notifications`.
  - Manual run API handler reads the current user from `moduleapi.RequestAuthContext`.
  - Scheduler module adapts successful manual runs into `moduleapi.PublishNotificationInput`.
  - Success notifications target only the triggering user.
  - Missing user skips with diagnostic logging.
  - Publish failure is logged and does not affect run status or API response.
- Batch 3 committed `35430d26 test(notification): cover scheduler success source switch`.
  - Added focused publisher coverage for `notification.source.scheduled_task_success.enabled`.
  - Confirmed disabled scheduler success source skips persistence and enabled source persists.
  - Kept existing key-based dedupe semantics; no new dedupe subsystem was added.

Validation:

```bash
cd server && go test ./internal/scheduler ./modules/scheduler ./modules/notification ./modules/system-config
cd server && go run ./cmd/graft validate backend --stage lint
```

Both commands passed on 2026-06-11.

Dependency confirmation:

```bash
cd server && go list -deps ./internal/scheduler | rg 'graft/server/(internal/httpx|modules/auth|modules/notification)|github.com/gin-gonic/gin' || true
cd server && go list -deps ./modules/notification | rg '^graft/server/modules/scheduler' || true
```

Both checks returned no matches. `server/modules/scheduler` imports `server/internal/moduleapi` and does not import
notification module internals.

Remaining risks:

- None requiring another implementation batch.
- Broader full backend validation beyond the required focused-scope tests and backend lint was not run for this topic.
