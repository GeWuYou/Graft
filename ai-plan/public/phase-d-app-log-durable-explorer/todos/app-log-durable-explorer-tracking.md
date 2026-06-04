# App Log Durable Explorer Tracking

## Current Status

- Status: `archive-ready`
- Branch: `feat/app-log-durable-explorer`
- Loop mode: `topic-completion-loop`

## Batches

- [x] Batch 1: backend approval and runtime foundation
  - Approve repository-owned durable App Log storage for this topic.
  - Add logger-owned schema/migration, repository, runtime sink wiring, cleanup lifecycle, and focused tests.
  - Permission/menu registration was not added in Batch 1 because the existing route/menu registration pattern is tied to read API registration; Batch 2 must define the read permission/menu with the OpenAPI + explorer contract.
- [x] Batch 2: OpenAPI and web Explorer
  - Add canonical read contracts and generated types.
  - Add `web/src/modules/app-log/**` list/detail troubleshooting UI with route/menu/permission/i18n boundaries.
- [x] Batch 3: final validation and archive readiness
  - Run required cross-boundary validation.
  - Update governance evidence and closeout this topic as archive-ready when acceptance passes.

## Acceptance Criteria

- App Log durable storage belongs to `server/internal/logger/**`.
- App Log read permission is distinct from access/audit permissions. Batch 2 must use a logger-owned permission such as `app_log.read`; it must not reuse `access_log.read` or `audit.read`.
- App Log Explorer supports bounded time, severity, component, operation, request ID, trace ID, and message keyword filtering.
- App Log detail shows only canonical App Log runtime troubleshooting fields.
- Cross-boundary validation is run and reported truthfully.

## Batch 3 Evidence

- Start state:
  - clean working tree at committed Batch 2 state `1d06b7e`
- Full cross-boundary validation:
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- Generated artifact sync:
  - backend OpenAPI freshness checks reported App Log generated bindings fresh through `graft validate backend`
  - `cd web && bun run openapi:types:check`
- Manual migration comment check:
  - `server/internal/logger/migrations/202606040001_app_log_foundation.sql`
  - `app_logs` table comment present
  - all 12 columns have Chinese comments
- Archive verdict:
  - `archive-ready`

## Batch 1 Evidence

- Durable table: `server/internal/logger/migrations/202606040001_app_log_foundation.sql`
- Migration checksum: `server/internal/logger/migrations/atlas.sum`
- Runtime wiring:
  - `server/internal/logger/storage_repository.go`
  - `server/internal/logger/retention.go`
  - `server/internal/logger/applog.go`
  - `server/internal/app/runtime.go`
- Focused validation:
  - `cd server && go test ./internal/logger ./internal/app ./internal/cli ./internal/config ./internal/moduleregistry`
  - `cd server && atlas migrate hash --dir file://internal/logger/migrations`
- Manual migration comment check:
  - `app_logs` table comment present
  - all 12 columns have Chinese comments

## Batch 2 Evidence

- App Log read permission: `app_log.read`
- App Log menu path: `/logs/app`
- App Log API paths:
  - `GET /api/app-log`
  - `GET /api/app-log/{id}`
- Canonical filters:
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
- OpenAPI source:
  - `openapi/paths/app-log.logs.yaml`
  - `openapi/paths/app-log.detail.yaml`
  - `openapi/components/schemas/app-log-*.yaml`
- Web module:
  - `web/src/modules/app-log/**`
