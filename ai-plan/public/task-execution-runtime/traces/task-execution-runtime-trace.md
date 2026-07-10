# Task Execution Runtime Trace

## 2026-07-10 task-runtime-foundation-authority

- Work Intake classified the request as a long-running cross-boundary feature and persisted Work Contract v1.
- Added architecture authority, roadmap and ADR-004 before runtime code.
- Locked `Task / TaskPlan / Stage / StageExecutor` terms; Temporal is design inspiration only, not a product/runtime dependency.
- Defined PostgreSQL as the sole Task truth and Redis as auxiliary only.
- Defined `unknown` Stage plus `needs_attention` Task recovery for interrupted external commands.
- Added canonical Task OpenAPI and moduleapi boundary; no module runtime or migration was introduced.

## Locked Decisions

- Stage rows are authoritative for Stage lifecycle; task events only preserve non-derivable lifecycle/retry/cancel/recovery facts.
- Scheduler submits Tasks but does not execute Stages.
- Task Detail capability flags are authoritative UI controls.

## 2026-07-10 task-module-persistence-state-machine

- Added compile-time `task` module registration and embedded `modules/task/migrations` directory.
- Added append-only Task Runtime fact tables: `tasks`, `task_stages`, `task_events`, and `task_logs`, including Chinese SQL comments, serial plan constraints, ordered replay constraints, and active-owner uniqueness.
- Added module-owned SQL repository with atomic plan persistence, compare-and-swap Task/Stage transitions, and event/log replay reads.
- Added Task and Stage state-machine validation; Stage lifecycle remains authoritative in `task_stages`, while the initial `created` event and later non-derivable events belong to `task_events`.
- No worker, dispatcher, executor registry, API, realtime topic, Project integration, or automatic recovery was added.
- Validation passed: migration SQL/version gates, focused Task/module-registry tests, backend lint, and full `graft validate backend`.

## 2026-07-10 task-runtime-worker-and-recovery

- Added a PostgreSQL-backed serial Stage dispatcher and fixed in-process worker pool to `server/modules/task`; Redis is not read or written by the correctness path.
- Added consumer-facing `TaskService` submission, cancellation and operator-approved Stage retry, plus a `TaskRuntimeRegistrar` for consumer-owned `StageExecutor` registration.
- Stage plans are validated against the registered executors before they are persisted. Workers use `FOR UPDATE SKIP LOCKED` on PostgreSQL to claim one serially eligible Stage; SQLite retains equivalent test-only behavior.
- Cancellation is cooperative: a running executor receives its `Cancel` hook and a cancelled context; pending/scheduled and needs-attention Tasks can finalize without an executor.
- Recovery distinguishes policies: interrupted `manual_reconcile` Stages become `unknown` and their Task becomes `needs_attention` with `recovery_required`; explicitly idempotent Stages return to pending for controlled retry.
- Added direct regression tests for serial completion, retryable execution, operator retry, cancellation from needs-attention, and non-resumable crash recovery.
- No HTTP, realtime, OpenAPI, Project executor, web, scheduler, Redis, MQ, or distributed worker change was made.

## 2026-07-10 task-api-realtime-and-project-adoption

- Added owner-authorized Task list/detail/stage/event/log/cancel/retry HTTP routes and a reusable `task:{id}` realtime topic. Task facts and executor output are persisted before their corresponding realtime notification is published.
- Project registers the Compose Stage executors and its `compose_project` owner authorizer; lifecycle actions submit frozen TaskPlans and return the accepted Task receipt with HTTP 202.
- Canonical OpenAPI lifecycle responses now reference `enveloped-task-receipt`; the bundle and generated backend types were regenerated.

## 2026-07-10 task-web-module-and-project-ui

- Added the `web/src/modules/task` presentation boundary with a reusable Task Detail Drawer and Project-scoped Task History.
- Lifecycle submission receipts now open the shared Drawer from both Project list and Project detail instead of waiting for a final Compose response.
- The Drawer seeds task detail and log history through HTTP, then subscribes to `task:{id}`. Each realtime notification triggers durable detail and incremental-log backfill, so reconnects and missed notifications do not lose task facts.
- The generic Task API was already canonical in `openapi/**`; regenerated the frontend schema so API wrappers consume generated Task types rather than manual transport DTOs.
- Task capabilities are rendered directly from server-owned detail data for cancel, retry and log download controls.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "task-runtime-foundation-authority",
    "task-module-persistence-state-machine",
    "task-runtime-worker-and-recovery",
    "task-api-realtime-and-project-adoption",
    "task-web-module-and-project-ui"
  ],
  "pending_batches": [
    "task-final-integration-archive-readiness"
  ],
  "current_batch": "task-web-module-and-project-ui",
  "next_batch": "task-final-integration-archive-readiness",
  "closeout_status": "completed_no_handoff"
}
```
