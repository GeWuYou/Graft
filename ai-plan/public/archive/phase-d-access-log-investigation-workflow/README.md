# Phase D Access Log Investigation Workflow

## Status

- Topic: `phase-d-access-log-investigation-workflow`
- Status: `archived`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `phase-d-access-log-runtime-storage`
  - `phase-d-access-log-explorer-contract`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/httpx/**` owns canonical access-log runtime, query, and detail semantics
  - `openapi/**` is the shared wire-contract authority for the implemented access-log explorer surface
  - `web/src/modules/access-log/**` and `web/src/modules/audit/**` are downstream consumers and navigation surfaces only
  - `ai-plan/public/**` is the canonical recovery/archive owner for this completed bounded topic

## Implemented Workflow

This bounded slice completed the operator investigation workflow that starts from audit evidence and lands on canonical access-log records without inventing a second request-investigation authority.

Implemented behavior:

1. audit surfaces now deep-link related request investigation into `Access Log` instead of reopening `Audit Log` as the request-fact surface
2. access-log explorer accepts both `request_id` and `trace_id` as canonical seeded filters
3. access-log detail returns `trace_id` and exposes related audit-navigation actions from the canonical access-log record
4. generated server and web contracts align with the implemented request/trace investigation path

## Files Changed

- `ai-plan/design/domains/audit/日志治理开发规范.md`
- `openapi/components/schemas/access-log-detail-response.yaml`
- `openapi/dist/openapi.bundle.json`
- `openapi/paths/access-log.logs.yaml`
- `server/internal/contract/openapi/accesslog/zz_generated.accesslog.go`
- `server/internal/contract/openapi/generated/types.gen.go`
- `server/internal/httpx/accesslog.go`
- `server/internal/httpx/accesslog_explorer.go`
- `server/internal/httpx/accesslog_repository.go`
- `server/internal/httpx/accesslog_repository_test.go`
- `server/internal/httpx/accesslog_test.go`
- `server/internal/httpx/authz.go`
- `server/internal/httpx/migrations/202605300001_access_log_foundation.sql`
- `server/internal/httpx/migrations/atlas.sum`
- `server/internal/httpx/response.go`
- `server/internal/logger/README.md`
- `web/src/contracts/openapi/generated/schema.ts`
- `web/src/modules/access-log/components/AccessLogDetailDrawer.vue`
- `web/src/modules/access-log/components/AccessLogFilters.vue`
- `web/src/modules/access-log/components/AccessLogTable.vue`
- `web/src/modules/access-log/contract/deep-link.ts`
- `web/src/modules/access-log/locales/en-US.json`
- `web/src/modules/access-log/locales/zh-CN.json`
- `web/src/modules/access-log/pages/list/index.vue`
- `web/src/modules/access-log/types/access-log.ts`
- `web/src/modules/audit/components/AuditDetailDrawer.vue`
- `web/src/modules/audit/contract/deep-link.ts`
- `web/src/modules/audit/contract/navigation.ts`
- `web/src/modules/audit/locales/en-US.json`
- `web/src/modules/audit/locales/zh-CN.json`
- `web/src/modules/audit/pages/incident/index.test.ts`
- `web/src/modules/audit/pages/incident/index.vue`
- `web/src/modules/audit/pages/overview/index.vue`

## Validation Commands From Implementation Closeout

- `cd server && go test ./internal/httpx`
- `cd server && go build ./cmd/graft`
- `cd server && go run ./cmd/graft validate backend --stage lint`
- `cd web && bun run check`
- `git diff --check`

## Archive Verdict

- archive-ready status: `confirmed`
- remaining gaps: `none required for bounded topic`

## Commit Gate For This Repair Turn

- this recovery repair turn only restores truthful archive state under `ai-plan/public/**`
- full validation should be rerun before any commit that tries to include both the repaired recovery docs and the implementation slice
