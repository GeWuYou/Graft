# Container Management Console Redesign Tracking

## Topic

Container Management Console Redesign

## Scope

Optimize the existing container management page from an MVP CRUD-style list into a practical operations-console list
experience, then close backend/OpenAPI fields, controlled operations, validation, and governance.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/分页列表页统一规范与收敛计划.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/系统配置模型与渲染设计.md`
- `ai-plan/design/TDesign-MCP-辅助开发规范.md`
- `web/docs/frontend-i18n-guidelines.md`
- `web/docs/frontend-log-page-guidelines.md`

## Current Recovery Point

- Phase 1 wide-screen list convergence is implemented and validated on the frontend.
- Phase 2 detail and logs Drawer improvements are implemented and validated on the frontend.
- Container list now keeps refresh in the TableCard toolbar, uses local pagination, column settings, density toggle, and
  a stable detail/logs/more action pattern.
- Default visible columns are status, container, image, ports, runtime/status, created time, and operation; `started_at`
  and `restart_policy` are optional column settings.
- Detail Drawer now uses a 960px attached/destroyed Drawer with grouped identity/state/runtime/network/ports/mounts and
  labels/metadata sections using current detail API data only.
- Raw JSON is collapsed by default and serializes only the current detail response; no separate raw inspect request is
  preloaded.
- Logs Drawer now uses an 800px attached/destroyed Drawer, keeps logs unloaded until opened, preserves tail/since,
  timestamp/stdout/stderr/copy controls, and adds optional auto-refresh through the existing logs endpoint.
- Phase 3 backend/OpenAPI field and server-pagination authority is implemented and validated.
- The container list endpoint now accepts `limit` / `offset` plus `keyword` / `state` / `health` filters.
- The list response now returns `items`, `total`, `limit`, `offset`, `summary`, `runtime`, and low-cost row fields for
  identity, health availability, network summary, resource availability, Compose metadata, and action availability.
- Docker list rows intentionally do not preload raw inspect/log/env/stats. List health and resource stats degrade to
  explicit unavailable semantics unless a low-cost runtime list source exists.
- The web container page now consumes generated OpenAPI query/response types and server pagination.
- Phase 4 controlled operation closure is implemented and validated.
- Start / stop / restart now keep backend action availability, dangerous-action gating, audit failure metadata,
  OpenAPI action-response message fields, frontend disabled action state, confirmations, and i18n aligned.
- Remove/delete and batch operations remain deferred because the current design authority excludes delete from MVP and
  a full permission/gate/audit/OpenAPI/UI/test chain would exceed this bounded batch.
- Next batch: `phase-5-polish-validation-governance-closeout`.

## Task Checklist

- [x] Phase 0: planning and topic persistence.
- [x] Phase 1: wide-screen list convergence.
  - [x] PageHeader overview chips.
  - [x] Move refresh exclusively to TableCard toolbar.
  - [x] Tight FilterBar query layout.
  - [x] Default column redesign.
  - [x] Move `started_at` and `restart_policy` to optional columns.
  - [x] Add column settings.
  - [x] Add density toggle.
  - [x] Add `ManagementTablePagination`.
  - [x] Convert row actions to detail/logs/more.
  - [x] Preserve fill/internal-scroll policy and prevent page-level horizontal scroll.
- [x] Phase 2: detail and logs Drawer improvements.
  - [x] Detail Drawer width and sections.
  - [x] Raw JSON collapsed area.
  - [x] Copy ID.
  - [x] Logs Drawer auto-refresh, copy, timestamps, and error states.
- [x] Phase 3: backend/OpenAPI field and pagination enhancement.
  - [x] List pagination and filters.
  - [x] List summary.
  - [x] Health field.
  - [x] Stats/resource fields with graceful unavailability.
  - [x] Ports/network/Compose summaries.
  - [x] Nullable rules and generated artifacts.
- [x] Phase 4: controlled operations closure.
  - [x] Operation availability flags.
  - [x] Optional remove endpoint and permission deferred; design authority still excludes delete from MVP.
  - [x] Dangerous action gate.
  - [x] Audit for all write operations.
  - [x] Confirm dialogs; batch operations deferred with remove/delete.
  - [x] Error codes and i18n keys.
- [ ] Phase 5: experience polish and governance closeout.
  - [ ] Auto-refresh polish.
  - [ ] Stats refresh behavior.
  - [ ] Column preference persistence.
  - [ ] Runtime extension copy and field names.
  - [ ] Final validation.
  - [ ] Browser evidence under `.ai/artifacts/browser/container-page-width-check` or equivalent.
  - [ ] Update `ai-plan/design/容器管理设计.md`.
  - [ ] Archive this topic.
  - [ ] Delete `ai-plan/dolist/container-management-console-redesign-plan.md`.

## Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-planning-topic-persistence",
    "phase-1-wide-screen-list-convergence",
    "phase-2-detail-logs-drawers",
    "phase-3-backend-openapi-fields-pagination",
    "phase-4-controlled-operations-closure"
  ],
  "pending_batches": [
    "phase-5-polish-validation-governance-closeout"
  ],
  "current_batch": null,
  "next_batch": "phase-5-polish-validation-governance-closeout",
  "closeout_status": "active"
}
```
