# RBAC Visibility Governance

## Status

- Topic: `rbac-visibility-governance`
- Status: `archived`
- Branch: `feat/wt-rbac-further-development`
- Task class: `cross-boundary`

## Goal

Govern the existing RBAC visibility chain end to end:

- `permission`
- `bootstrap menus`
- `dynamic routes`
- `element visibility`
- `API guard`

This topic exists to strengthen the current closure path rather than expand RBAC scope.

## Explicit Non-Goals

- Do not add menu CRUD.
- Do not add resource CRUD.
- Do not add a resource table.
- Do not migrate menu canonical truth from registry/bootstrap into database-owned CRUD.
- Do not derive and persist a new resource model by reverse-parsing permission codes.

## Current Repository Conclusion

- Backend already owns permission metadata through plugin registration and the platform permission registry.
- Backend already filters bootstrap menus by granted permission codes.
- Frontend already builds dynamic routes from bootstrap menus and uses bootstrap permissions for visibility decisions.
- Backend already performs request-time permission-code authorization on protected APIs.
- Resource is not a first-class persisted object in the current RBAC implementation and should remain out of scope for this topic.
- A low-cost observability slice is feasible only as a read-only session snapshot built from existing bootstrap and mounted-route state; a stable "missing permission reason" contract does not exist yet beyond guarded API `403` denial detail.

## Scope

- `ai-plan/public/rbac-visibility-governance/**`
- `ai-plan/public/README.md`
- `server/plugins/rbac/**`
- `server/internal/permission/**`
- `server/internal/menu/**`
- `server/internal/httpx/**`
- `server/plugins/user/bootstrap.go`
- `web/src/store/modules/permission.ts`
- `web/src/utils/route/**`
- `web/src/app/bootstrap/**`
- `web/src/modules/rbac/**`
- `web/src/modules/access-control/**`
- bounded OpenAPI/generated contract files only if required

## Planned Batches

1. Baseline audit and current visibility chain mapping.
2. Bootstrap menu and dynamic route canonical-path alignment.
3. Critical button-level permission visibility coverage.
4. Backend API guard consistency audit and targeted fixes.
5. Capability snapshot observability design, and low-cost implementation only if clearly justified.

## Batch 5 Design Conclusion

- The repository should not add a new backend observability contract in this topic.
- If a future slice wants a capability snapshot, keep it frontend-only and read-only:
  - current user, roles, permissions, and visible menus can come directly from the existing bootstrap snapshot
  - dynamic routes can come from the already mounted runtime route tree in the permission store/router bootstrap path
  - denied permission evidence can only be shown when a guarded API actually returns `403` with the existing denied `permission` detail
- The repository should not promise a generalized "missing permission reason" for hidden menus, hidden routes, or hidden buttons in this topic because that would require a new canonical reason model across backend guards, bootstrap filtering, and frontend-only `v-permission` decisions.
- Therefore Batch 5 closes as design-only with no implementation recommendation for the current owned scope.

## Batch Guardrails

- Prefer one bounded slice per commit.
- Keep validation honest and minimal for each batch.
- Do not widen into generalized RBAC redesign.
- Preserve OpenAPI generated contract as the API typing source of truth.

## Final Archive Record

- Status: `archived`
- Archive reason: the topic completed its full Option A loop and left no blocking gaps inside the owned scope.
- Final result:
  - canonical access-control paths are now limited to `/access-control`, `/access-control/overview`, `/access-control/users`, `/access-control/roles`, and `/access-control/permissions`
  - frontend route/menu compatibility drift for legacy `/users`, `/roles`, `/permissions` and historical `title_key` rewrites was removed
  - owned RBAC/access-control critical actions now use the canonical `v-permission` visibility path
  - backend permission guard coverage across RBAC and bootstrap-adjacent management routes was verified as already consistent
  - capability snapshot observability remains design-only, with no new backend contract added
- Final validation summary:
  - `cd web && bun run check`
  - `cd server && go run ./cmd/graft validate backend`
  - `git diff --check`
- Follow-up status: `superseded`
- Superseded by:
  - operating rule `feature-delivery-with-existing-rbac-visibility-chain`
- Next-session prompt: `No continuation required.`
