# Dashboard Contribution Todos

## Loop Batch State

- completed_batches:
  - Phase 1: backend dashboard registry, aggregate route, OpenAPI source, and core module-runtime widget
  - Phase 2: web dashboard renderer and home route integration
  - Phase 3: RBAC access summary widget and final cross-boundary validation
  - Phase 4: archive-readiness closeout
- pending_batches:
  - none
- current_batch: Phase 4 completed
- next_batch: none
- terminal_status: archive-ready

## Phase 1 - Backend Registry And Core Widget

- [x] Add `server/internal/dashboard` registry, definition, loader, permission filtering, error handling, and tests.
- [x] Inject `DashboardRegistry` into `module.Context` through `server/internal/app/runtime.go`.
- [x] Register authenticated dashboard summary route in the core authenticated route registration path.
- [x] Add OpenAPI source for `GET /api/dashboard/summary` and `GET /api/dashboard/widgets/{widget_id}`.
- [x] Add the `core.module-runtime-health` widget using existing module runtime snapshot data.
- [x] Regenerate required OpenAPI artifacts.
- [x] Run focused backend/OpenAPI validation.
- [x] Commit the validated Phase 1 slice through `$graft-commit`.

## Phase 2 - Web Renderer

- [x] Add `web/src/modules/dashboard` API, contract paths, types, page, renderer, and widget components.
- [x] Render fixed system summary and generic widget grid.
- [x] Support MVP widget types: `stat-group`, `alert-list`, `link-list`, `timeline`, `health`.
- [x] Integrate the dashboard route as the home page without importing module business components.
- [x] Add loading, empty, disabled, and per-widget error states.
- [x] Add dashboard locales and focused frontend tests.
- [x] Commit the validated Phase 2 slice through `$graft-commit`.

## Phase 3 - RBAC Access Summary And Final Validation

- [x] Add `rbac.access-summary` stat-group widget with real permissions: `user.read`, `role.read`, `permission.read`.
- [x] Use module-owned service/repository boundaries; do not let dashboard core import RBAC/user internals.
- [x] Validate server, OpenAPI generated freshness, web type usage, i18n governance, and dashboard renderer behavior.
- [x] Update recovery trace and todos.
- [x] Commit the validated Phase 3 slice through `$graft-commit`.

## Phase 4 - Archive Readiness

- [x] Confirm all acceptance conditions are met.
- [x] Confirm no dashboard persistence or user preference scope leaked into MVP.
- [x] Mark this topic `archive-ready`.
- [x] Commit final recovery material updates if needed.
