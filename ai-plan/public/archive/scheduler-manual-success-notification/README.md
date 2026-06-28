# Scheduler Manual Success Notification

## Current State

- Status: `archive-ready`
- Task class for implementation: `server`
- Current branch: `feat/scheduler-manual-success-notification`
- Default recovery source:
  - `ai-plan/public/archive/scheduler-manual-success-notification/todos/scheduler-manual-success-notification-tracking.md`
  - `ai-plan/public/archive/scheduler-manual-success-notification/traces/scheduler-manual-success-notification-trace.md`
- Design authority:
  - `ai-plan/design/domains/notification/通知中心设计.md`
  - `server/internal/moduleapi.NotificationPublisher`
  - `server/internal/scheduler` runtime notifier boundary

## Goal

Implement in-app notification delivery for successful manual scheduled task runs.

The target behavior is:

- API-triggered manual run success sends one unread in-app notification to the triggering user.
- Cron-triggered success does not notify.
- Run failure notification behavior stays unchanged.
- Notification publish errors are logged and never change task run status or API response.
- Notification bell and `/notifications` can read the resulting delivery through existing notification APIs.

## Archive Summary

- Runtime boundary implemented in `87e197c5`.
- Scheduler module publisher adapter implemented in `ed331a70`.
- Notification publisher source-switch proof committed in `35430d26`.
- Required backend-focused tests and lint passed.
- Dependency graph confirmed:
  - `server/internal/scheduler` imports no Gin, `httpx`, auth module, notification module, or notification DTO.
  - `server/modules/scheduler` uses `moduleapi.NotificationPublisher` as its notification publishing boundary and does not import notification internals.
  - `server/modules/notification` does not import scheduler.
  - no compile-time cycle was introduced.

## Scope

Owned scope for the implementation loop:

- `server/internal/scheduler/**`
- `server/modules/scheduler/**`
- focused tests under `server/modules/notification/**` and `server/modules/system-config/**` only when needed
- `ai-plan/public/scheduler-manual-success-notification/**`

Explicit non-scope:

- no generic EventBus
- no scheduler runtime redesign
- no Notification Center refactor
- no frontend changes unless implementation proves the existing notification API or navigation contract is insufficient
- no OpenAPI changes expected
- no legacy or compatibility config key

## Dependency Graph Target

```text
scheduler runtime
  -> notifier interfaces
  -> scheduler module adapter
  -> moduleapi.NotificationPublisher
  -> notification module
```

The implementation must confirm that:

- `server/internal/scheduler` imports no `httpx`, Gin, auth module, notification module, or notification DTO.
- Scheduler module uses `moduleapi.NotificationPublisher` as its only notification dependency.
- Notification module does not import scheduler module.
- No compile-time cycle is introduced.

## Suggested Multi-Agent Loop Prompt

```text
Use $graft-multi-agent-loop in topic-completion-loop mode.

Startup receipt:
- governance source: root AGENTS.md
- task class: server
- recovery source: parent topic ai-plan/public/scheduler-manual-success-notification
- owned scope: server/internal/scheduler/**, server/modules/scheduler/**, focused tests under server/modules/notification/** and server/modules/system-config/** only if needed, ai-plan/public/scheduler-manual-success-notification/** for trace updates

Task:
Implement manual scheduled task success in-app notifications according to ai-plan/design/domains/notification/通知中心设计.md.

Constraints:
- Do not introduce EventBus.
- Do not redesign scheduler runtime.
- Do not refactor Notification Center.
- Runtime must not import httpx, Gin, auth, notification module, or notification DTOs.
- Scheduler module may use moduleapi.NotificationPublisher as the only notification dependency.
- No notification -> scheduler compile-time dependency.
- Cron success must not notify.
- Manual success targets current user only; missing user skips with diagnostic logging.
- Notification publish failure must not affect run status or API response.

Implementation refinement:
- Prefer RunTrigger with Type and TriggerUserID.
- Prefer RunSuccessNotifier and RunFailureNotifier as scheduler-domain hooks.
- Keep notifier registration consistent with SetRunFailureNotifier(...) and SetRunSuccessNotifier(...).
- If a shared run-completion path exists, dispatch failed -> failure notifier, manual success -> success notifier, cron success -> no notification.

Loop budget:
- loop_mode=topic-completion-loop
- max_rounds=4
- max_commits=2
- max_runtime_minutes=90
- soft_timeout_minutes=30
- checkpoint_budget=1
- validation failure policy: stop-on-failure after one retry
- allowed_scopes as above

Initial pending batches:
1. Runtime boundary: add RunTrigger and RunSuccessNotifier, route manual trigger through completion path, tests.
2. Scheduler module adapter: current user extraction, success publisher input, message keys, tests.
3. Notification config/dedupe validation tests if gaps remain.
4. Archive-readiness: focused tests, backend lint, dependency graph confirmation, topic trace update.

Required validation:
cd server && go test ./internal/scheduler ./modules/scheduler ./modules/notification ./modules/system-config
cd server && go run ./cmd/graft validate backend --stage lint
```
