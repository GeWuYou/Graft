# RBAC Feature And N+1 Hardening Trace

## 2026-05-26 user-list role summary N+1 removed

- Extended `GET /api/users` list items with minimal embedded `roles` summaries instead of keeping role summaries as a row-level follow-up read.
- Added a stable RBAC cross-plugin capability for batch role summaries by user IDs so `user` continues to depend on a documented capability boundary instead of RBAC repository internals.
- Implemented backend batch role loading with one SQL query over the current user ID set; no per-user role read loop was introduced on the backend.
- Updated the user list page to consume `row.roles` directly for rendering and filtering.
- Kept `GET /api/users/{id}/roles` in place for the single-user role drawer path only.
- Recorded the guardrail that user-list surfaces must not reintroduce row-level `getUserRoleBindings(user.id)` fanout or equivalent `Promise.allSettled(userItems.map(...))` logic.

## 2026-05-26 RBAC feature / OpenAPI / generated / N+1 audit mapping

- Established startup receipt for a `cross-boundary` RBAC audit:
  - governance source: `root AGENTS.md`
  - recovery source:
    - `ai-plan/public/README.md`
    - `ai-plan/public/rbac-further-development/todos/rbac-further-development-tracking.md`
    - `ai-plan/public/rbac-further-development/traces/rbac-further-development-trace.md`
  - branch: `feat/wt-rbac-further-development`
  - worktree status at start: clean
- Read the active RBAC plugin, user-role consumer surface, runtime/plugin registry, OpenAPI paths, generated type
  consumption boundaries, frontend RBAC module, frontend user-role UI, bootstrap route wiring, and permission store.
- Confirmed the currently implemented RBAC management surface:
  - `GET /api/roles`
  - `POST /api/roles`
  - `POST /api/roles/{id}/update`
  - `GET /api/roles/{id}/permissions`
  - `POST /api/roles/{id}/permissions/assign`
  - `GET /api/permissions`
  - `GET /api/users/{id}/roles`
  - `POST /api/users/{id}/roles/assign`
- Confirmed the intentional current governance stance that permission management remains read-only because permission
  metadata is still canonical in plugin registration / permission registry rather than admin-side CRUD.
- Confirmed the main practical N+1 risk is in `web/src/modules/user/pages/index.vue`, where the user list role summary
  performs one `getUserRoleBindings(user.id)` call per visible user row via `Promise.allSettled(...)`.
- Confirmed the main backend RBAC repository reads are mostly single-query reads with SQL-level aggregate subqueries,
  not obvious row-by-row repository N+1 on the read path.
- Recorded the recommended follow-up order:
  1. eliminate user-list role-summary N+1
  2. add canonical role detail read contract
  3. align user-role write naming with replace semantics and decide whether delta endpoints are needed
  4. evaluate role delete / status lifecycle semantics
  5. defer batch user-role operations until the above stabilize
