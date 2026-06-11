# Scheduled Task Job Definition Model Closure Trace

## 2026-06-11 Active Topic Created

- Created the active recovery topic `scheduled-task-job-definition-model-closure` from a read-only exploration request.
- Preserved the exploration as recovery material rather than changing code, database schema, migrations, OpenAPI, or web
  implementation files.
- Updated the public recovery index so this topic has a stable default entry.

## 2026-06-11 Read-Only Exploration Summary

- Verified the scheduler model currently spans:
  - `scheduled_tasks`
  - `scheduler_job_definitions`
  - `scheduler_task_runs`
- Observed local data already separates task instances from Job Definitions:
  - builtin retention tasks commonly use identical `task_key` and `job_key`
  - custom task `wq111` points to builtin Job Definition `audit.audit-log-retention-cleanup`
- Checked that sampled scheduled task and task run JSON did not expose sensitive data.
- Identified current list UI label `Job 类型 / Job Type` as misleading because the rendered value is a Job Definition
  title resolved from `job_key`.
- Noted server mapper semantics currently overlap `owner` and `module`.
- Found that the frontend scheduled-task list page lacks a separate presenter/view-model boundary and mixes DTO consumption,
  fallback display, i18n, config merge, and detail model assembly.

## Key Decision For Next Slice

- Treat Scheduled Task as the task instance.
- Treat Job Definition as the code-registered execution definition.
- Treat Task Run as the historical execution record.
- Preserve `task_key` / `job_key` separation.
- Introduce or formalize `category` and compact display metadata for Job Definition list display.
- Keep full Job Definition title/description/key in details rather than in the list's compact type column.

## 2026-06-11 Destructive Model Closure Implemented

- Rewrote the scheduler table model destructively for early development:
  - `scheduled_tasks` now stores task-instance data without `task_type`, `params_json`, or task-level module/owner
    duplication.
  - `scheduler_job_definitions` now stores Job Definition metadata including `module_key`, `category`,
    `short_title_key`, and `short_title`.
  - `scheduler_task_runs` now stores execution snapshots for task/job title metadata, `job_category`, `module_key`,
    `task_builtin`, result data, and `error_message`.
  - Scheduler soft delete columns were aligned to bigint epoch semantics where `0` means live.
- Updated `cronx.Job` and builtin retention job registration so Job Definitions own category, short-title metadata,
  config schema/default config, default cron, and default enabled behavior.
- Updated scheduler repository/runtime/HTTP mapping so:
  - task execution resolves Job Definition by `task.job_key`
  - effective config is computed from Job Definition defaults plus task config overrides
  - Task Runs record both task and Job Definition snapshots
  - OpenAPI output does not expose misleading owner/module/task-type/params fields.
- Updated OpenAPI source and generated server/web contracts.
- Added the scheduled-task presenter boundary at
  `web/src/modules/scheduled-task/presenter/scheduled-task-presenter.ts`.
- Updated the scheduled-task list/detail UI:
  - removed `Job 类型 / Job Type`
  - list display uses compact category/module information
  - details are grouped into task instance, job definition, configuration, and run information
  - immediate run is folded into the More action menu.
- Added i18n keys for Job Definition categories, retention short titles, category/job columns, and detail sections.

## 2026-06-11 Validation

- Passed focused backend tests:
  `cd server && go test ./internal/cronx ./internal/scheduler ./modules/scheduler ./internal/httpx ./internal/logger ./modules/audit`
- Passed backend completion validation:
  `cd server && go run ./cmd/graft validate backend`
- Passed focused scheduled-task frontend test:
  `cd web && bun run vitest run src/modules/scheduled-task/pages/list/index.test.ts`
- Passed frontend completion validation:
  `cd web && bun run check`
- Ran old-concept search over the scheduler/OpenAPI/web owned scope with no matches for removed scheduler concepts.

## Closeout Decision

- The implementation slice is complete and ready for a scoped commit.
- After commit, the remaining administrative follow-up is to archive or close this public recovery topic.

## Future Session Reference

Use the reference below at the start of a new session, then append the new task prompt after it:

```text
【恢复引用：ai-plan/public/scheduled-task-job-definition-model-closure】
请先按 root AGENTS.md 完成 startup preflight，然后读取：
- ai-plan/public/README.md
- ai-plan/public/scheduled-task-job-definition-model-closure/README.md
- ai-plan/public/scheduled-task-job-definition-model-closure/todos/scheduled-task-job-definition-model-closure-tracking.md
- ai-plan/public/scheduled-task-job-definition-model-closure/traces/scheduled-task-job-definition-model-closure-trace.md

task class: cross-boundary
recovery source: parent topic scheduled-task-job-definition-model-closure
owned scope:
- server/internal/scheduler/**
- server/internal/cronx/**
- server/modules/scheduler/**
- openapi/components/schemas/scheduled-task*
- openapi/paths/scheduled-tasks*
- web/src/modules/scheduled-task/**

请基于该主题继续推进 Scheduled Task / Job Definition 概念模型收口。
```
