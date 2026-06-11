# Scheduled Task Job Definition Model Closure

## Topic

- Topic: `scheduled-task-job-definition-model-closure`
- Status: `active recovery entry`
- Goal: converge Scheduled Task, Job Definition, and Task Run concepts across database, server registration/API, and web presenter/view-model boundaries.
- Recovery source: read-only exploration from 2026-06-11
- Current worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-audit-plugin-mvp`

## Recovery Entry

- Tracking:
  `ai-plan/public/scheduled-task-job-definition-model-closure/todos/scheduled-task-job-definition-model-closure-tracking.md`
- Trace:
  `ai-plan/public/scheduled-task-job-definition-model-closure/traces/scheduled-task-job-definition-model-closure-trace.md`

## Startup Package For Future Sessions

- governance source: root `AGENTS.md`
- task class: `cross-boundary` for implementation slices touching server, OpenAPI, and web
- recovery source: parent topic `scheduled-task-job-definition-model-closure`
- owned scope:
  - `server/internal/scheduler/**`
  - `server/internal/cronx/**`
  - `server/modules/scheduler/**`
  - `openapi/components/schemas/scheduled-task*`
  - `openapi/paths/scheduled-tasks*`
  - `web/src/modules/scheduled-task/**`
- docs recovery scope:
  - `ai-plan/public/scheduled-task-job-definition-model-closure/**`
  - `ai-plan/public/README.md`

## Current Conclusion

The current UI label `Job 类型 / Job Type` is misleading. It displays the Job Definition title resolved from `job_key`,
not a stable type, category, executor type, or task class. The next implementation should preserve the existing
`task_key` / `job_key` separation, introduce a stable display category and optional short title for Job Definitions,
and move list/detail presentation logic into a clearer frontend presenter/view-model boundary.
