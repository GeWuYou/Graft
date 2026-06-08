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

## 2026-06-07 - Phase 2 Web Renderer And Home Route Integration

- Implemented `web/src/modules/dashboard` as the frontend dashboard module:
  - OpenAPI-derived API client for `GET /api/dashboard/summary` and focused widget refresh through `GET /api/dashboard/widgets/{widget_id}`
  - dashboard contract paths and generated-schema type aliases
  - `DashboardHomePage` fixed system summary and generic widget grid
  - `DashboardRenderer` with type-only dispatch for `stat-group`, `alert-list`, `link-list`, `timeline`, and `health`
  - per-widget error, disabled, empty, and focused retry states.
- Replaced the starter `app/home` card with a thin shell wrapper that renders the dashboard module page.
- Added dashboard-owned `zh-CN` and `en-US` locale catalogs and removed unused starter home description/eyebrow keys from root locale catalogs.
- Added focused frontend tests for:
  - dashboard API path usage and widget id encoding
  - renderer ordering, type-based rendering, empty state, and error retry event
  - page summary loading, fixed summary rendering, widget refresh, and page-level error state.
- TDesign MCP preflight:
  - ui_component_change: yes
  - mcp_queried: yes
  - framework: vue-next
  - components: Card, Loading, Empty, Statistic, List, Timeline, Tag, Alert, Button
  - queries: get_component_docs
  - adoption: adopted
  - reason: used queried component props and slot behavior for dashboard cards, loading, empty, list, timeline, tag, alert, and buttons; TDesign `Result` component was unavailable, so dashboard uses `Empty`/`Alert` instead.
- Validation passed:
  - `cd web && bun run test:run -- src/modules/dashboard src/modules/index.test.ts src/router/index.test.ts`
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run typecheck`
  - `cd web && bun run lint:i18n`
  - `cd web && bun run check`
- Notes:
  - The dashboard page does not import audit, scheduler, rbac, user, monitor, or system-config business components.
  - The renderer branches only on `DashboardWidget.type`; widget id and module key remain display/data metadata.
  - No dashboard persistence, preferences, layouts, presets, favorites, recent visits, drag-and-drop, or markdown support was introduced.

## 2026-06-07 - Phase 3 RBAC Access Summary Widget

- Implemented the first business module dashboard contribution:
  - id: `rbac.access-summary`
  - module_key: `rbac`
  - type: `stat-group`
  - required_permissions: `user.read`, `role.read`, `permission.read`
  - payload: users, roles, and permissions stat items with stable key-first labels and fallback strings.
- Kept module boundaries explicit:
  - the widget loader lives in `server/modules/rbac`
  - `server/internal/dashboard` remains generic and does not import RBAC or user internals
  - RBAC reads role and permission counts through its module-owned management reader
  - RBAC reads user count through the stable `moduleapi.UserService.CountUsers` boundary implemented by the user module.
- Added backend tests for:
  - RBAC widget registration when `DashboardRegistry` is present
  - real required permission codes
  - stat-group payload item shape and route locations.
- Validation completed during Phase 3:
  - `cd server && go test ./internal/moduleapi ./modules/user ./modules/rbac`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run test:run -- src/modules/dashboard src/modules/index.test.ts src/router/index.test.ts`
  - `cd web && bun run check`
- Notes:
  - The first backend lint run failed because the RBAC registration edit pushed `Register` over the configured cyclomatic complexity threshold and duplicated zh/en message literal blocks.
  - The lint failure was fixed by extracting RBAC service registration and shared message resource construction without changing widget behavior.
- Commit: Phase 3 scope committed through `$graft-commit`; see loop closeout for short SHA.

## 2026-06-07 - Phase 4 Archive Readiness

- Inspected final worktree and recent commits:
  - `cf68cbb feat(dashboard): add backend contribution registry`
  - `673dfb4 feat(dashboard): add web dashboard renderer`
  - `587e3e2 feat(dashboard): add rbac access summary widget`
- Acceptance checks passed:
  - home dashboard has fixed system summary plus module-contributed widgets
  - dashboard page does not import audit, scheduler, rbac, user, monitor, or system-config business components
  - renderer dispatch remains based on stable `DashboardWidgetType`
  - widget data is loaded through backend aggregation, not frontend N-interface composition
  - server-side permission filtering, loader timeout, error handling, and panic recovery are present
  - Phase 1 required widgets stayed limited to `core.module-runtime-health` and `rbac.access-summary`
  - no dashboard persistence tables were introduced
- Scope leak scan confirmed no MVP support for dashboard preferences, layouts, presets, favorites, recent visits, announcements, quick actions, markdown widgets, or drag-and-drop layout customization.
- Final validation for archive-readiness:
  - `git status --short --branch`
  - `git diff --check`
  - targeted `rg` scans for dashboard imports, renderer branching, registry permissions, loader safety, and forbidden MVP scope terms
- Topic status set to `archive-ready`.
