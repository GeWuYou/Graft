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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["task-runtime-foundation-authority"],
  "pending_batches": [
    "task-module-persistence-state-machine",
    "task-runtime-worker-and-recovery",
    "task-api-realtime-and-project-adoption",
    "task-web-module-and-project-ui",
    "task-final-integration-archive-readiness"
  ],
  "current_batch": "task-runtime-foundation-authority",
  "next_batch": "task-module-persistence-state-machine",
  "closeout_status": "completed_no_handoff"
}
```
