# Scheduled Task Job Definition Model Closure

## Topic

- Topic: `scheduled-task-job-definition-model-closure`
- Status: `archived`
- Goal: converge Scheduled Task, Job Definition, and Task Run concepts across database, server registration/API, and web presenter/view-model boundaries.
- Recovery source: read-only exploration from 2026-06-11; destructive model-closure implementation completed on
  2026-06-11
- Archive date: `2026-06-12`
- Current worktree: `<repo-root>`

## Recovery Entry

- Tracking:
  `ai-plan/public/archive/scheduled-task-job-definition-model-closure/todos/scheduled-task-job-definition-model-closure-tracking.md`
- Trace:
  `ai-plan/public/archive/scheduled-task-job-definition-model-closure/traces/scheduled-task-job-definition-model-closure-trace.md`

## Startup Package For Future Sessions

- governance source: root `AGENTS.md`
- task class: `cross-boundary` for implementation slices touching server, OpenAPI, and web
- recovery source: archived topic `scheduled-task-job-definition-model-closure`
- owned scope:
  - `server/internal/scheduler/**`
  - `server/internal/cronx/**`
  - `server/modules/scheduler/**`
  - `openapi/components/schemas/scheduled-task*`
  - `openapi/paths/scheduled-tasks*`
  - `web/src/modules/scheduled-task/**`
- docs recovery scope:
  - `ai-plan/public/archive/scheduled-task-job-definition-model-closure/**`
  - `ai-plan/public/README.md`

## Current Conclusion

The destructive model-closure slice has been implemented across scheduler database migrations, backend registration and
repository mapping, OpenAPI schemas/generated contracts, and the scheduled-task frontend. Scheduled Task now represents
the task instance, Job Definition owns execution metadata such as `module_key`, `category`, `short_title`,
`config_schema`, and `default_config`, and Task Run records execution-time task/job snapshots.

The misleading `Job 类型 / Job Type` product wording has been removed from the scheduled-task UI. The list now uses
category/module-oriented compact display through a presenter boundary, and the detail drawer is organized into task
instance, job definition, configuration, and run information sections.

## Validation

- `cd server && go test ./internal/cronx ./internal/scheduler ./modules/scheduler ./internal/httpx ./internal/logger ./modules/audit`
- `cd server && go run ./cmd/graft validate backend`
- `cd web && bun run vitest run src/modules/scheduled-task/pages/list/index.test.ts`
- `cd web && bun run check`

## Final Archive Record

- Archive reason: the Scheduled Task / Job Definition / Task Run concept model closure has been implemented and
  validated, and this topic no longer belongs in the active recovery index.
- Implementation evidence:
  - `8bff6b65 feat(scheduler): close scheduled task job model`
  - `cbd732b7 fix(scheduled-task): refine task list presentation`
- Final status: `archived`.
