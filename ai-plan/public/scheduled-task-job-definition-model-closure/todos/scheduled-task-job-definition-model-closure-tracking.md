# Scheduled Task Job Definition Model Closure Tracking

## Topic

- Topic: `scheduled-task-job-definition-model-closure`
- Status: `implemented pending archive review`
- Goal: close the concept and data-model boundary between Scheduled Task instances, code-registered Job Definitions,
  and Task Run execution records.
- Recovery source: read-only exploration completed on 2026-06-11.
- Current worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-audit-plugin-mvp`

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/design/数据库表设计与迁移规范.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/前端架构设计.md`

## Current Recovery Point

- The destructive model-closure slice has been implemented across scheduler migrations, backend runtime/repository/API
  mapping, OpenAPI schemas/generated contracts, and the scheduled-task frontend page.
- Scheduled Task and Job Definition are now separated across storage, execution, API response shape, and UI presenter
  semantics.
- The recovery topic should now be archived or closed after the implementation commit lands.

## Current Evidence

- Database tables found in scheduler migrations:
  - `scheduled_tasks`
  - `scheduler_job_definitions`
  - `scheduler_task_runs`
- Representative local data observed during read-only SELECT checks:
  - builtin task `httpx.access-log-retention-cleanup` points to the same `job_key`
  - builtin task `logger.app-log-retention-cleanup` points to the same `job_key`
  - builtin task `audit.audit-log-retention-cleanup` points to the same `job_key`
  - custom task `wq111` points to `audit.audit-log-retention-cleanup`
- The custom `wq111` row proves `task_key` and `job_key` are already materially separate in the current data model.
- Recent `scheduler_task_runs` rows included manual and cron retention runs, including manual success for `wq111`.
- No sensitive values were found in the sampled task/run JSON; observed JSON contained retention config and result data.

## Current Model Findings

- `task_key` identifies a Scheduled Task instance.
- `job_key` identifies the code-registered Job Definition that executes the task.
- Builtin tasks can use `task_key == job_key`; custom/user-created tasks can point many task instances at one Job
  Definition.
- `task_type` and `params_json` were removed from scheduler storage/API surfaces.
- task-level `owner/module/module_key` duplication was removed from Scheduled Task API output; module ownership comes
  from the bound Job Definition.
- Job Definition now owns `module_key`, `category`, `short_title`, `config_schema`, `default_config`, `default_cron`,
  and default enabled semantics.
- Scheduled Task owns `config_json`, `config_source`, `cron_expression`, enabled/builtin state, title/description, and
  effective config at read time.
- Task Run keeps execution-time snapshots of task/job title metadata, `job_category`, `module_key`, `task_builtin`,
  result, and `error_message`.

## Current Registration Findings

- Main server evidence paths:
  - `server/internal/cronx/registry.go`
  - `server/internal/scheduler/runtime.go`
  - `server/internal/scheduler/repository.go`
  - `server/internal/httpx/accesslog_retention.go`
  - `server/internal/logger/retention.go`
  - `server/modules/audit/retention.go`
- `cronx.Job` now uses explicit Job Definition metadata: key, module key, category, title/title key, short title/short
  title key, description/description key, config schema, default config, schedule/default cron, default enabled state,
  handler, and actions.
- The builtin retention jobs use `category=retention` with short-title keys for access log, app log, and audit log
  compact display.
- Execution resolves the registered Job Definition by `task.job_key` and writes Task Run snapshots for both task and
  job metadata.

## Current API And Frontend Findings

- OpenAPI/DTO evidence paths:
  - `openapi/components/schemas/scheduled-task-item.yaml`
  - `openapi/components/schemas/scheduled-task-job-definition-item.yaml`
  - `openapi/components/schemas/scheduled-task-run-item.yaml`
  - `openapi/components/schemas/create-scheduled-task-request.yaml`
  - `openapi/components/schemas/update-scheduled-task-request.yaml`
  - `server/modules/scheduler/mapper_http.go`
