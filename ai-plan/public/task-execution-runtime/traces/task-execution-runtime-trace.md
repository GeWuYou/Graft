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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "task-runtime-foundation-authority",
    "task-module-persistence-state-machine"
  ],
  "pending_batches": [
    "task-runtime-worker-and-recovery",
    "task-api-realtime-and-project-adoption",
    "task-web-module-and-project-ui",
    "task-final-integration-archive-readiness"
  ],
  "current_batch": "task-module-persistence-state-machine",
  "next_batch": "task-runtime-worker-and-recovery",
  "closeout_status": "completed_no_handoff"
}
```
