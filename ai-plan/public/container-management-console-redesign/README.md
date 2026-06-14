# Container Management Console Redesign

## Current Status Summary

- Topic goal: reshape the existing container management page into a usable operations-console list experience.
- Status: `active`.
- Task class: `cross-boundary`.
- Canonical design authority: `ai-plan/design/容器管理设计.md`.
- Temporary implementation checklist: `ai-plan/dolist/container-management-console-redesign-plan.md`.
- This topic starts from the already widened page layout; do not treat page width as the primary solution.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary: `ai-plan/design/容器管理设计.md` + OpenAPI source + `server/modules/container/**` + `web/src/modules/container/**` + shared management table components

## Owned Scope

Allowed scopes:

- `ai-plan/design/容器管理设计.md`
- `ai-plan/dolist/container-management-console-redesign-plan.md`
- `ai-plan/public/container-management-console-redesign/**`
- `ai-plan/public/README.md`
- `openapi/**`
- `server/modules/container/**`
- generated OpenAPI artifacts required by repository workflow
- `web/src/modules/container/**`
- shared management table components only when a container change exposes a real reusable gap
- menu, permission, audit, system-config, and i18n assets directly required by the container topic
- `.ai/artifacts/browser/container-page-width-check/**` or equivalent browser evidence directory

Do not expand into:

- container creation
- image update/pull/build/push
- exec terminal
- Kubernetes
- unrelated monitor/log/audit feature work
- page-width-only layout tweaks
- generic table framework replacement

## Phase Plan

- Phase 1: wide-screen list convergence.
- Phase 2: detail and logs Drawer improvements.
- Phase 3: backend/OpenAPI field and pagination enhancement.
- Phase 4: controlled operation closure.
- Phase 5: experience polish and final governance closeout.

## Current Recovery Point

- Current implementation is MVP-grade and already has list/detail/logs/start/stop/restart.
- Current frontend page uses `ManagementPageHeader`, `ManagementTableCard`, `TableViewToolbar`, and
  `resolveTableWidthPolicy`.
- Current page lacks unified pagination, column settings, density, batch actions, service-side pagination/filtering, and
  operations-console default columns.
- Next batch: `phase-1-wide-screen-list-convergence`.

## Acceptance Conditions

- UI consistently names the capability `容器管理`; runtime details may show Docker.
- Default table columns prioritize status, container, image, ports, IP/network, resources, uptime/health, created time,
  and stable row actions.
- `started_at` and `restart_policy` are optional columns, not default columns.
- Refresh exists in one place only: TableCard toolbar.
- `ManagementTablePagination`, column settings, density control, and internal table scroll policy are applied.
- No body/page-level horizontal scroll on a 1920-wide viewport.
- Detail/log entry points remain usable and do not preload logs or raw inspect data.
- Backend/OpenAPI changes stay authority-first and do not rely on frontend compatibility patches.
- Permissions, dangerous action gates, audit, system config, and i18n stay aligned with repository governance.
- Final closeout updates `ai-plan/design/容器管理设计.md`, archives this topic, and deletes the temporary dolist file.

## Validation Targets

```bash
cd web && bun run check
cd server && go run ./cmd/graft validate backend
git diff --check
```

Additional expected focused checks:

- container frontend Vitest for presenter, columns, actions, pagination, Drawer entry points
- container backend focused Go tests for list filters/pagination, disabled/unavailable runtime, dangerous action gates,
  audit, and logs query behavior
- OpenAPI validation/generation checks required by the changed contract slice
- Playwright/browser evidence for 1920-wide no page-level horizontal scroll and stable table layout
