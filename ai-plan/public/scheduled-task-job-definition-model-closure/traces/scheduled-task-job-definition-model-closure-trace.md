# Scheduled Task Job Definition Model Closure Trace

## 2026-06-11 Active Topic Created

- Created the active recovery topic `scheduled-task-job-definition-model-closure` from a read-only exploration request.
- Preserved the exploration as recovery material rather than changing code, database schema, migrations, OpenAPI, or web
  implementation files.
- Updated the public recovery index so this topic has a stable default entry.

## 2026-06-11 Read-Only Exploration Summary

- Confirmed the scheduler model currently spans:
  - `scheduled_tasks`
  - `scheduler_job_definitions`
  - `scheduler_task_runs`
- Confirmed local data already separates task instances from Job Definitions:
  - builtin retention tasks commonly use identical `task_key` and `job_key`
  - custom task `wq111` points to builtin Job Definition `audit.audit-log-retention-cleanup`
- Confirmed sampled scheduled task and task run JSON did not expose sensitive data.
- Confirmed current list UI label `Job 类型 / Job Type` is misleading because the rendered value is a Job Definition
  title resolved from `job_key`.
- Confirmed server mapper semantics currently overlap `owner` and `module`.
- Confirmed frontend scheduled-task list page lacks a separate presenter/view-model boundary and mixes DTO consumption,
  fallback display, i18n, config merge, and detail model assembly.

## Key Decision For Next Slice

- Treat Scheduled Task as the task instance.
- Treat Job Definition as the code-registered execution definition.
- Treat Task Run as the historical execution record.
- Preserve `task_key` / `job_key` separation.
- Introduce or formalize `category` and compact display metadata for Job Definition list display.
- Keep full Job Definition title/description/key in details rather than in the list's compact type column.

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
