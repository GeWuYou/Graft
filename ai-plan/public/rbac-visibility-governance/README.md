# RBAC Visibility Governance

## Status

- Topic: `rbac-visibility-governance`
- Status: `active`
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-rbac-further-development`
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

## Batch Guardrails

- Prefer one bounded slice per commit.
- Keep validation honest and minimal for each batch.
- Do not widen into generalized RBAC redesign.
- Preserve OpenAPI generated contract as the API typing source of truth.
