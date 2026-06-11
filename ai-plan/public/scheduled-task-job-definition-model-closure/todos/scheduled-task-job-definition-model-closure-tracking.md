# Scheduled Task Job Definition Model Closure Tracking

## Topic

- Topic: `scheduled-task-job-definition-model-closure`
- Status: `active recovery entry`
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

- The last completed slice was a read-only exploration of Scheduled Task database tables, Job Definition registration,
  OpenAPI DTOs, and the scheduled-task frontend page.
- No code, database, migration, OpenAPI, or frontend implementation files were modified by the exploration slice.
- The exploration found that the current product concept is close to the desired model, but naming and presentation
  boundaries are confusing and should be tightened before more scheduler UI work is added.

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

- `task_key` identifies a task instance.
- `job_key` identifies the code-registered Job Definition that executes the task.
- Builtin tasks often use `task_key == job_key`, but custom/user-created tasks can point many task instances at one
  Job Definition.
- `task_type` currently behaves like historical/redundant data; current code effectively treats it as `job`.
- `params_json` is historical/redundant compared with `config_json`.
- Job Definition `config_schema` and `default_config` belong to the code-registered definition.
- Scheduled Task `config_json` and effective/final config belong to the task instance and execution view.
- Task Run should keep `task_key` and `job_key` snapshots so historical executions remain traceable after future Job
  Definition changes.

## Current Registration Findings

- Main server evidence paths:
  - `server/internal/cronx/registry.go`
  - `server/internal/scheduler/runtime.go`
  - `server/internal/scheduler/repository.go`
  - `server/internal/httpx/accesslog_retention.go`
  - `server/internal/logger/retention.go`
  - `server/modules/audit/retention.go`
- Job Definition fields currently include some mix of:
  - `key`
  - title / title key / name-like display fields
  - description / description key
  - module / owner / source-like fields
  - type-like fields
  - config schema / default config
  - handler / action / runner behavior
- Execution depends on the registered key and handler path. Most title/description fields are display metadata.
- `owner`, `module`, and source-like labels overlap. Current server mapping sets `Owner` and `Module` from the same
  module key, so the concepts are not cleanly separated.

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
- The list page `Job 类型 / Job Type` column resolves display text from `row.job_key` through the Job Definition list
  and `jobDefinitionTitle()`.
- That value is a Job Definition title, not a type/category/executor.
- The scheduled-task page does not currently have a separate presenter/view-model layer. It directly mixes OpenAPI DTO
  aliases, i18n fallback, config merge, detail drawer model construction, and UI display logic.
- The page exposes fallback-heavy display behavior; future work should avoid showing backend fallback strings where an
  i18n key-first display model is expected.

## Recommended Concept Model

- Scheduled Task: user/system-created task instance.
  - Suggested fields: `task_key`, `task_name`, `cron_expression`, `enabled`, `config_json`, `builtin`,
    `created_by`, `updated_by`, `last_run`, `next_run`, `status`.
- Job Definition: code-registered execution definition.
  - Suggested fields: `job_key`, `module`, `category`, `title_key`, `short_title_key`, `description_key`,
    `config_schema`, `default_config`, `handler`.
- Task Run: one execution record.
  - Suggested fields: `run_id`, `task_key`, `job_key`, `trigger_type`, `status`, `started_at`, `finished_at`,
    `duration_ms`, `result_json`, `error_text`.

## Recommended Product Semantics

- Replace list-column meaning of `Job 类型 / Job Type` with one of:
  - `category`
  - `category + module`
  - `shortTitle`
- Keep full Job Definition title, description, and `job_key` in the detail drawer.
- Add stable Job Definition `category` values such as:
  - `retention`
  - `sync`
  - `maintenance`
  - `notification`
  - `report`
  - `workflow`
  - `custom`
- Consider adding `shortTitle` / `displayShortName` for compact list display.
- Preferred list examples:
  - `Retention · HTTPX`
  - `Access Log`
- Preferred detail examples:
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
- Prefer `module` as the Job Definition owning module, for example `core.httpx`, `logger`, or `audit`.
- Do not put a task-level `module` on Scheduled Task unless it represents the creator/owning task domain rather than
  the executable Job Definition module.
- Treat compatibility aliases as exceptions under root `AGENTS.md` authority-first rules; record owner, reason,
  downstream consumers, cleanup trigger, and validation if a bridge is unavoidable.

## Recommended Next Batches

1. Contract/design slice:
   - Decide final DTO names for `category`, `shortTitle` / `short_title_key`, and canonical module/source naming.
   - Decide whether `task_type` and `params_json` become deprecated compatibility fields or are removed in a later
     migration.
   - Record any compatibility exception before implementing it.
2. Server database/API slice:
   - Add or backfill Job Definition category and short-title metadata.
   - Keep `task_key` and `job_key` separate in storage and API.
   - Ensure Task Run records preserve execution-time `task_key` and `job_key` snapshots.
3. Frontend presenter/view-model slice:
   - Introduce a scheduled-task presenter/view-model boundary.
   - Move list display logic out of the Vue page.
   - Show category/module/short title in lists and full Job Definition metadata in detail.
4. Compatibility cleanup slice:
   - Remove or hide misleading `Job Type` wording.
   - Clean excessive fallback exposure.
   - Reduce redundant `owner/module/source` displays.

## Immediate Next Step

Start a cross-boundary planning or implementation slice from this topic. The first concrete decision should be the
canonical display contract: `category`, `shortTitle`, and `module` semantics for Job Definitions, plus the deprecation
path for misleading `Job Type` naming.
