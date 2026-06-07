# Dashboard Contribution Todos

## Loop Batch State

- completed_batches: []
- pending_batches:
  - Phase 1: backend dashboard registry, aggregate route, OpenAPI source, and core module-runtime widget
  - Phase 2: web dashboard renderer and home route integration
  - Phase 3: RBAC access summary widget and final cross-boundary validation
  - Phase 4: archive-readiness closeout
- current_batch: setup
- next_batch: Phase 1
- terminal_status: in-progress

## Phase 1 - Backend Registry And Core Widget

- [ ] Add `server/internal/dashboard` registry, definition, loader, permission filtering, error handling, and tests.
- [ ] Inject `DashboardRegistry` into `module.Context` through `server/internal/app/runtime.go`.
- [ ] Register authenticated dashboard summary route in the core authenticated route registration path.
- [ ] Add OpenAPI source for `GET /api/dashboard/summary` and `GET /api/dashboard/widgets/{widget_id}`.
- [ ] Add the `core.module-runtime-health` widget using existing module runtime snapshot data.
- [ ] Regenerate required OpenAPI artifacts.
- [ ] Run focused backend/OpenAPI validation.
- [ ] Commit the validated Phase 1 slice through `$graft-commit`.

## Phase 2 - Web Renderer

- [ ] Add `web/src/modules/dashboard` API, contract paths, types, page, renderer, and widget components.
- [ ] Render fixed system summary and generic widget grid.
- [ ] Support MVP widget types: `stat-group`, `alert-list`, `link-list`, `timeline`, `health`.
- [ ] Integrate the dashboard route as the home page without importing module business components.
- [ ] Add loading, empty, disabled, and per-widget error states.
- [ ] Add dashboard locales and focused frontend tests.
- [ ] Commit the validated Phase 2 slice through `$graft-commit`.

## Phase 3 - RBAC Access Summary And Final Validation

- [ ] Add `rbac.access-summary` stat-group widget with real permissions: `user.read`, `role.read`, `permission.read`.
- [ ] Use module-owned service/repository boundaries; do not let dashboard core import RBAC/user internals.
- [ ] Validate server, OpenAPI generated freshness, web type usage, i18n governance, and dashboard renderer behavior.
- [ ] Update recovery trace and todos.
- [ ] Commit the validated Phase 3 slice through `$graft-commit`.

## Phase 4 - Archive Readiness

- [ ] Confirm all acceptance conditions are met.
- [ ] Confirm no dashboard persistence or user preference scope leaked into MVP.
- [ ] Mark this topic `archive-ready` or record a precise blocker.
- [ ] Commit final recovery material updates if needed.

