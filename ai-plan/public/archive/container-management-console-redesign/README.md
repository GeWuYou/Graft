# Container Management Console Redesign

## Current Status Summary

- Topic goal: reshape the existing container management page into a usable operations-console list experience.
- Status: `archive-ready`.
- Task class: `cross-boundary`.
- Canonical design authority: `ai-plan/design/容器管理设计.md`.
- Temporary implementation checklist: deleted at final closeout after design authority was updated.
- This topic starts from the already widened page layout; do not treat page width as the primary solution.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary: `ai-plan/design/容器管理设计.md` + OpenAPI source + `server/modules/container/**` + `web/src/modules/container/**` + shared management table components

## Owned Scope

Allowed scopes:

- `ai-plan/design/容器管理设计.md`
- `ai-plan/dolist/container-management-console-redesign-plan.md` (deleted during final closeout)
- `ai-plan/public/archive/container-management-console-redesign/**`
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

- Phase 1 wide-screen list convergence is implemented and validated on the frontend.
- Phase 2 detail and logs Drawer improvements are implemented and validated on the frontend.
- Detail Drawer now uses a wider operations-console layout, copy-ID context action, grouped existing detail fields,
  labels/metadata, and collapsed raw detail JSON based only on the current detail response.
- Logs Drawer now keeps logs unloaded until opened and provides clearer controls, copy, timestamp/stdout/stderr toggles,
  optional safe auto-refresh through the existing logs endpoint, and clearer empty/error/loading states.
- Phase 3 backend/OpenAPI field and server-pagination authority is implemented and validated.
- Container list now uses OpenAPI-owned `limit` / `offset` pagination and `keyword` / `state` / `health` filters.
- List responses expose `items`, `total`, `limit`, `offset`, `summary`, `runtime`, and low-cost list row fields.
- Docker list rows avoid raw inspect/log/env/stats preloading; resource stats and health degrade to explicit unavailable semantics on the list path.
- Phase 4 controlled operation closure is implemented and validated.
- Start / stop / restart action availability, dangerous-action gate, audit metadata, confirmation copy, and action
  response i18n are aligned across backend, OpenAPI consumer, and web UI.
- Remove/delete and batch operations remain deferred because deletion is outside the current MVP design authority and
  was not safe to introduce end-to-end in this bounded batch.
- Phase 5 final governance closeout is complete.
- Browser evidence for the 1920-wide no page/body horizontal-scroll check was captured under
  `.ai/artifacts/browser/container-page-width-check/`.
- This topic now lives under `ai-plan/public/archive/container-management-console-redesign/`.

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
- Final closeout updated `ai-plan/design/容器管理设计.md`, archived this topic, and deleted the temporary dolist file.

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
- Phase 3 focused checks completed: OpenAPI bundle, Go OpenAPI generate, frontend OpenAPI type generation/check, focused
  container Go tests, focused container frontend tests, and frontend typecheck.
- Playwright/browser evidence for 1920-wide no page-level horizontal scroll and stable table layout
