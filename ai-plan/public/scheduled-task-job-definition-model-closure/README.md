# Scheduled Task Job Definition Model Closure

## Topic

- Topic: `scheduled-task-job-definition-model-closure`
- Status: `implemented pending archive review`
- Goal: converge Scheduled Task, Job Definition, and Task Run concepts across database, server registration/API, and web presenter/view-model boundaries.
- Recovery source: read-only exploration from 2026-06-11; destructive model-closure implementation completed on
  2026-06-11
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

## Remaining Follow-Up

- Archive or close this recovery topic after the implementation commit lands.
