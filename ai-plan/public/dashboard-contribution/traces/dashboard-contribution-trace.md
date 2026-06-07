# Dashboard Contribution Trace

## 2026-06-07 - Topic Setup

- Branch renamed from `feat/system-configuration` to `feat/dashboard-contribution`.
- Startup receipt established:
  - governance source: root `AGENTS.md`
  - task class: `cross-boundary`
  - recovery source: `parent topic`
  - authority summary: `server` runtime/module registries declare Dashboard widget contributions; `openapi/**` owns the wire contract; `web` consumes generated OpenAPI types and renders generic dashboard widgets.
- Final architecture decision:
  - MVP implementation starts in `server/internal/dashboard`.
  - The internal package is limited to registry, definitions, loader contract, and aggregate route.
  - Future dashboard persistence, layout, presets, favorites, recent visits, and preferences should move to a future `server/modules/dashboard`.
- Final widget contract decision:
  - Use `type + payload`.
  - Avoid `oneOf` and typed-slot payloads for MVP because current `openapi-typescript` and `oapi-codegen` generation would add avoidable complexity.
- Initial loop budget:
  - loop mode: `topic-completion-loop`
  - max rounds: 5
  - max commits: 5
  - max runtime: bounded by active session
  - validation failure policy: stop on directly affected validation failure

## 2026-06-07 - Phase 1 Backend Registry And Core Widget

- Implemented `server/internal/dashboard` as the MVP contribution surface:
  - registry validation and duplicate widget id rejection
  - widget definition, loader contract, type/size/status enums
  - authenticated aggregate routes for `/api/dashboard/summary` and `/api/dashboard/widgets/{widget_id}`
  - server-side required permission filtering
  - per-widget loader timeout, panic recover, and non-fatal error widget state
- Wired `DashboardRegistry` into `module.Context` from `server/internal/app/runtime.go`.
- Registered first core widget:
  - id: `core.module-runtime-health`
  - module_key: `core`
  - type: `health`
  - required_permissions: `modules.runtime.read`
  - source: existing module runtime snapshot.
- Added OpenAPI source, bundled spec, root Go generated types, dashboard narrow generated types, and web generated schema.
- Added direct tests for:
  - registry duplicate and validation behavior
  - registry ordering
  - permission filtering
  - loader error, panic, and timeout handling
  - dashboard route smoke behavior
  - OpenAPI route coverage.
- Validation passed:
  - `cd server && go test ./internal/dashboard ./internal/module ./internal/app ./internal/contract/openapi ./internal/contract/openapi/dashboard`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run openapi:types:check`
- Notes:
  - `server/go.mod` and `server/go.sum` now include `github.com/santhosh-tekuri/jsonschema/v6 v6.0.2` because the existing `go tool oapi-codegen` chain for `github.com/getkin/kin-openapi v0.140.0` required that module metadata before repository OpenAPI generation could run.
  - The existing backend OpenAPI freshness stage does not yet include the new dashboard narrow generated package; the package is still generated through `go generate ./internal/contract/openapi/dashboard` and covered by focused tests.
- Commit: Phase 1 scope committed through `$graft-commit`; see loop closeout for short SHA.
