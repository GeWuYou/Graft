# Task Execution Runtime

## Current Status Summary

- Topic objective: establish and deliver the platform Task Runtime for multi-stage business operations.
- Current status: `active`.
- Task class: `cross-boundary`.
- Intake summary: long-running feature requiring design, roadmap, ADR, active topic and loop execution.
- Canonical authority: `ai-plan/design/architecture/任务执行运行时设计.md`, `ADR-004`, `openapi/**`, and `server/internal/moduleapi/task.go`.
- Completed so far: foundation authority batch.
- Not started yet: module persistence/runtime implementation and consumer adoption.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: Task Runtime design + ADR + OpenAPI + moduleapi contract; `task` is a module and Project is its first consumer.

## Owned Scope

- `ai-plan/design/architecture/任务执行运行时设计.md`
- `ai-plan/roadmap/Task执行运行时实施计划.md`
- `ai-plan/design/decisions/ADR-004-task-runtime-state-machine.md`
- `ai-plan/public/task-execution-runtime/**`
- `openapi/**`, `server/internal/moduleapi/**`, future `server/modules/task/**`, Project adoption, and future `web/src/modules/task/**`

Out of scope:

- Temporal Server, MQ, distributed workers or DAG authoring.
- Replacing Scheduler or Container runtime authority.

## Locked Decisions

1. Use Task/TaskPlan/Stage/StageExecutor product and code terms, not Workflow/Activity.
2. PostgreSQL is sole truth; running Docker Stage after crash is `unknown`, parent Task is `needs_attention`.
3. `task` owns state, runner, logs, events and realtime; consumers own business plans and executors.

## Phase Plan

- `task-runtime-foundation-authority` completed.
- `task-module-persistence-state-machine` next.
- `task-runtime-worker-and-recovery`.
- `task-api-realtime-and-project-adoption`.
- `task-web-module-and-project-ui`.
- `task-final-integration-archive-readiness`.

## Current Recovery Point

- Foundation design, typed contracts and OpenAPI are committed before runtime work begins.
- Next step: implement task module persistence and state machine only; do not introduce worker or Project executor yet.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is stored in the tracking file.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
node scripts/openapi-bundle.mjs
```

## Loop Entry

- Preferred entry: `ai-plan/public/task-execution-runtime/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
