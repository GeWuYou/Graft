# Task 执行运行时实施计划

## 目标

在模块化单体内建立 `server/modules/task` 平台级 Task Runtime，使 Project、Image、Migration、Backup 等业务 module 能提交含多个 Stage 的持久化 Task，而不重复实现状态、worker、日志或 realtime。

## 分批路线

1. `task-runtime-foundation-authority`
   - Work Intake、设计、ADR、OpenAPI、`moduleapi` contract、active topic。
2. `task-module-persistence-state-machine`
   - `task` module scaffold、tasks/stages/events/logs migration、repository、状态机和 plan persistence。
3. `task-runtime-worker-and-recovery`
   - dispatcher、worker、retry/cancel、owner active constraint、unknown/needs_attention recovery。
4. `task-api-realtime-and-project-adoption`
   - generic API/topic、Project Compose StageExecutor、现有 lifecycle route 改为 `202` Task receipt。
5. `task-web-module-and-project-ui`
   - Task Detail Drawer、history、HTTP replay + WebSocket merge、Project active task entry。
6. `task-final-integration-archive-readiness`
   - cross-boundary validation、retention/observability review、acceptance and archive-readiness decision。

## Hard constraints

- PostgreSQL is source of truth; Redis is auxiliary only.
- `scheduler` may submit Task but never executes Stage.
- Task Runtime is a state-machine runtime, not a generic function queue.
- No MQ, Temporal Server, distributed worker, or unproven DAG model.
- `task` has no business-module dependency; consumers use `server/internal/moduleapi`.

## Acceptance

- A Project redeploy produces a persisted Task with replayable stages and logs.
- Refresh/reconnect preserves the Task detail from PostgreSQL.
- Crash leaves an indeterminate external command as Stage `unknown` and Task `needs_attention`.
- Task API capability fields, owner authorization, audit and realtime ticket scope are explicit.
- At least one second consumer is designed before claiming the contract is general, but not required for MVP completion.
