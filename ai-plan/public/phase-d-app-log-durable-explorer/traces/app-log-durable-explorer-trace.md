# App Log Durable Explorer Trace

## 2026-06-04 Startup

- Re-ran root startup preflight.
- Read root `AGENTS.md`, `.ai/environment/tools.ai.yaml`, `server/AGENTS.md`, and `web/AGENTS.md`.
- Classified the task as `cross-boundary`.
- Read `graft-multi-agent-loop`, `graft-multi-agent-task`, `graft-validation-runner`, `graft-web-module-scaffold`, and `graft-web-vibe-coding` workflow requirements.
- Renamed the local branch to `feat/app-log-durable-explorer`.
- Created this active topic from the archived `phase-d-app-log-retention-authz-and-storage-readiness` evidence.

## Batch State

- completed batches:
  - Batch 1 backend approval and runtime foundation
  - Batch 2 OpenAPI and web Explorer
- current batch: Batch 2 OpenAPI and web Explorer
- pending batches:
  - Batch 3 final validation and archive readiness

## 2026-06-04 Batch 1 Backend Foundation

- Approved repository-owned durable App Log runtime foundation under `server/internal/logger/**`.
- Added `app_logs` live migration under `server/internal/logger/migrations/**` and registered `internal/logger/migrations` in the default migration chain.
- Added logger-owned `AppLogRepository` with canonical create/list/delete support for:
  - `occurred_at`
  - `severity`
  - `component`
  - `operation`
  - `request_id`
  - `trace_id`
  - `route`
  - `method`
  - `error`
  - `message`
  - `fields`
- Wired `AppLogger` to preserve zap output and optionally persist best-effort repository records when `GRAFT_LOG_APP_LOG_PERSIST=true`.
- Added `GRAFT_LOG_APP_LOG_RETENTION` with bounded defaults:
  - local/test/dev: 3 days
  - staging: 7 days
  - production: 14 days
- Added `logger.app-log-retention-cleanup` lifecycle owned by logger boundary.
- Did not add read permission/menu/API in Batch 1; existing backend registration pattern should be paired with the Batch 2 OpenAPI + explorer read contract instead of inventing frontend behavior here.
- Validation:
  - `cd server && go test ./internal/logger ./internal/app ./internal/cli ./internal/config ./internal/moduleregistry`
  - `cd server && atlas migrate hash --dir file://internal/logger/migrations`
  - manual migration comment inspection confirmed `app_logs` table and all columns have Chinese comments.

## 2026-06-04 Batch 2 OpenAPI and Web Explorer

- Added logger-owned App Log Explorer read registration under `server/internal/logger/**`.
- Added distinct permission and menu semantics:
  - permission: `app_log.read`
  - menu: `/logs/app`
  - API group: `/api/app-log`
- Added read-only list/detail handlers with unknown query-key rejection.
- Kept query filters bounded to canonical App Log fields:
  - `occurred_from`
  - `occurred_to`
  - `severity`
  - `component`
  - `operation`
  - `request_id`
  - `trace_id`
  - `keyword`
  - `message`
  - `error`
- Added OpenAPI source contracts and regenerated server/frontend OpenAPI artifacts.
- Added `web/src/modules/app-log/**` with module-owned contract, API, bootstrap route, locales, filters, table, detail drawer, page, and focused tests.
- TDesign MCP preflight used for `vue-next` Table, Form, Drawer, Tag, DatePicker, Select, Input, and Button docs.
