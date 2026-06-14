# Container Management Console Redesign Temporary Plan

## Purpose

This temporary checklist preserves the current container management console redesign plan before implementation moves to
`ai-plan/public/container-management-console-redesign`.

Cleanup rule:

- Delete this file after the public topic reaches `archive-ready`.
- Before archive, update `ai-plan/design/容器管理设计.md` so the design authority matches the implemented revision.

## Current Findings

- The page width has already been relaxed; remaining issues are information architecture, operations-console density,
  table columns, action grouping, pagination, and detail/log entry quality.
- Current `web/src/modules/container/pages/list/index.vue` uses `ManagementPageHeader`, `ManagementTableCard`,
  `TableViewToolbar`, and `resolveTableWidthPolicy`.
- Current page does not use `ManagementTablePagination`, column settings, table density, batch operations, or a stable
  `TableActionMenu` action pattern.
- Refresh is duplicated in PageHeader and TableCard toolbar.
- Current default columns still look like generic CRUD: status, name, image, ports, created time, started time, restart
  policy, operation.
- `started_at` and `restart_policy` are low-value default columns because they are often empty.
- Current action column is too wide and flat: detail, logs, start, stop, restart.
- Current list API has only MVP fields: `id`, `names`, `image`, `image_id`, `labels`, `ports`, `restart_policy`,
  `runtime`, `created_at`, `started_at`, `state`, `status`.
- Detail currently adds `command`, `entrypoint`, `working_dir`, `mounts`, `networks`, `runtime_info`,
  `inspect_updated_at`.
- Missing list-level operations-console fields include `shortId`, primary `name`, `health`, `ipAddress`, network
  summary, `uptime`, CPU/memory stats, restart count, Compose labels, and action availability flags.
- Current backend supports list/detail/logs/start/stop/restart and audits start/stop/restart.
- Current permissions are `ops.container.view`, `ops.container.detail`, `ops.container.logs`, `ops.container.start`,
  `ops.container.stop`, `ops.container.restart`.
- Current dangerous action gate is `ops.container.actions.dangerous_enabled`; remove/delete is not implemented.

## Target Layout

- Page type remains `list-form-detail`, shaped as an operations-status list.
- UI name is `容器管理`; runtime display may say Docker, but product and contract naming stays container-oriented.
- PageHeader:
  - Title: `容器管理`.
  - Subtitle: `查看本机容器运行状态、资源占用、端口映射，并执行受控启停操作。`
  - Header actions hide unimplemented create/update/update-image actions.
  - Refresh moves to TableCard toolbar only.
  - Meta/overview uses compact chips for total, running, stopped, exited/error, healthy, unhealthy, runtime, endpoint.
- FilterBar:
  - Query conditions only.
  - Keyword searches name, ID, image, IP, ports.
  - State filter: all/running/stopped/error.
  - Health filter after the backend field exists.
  - Network/Compose project only after low-cost backend fields exist.
- TableCard:
  - Title: `容器列表`.
  - Subtitle: `来自当前启用的容器运行时。`
  - Toolbar: refresh, column settings, density, batch operations.
  - Footer: `ManagementTablePagination`, default pageSize 20, options 10/20/50/100.

## Default Columns

- selection: fixed left, width 48.
- status: localized status tag, width 96.
- container: name + shortId, full ID tooltip/copy, width 240-280.
- image: repository:tag + runtime, ellipsis tooltip, width 280-340.
- ports: at most two ports plus `+N` tooltip, width 160-220.
- network/ip: primary IP with networks tooltip, width 140-180.
- resource: CPU and memory combined; show `N/A` when stats unavailable, width 180-220.
- runtime status: uptime/status + health, width 180-240.
- createdAt: locale-aware timestamp, width 160-180.
- actions: fixed right, detail/logs/more, width 140-160.

Optional columns:

- full ID
- startedAt
- restartPolicy
- restartCount
- networks
- Compose project
- Compose service
- labels
- runtime endpoint
- image ID
- command
- mount count

## Width And Scroll Policy

- Reuse `resolveTableWidthPolicy`; do not invent another table width system.
- Default visible columns should use fill mode on 1920-wide pages.
- Optional columns may switch to internal table horizontal scroll.
- Body/page-level horizontal scroll is forbidden.
- Selection is fixed left; actions are fixed right.
- Long text must use ellipsis + tooltip.

## Row Actions

- Inline actions: detail, logs, more.
- More menu: start, stop, restart, copy ID.
- Phase 2 may add Inspect under detail/raw JSON.
- Phase 4 may add remove when the backend, permission, dangerous action gate, and audit are implemented.
- Running containers: stop/restart available, start hidden or disabled.
- Exited containers: start available, stop hidden or disabled, restart follows backend `canRestart`.
- Unknown/error containers: detail/logs/copy ID/refresh available, write actions disabled.
- Dangerous operations require confirmation and backend enforcement.

## Drawer Plan

- Detail Drawer Phase 2 target width: 800 or 960.
- Detail sections: basic info, state info, network/ports, resources, Compose/labels, raw JSON collapsed by default.
- Logs Drawer keeps tail, timestamps, stdout/stderr, refresh, copy; add auto-refresh and clearer error states in Phase 2.
- Logs and raw inspect are never loaded by default in the list.

## Backend And OpenAPI Plan

- Phase 3 list supports pagination, keyword, state, health.
- Phase 3 response returns `summary`, `items`, and runtime metadata.
- Low-cost additions: `shortId`, `name`, primary `ipAddress`, network summary, `uptime`, `restartCount`,
  Compose project/service from labels, and `canStart/canStop/canRestart`.
- Docker inspect/stats backed additions: health and resource usage; stats failures should degrade to `N/A` semantics.
- High-cost fields stay out of the list: raw inspect, logs, env vars, huge labels/raw JSON, long-term stats history.
- Remove/delete belongs to Phase 4, not Phase 1.

## Permissions, Config, Audit, I18n

- Keep current permissions for Phase 1-3.
- Add `ops.container.remove` only when remove is actually implemented.
- Keep system config authority in `server/modules/container/config.go` and `contract/config.go`.
- Menu visibility remains backend bootstrap/menu authority and should respect runtime feature gating.
- Audit start/stop/restart; add remove audit in Phase 4.
- All UI copy must be key-first in module locale files for `zh-CN` and `en-US`.

## Phase Checklist

- [ ] Phase 1: wide-screen list convergence.
- [ ] Phase 2: detail and logs Drawer improvements.
- [ ] Phase 3: backend/OpenAPI field and pagination enhancement.
- [ ] Phase 4: controlled operations closure including remove, permissions, dangerous gate, audit, and confirmations.
- [ ] Phase 5: experience polish including auto-refresh, stats refresh, column preference persistence, runtime extension space.
- [ ] Final: update `ai-plan/design/容器管理设计.md`.
- [ ] Final: archive `ai-plan/public/container-management-console-redesign`.
- [ ] Final: delete this temporary dolist file.