- Frontend evidence paths:
  - `web/src/modules/scheduled-task/pages/list/index.vue`
  - `web/src/modules/scheduled-task/types/scheduled-task.ts`
- OpenAPI no longer exposes scheduler `owner`, duplicate `module`, `task_type`, `params_json`, or duplicate `error`
  fields on Scheduled Task, Job Definition, or Task Run response schemas.
- `ScheduledTaskItem` includes a nested Job Definition summary for display metadata instead of scattering duplicated job
  fields across the task instance.
- The scheduled-task page now has a presenter layer at
  `web/src/modules/scheduled-task/presenter/scheduled-task-presenter.ts`.
- The list no longer renders `Job 类型 / Job Type`; it displays compact category/module information through the
  presenter.
- The detail drawer is grouped into task instance, job definition, configuration, and run information sections.

## Implemented Concept Model

- Scheduled Task: user/system-created task instance.
  - Fields include `task_key`, `job_key`, title/description metadata, `cron_expression`, `enabled`, `builtin`,
    `config_json`, `config_source`, timestamps, `last_run`, `next_run`, and effective config in read models.
- Job Definition: code-registered execution definition.
  - Fields include `job_key`, `module_key`, `category`, title/short-title/description metadata, `config_schema`,
    `default_config`, `default_cron`, `default_enabled`, enabled state, and actions.
- Task Run: one execution record.
  - Fields include `task_key`, `job_key`, task/job title snapshots, job short title, job category, module key,
    task builtin state, trigger/status/timing, result summary/json, and `error_message`.

## Implemented Product Semantics

- The scheduled-task list uses category/module-oriented compact display.
- Full Job Definition title, short title, description, category, module, default cron, config schema, and default config
  are shown in the detail drawer.
- Stable Job Definition `category` values are:
  - `retention`
  - `sync`
  - `maintenance`
  - `notification`
  - `report`
  - `workflow`
  - `custom`
- `short_title` and `short_title_key` were added for compact list display.
- List examples:
  - `Retention · HTTPX`
  - `Access Log`
- Detail examples:
  - task name: `访问日志保留清理`
  - category: `日志保留`
  - source module: `HTTPX`
  - Job Definition: `Access log retention cleanup`
  - Job Key: `httpx.access-log-retention-cleanup`

## Guardrails

- Do not rename a Job Definition title as `Job Type`.
- Do not collapse `task_key` and `job_key`; multiple task instances must be able to point to one Job Definition.
- Do not add new `owner`, `module`, `source_module`, or `namespace` synonyms without first choosing a canonical
  authority concept.
- Use `module_key` as the canonical Job Definition owning module field, for example `core.httpx`, `core.logger`, or
  `audit`.
- Do not put a task-level `module_key` on Scheduled Task unless it represents a distinct future task-instance ownership
  concept rather than the executable Job Definition module.
- Treat compatibility aliases as exceptions under root `AGENTS.md` authority-first rules; record owner, reason,
  downstream consumers, cleanup trigger, and validation if a bridge is unavoidable.

## Completed Validation

- Focused backend tests:
  `cd server && go test ./internal/cronx ./internal/scheduler ./modules/scheduler ./internal/httpx ./internal/logger ./modules/audit`
- Backend completion entrypoint:
  `cd server && go run ./cmd/graft validate backend`
- Focused scheduled-task frontend test:
  `cd web && bun run vitest run src/modules/scheduled-task/pages/list/index.test.ts`
- Frontend completion entrypoint:
  `cd web && bun run check`
- Old concept scan over owned scheduler/OpenAPI/web scope:
  no matches for `Job 类型`, `Job Type`, `task_type`, `params_json`, `owner`, `source_module`,
  `display_name_key`, `config_schema_json`, `default_config_json`, `default_cron_expression`, `schedule_type`, or
  `error_summary`.

## Recommended Next Step

Archive or close this recovery topic after the implementation commit lands.
