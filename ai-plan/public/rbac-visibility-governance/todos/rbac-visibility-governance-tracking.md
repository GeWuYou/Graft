# RBAC Visibility Governance Tracking

## Topic

- Topic: `rbac-visibility-governance`
- Status: `active`
- Goal: strengthen the existing RBAC visibility closure path without introducing menu CRUD or resource CRUD.
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-rbac-further-development`
- Branch: `feat/wt-rbac-further-development`

## Scope

- Owned scope:
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

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/roadmap/MVP实施计划.md`

## Governance Guardrails

- No menu CRUD.
- No resource CRUD.
- No resource table.
- No migration of menu truth from registry/bootstrap to database CRUD.
- No hand-written API DTO truth that bypasses OpenAPI generated contract.

## Current Recovery Point

- Topic initialized on the dedicated RBAC worktree and branch pair.
- The current implementation direction is Option A only:
  - govern `permission -> bootstrap menus -> dynamic routes -> element visibility -> API guard`
  - avoid menu and resource management expansion
- Batch 1 read-only baseline audit completed.
- The current closure path is present end to end, but the main drift still sits on the `web` side:
  - compatibility-heavy access-control bootstrap normalization
  - frontend-owned menu hierarchy synthesis
  - legacy `title_key` rewriting
  - inconsistent button-level visibility conventions
  - one frontend permission-name alias for the same backend code
- Batch 2 canonical path alignment completed.
- Canonical access-control bootstrap truth is now:
  - `/access-control`
  - `/access-control/overview`
  - `/access-control/users`
  - `/access-control/roles`
  - `/access-control/permissions`
- Legacy `/users`, `/roles`, `/permissions` compatibility handling has been removed from the frontend bootstrap route adapter.
- Batch 3 critical element permission coverage completed on owned RBAC/access-control surfaces.
- Critical RBAC/access-control actions now use the canonical `v-permission` visibility path.

## Batch Plan

1. Batch 1: baseline audit and visibility chain map. Status: completed.
2. Batch 2: canonical bootstrap menu and route alignment. Status: completed.
3. Batch 3: critical element permission coverage. Status: completed.
4. Batch 4: backend permission-guard consistency audit. Status: completed.
5. Batch 5: capability snapshot observability design. Status: next.

## Immediate Next Step

- Execute Batch 5 under `graft-multi-agent-loop`.
- Design a low-cost capability snapshot observability slice only if it can stay within Option A and avoid RBAC scope expansion.
- Keep the next round read-first on current backend/frontend snapshot surfaces before proposing instrumentation.

## Batch 1 Findings Summary

- Current visibility chain map:
  - permission registry declarations feed backend route guards
  - bootstrap returns granted permissions plus permission-filtered menus
  - `web` permission store persists bootstrap snapshot and builds dynamic routes
  - layouts consume mounted dynamic routes for sidebar/header rendering
  - button-level visibility exists, but is not yet standardized on one mechanism
- Concrete drift points:
  - `web/src/utils/route/bootstrap.ts` still rewrites legacy `/users`, `/roles`, `/permissions` paths into `/access-control/*`
  - `web` still synthesizes access-control root and overview hierarchy instead of consuming one canonical upstream shape
  - `web` still rewrites historical `title_key` values into access-control keys
  - `v-permission` exists, but critical RBAC/user surfaces still depend mostly on per-page computed booleans
  - `web/src/modules/rbac/contract/permissions.ts` still keeps a semantic alias for the same backend permission code

## Batch 2 Decision Record

- Breaking migration allowed and applied.
- Deleted legacy compatibility paths:
  - `/users`
  - `/roles`
  - `/permissions`
- Deleted frontend bootstrap normalize / rewrite / fallback logic for:
  - legacy access-control paths
  - historical access-control `title_key` rewrites
- Preferred backend registry/bootstrap truth over frontend compatibility transforms.
- Backend now emits canonical access-control bootstrap menus in stable order, including explicit `/access-control` root.
- Validation completed for Batch 2:
  - `cd web && bun run check`
  - `cd server && go test ./plugins/rbac ./plugins/user`
  - `git diff --check`

## Batch 3 Decision Record

- Standardized owned RBAC/access-control surfaces onto the existing `v-permission` directive for dangerous or privileged actions.
- Tightened access-control overview behavior so it no longer calls protected read APIs when the current session lacks the corresponding read permission.
- Kept the round inside owned `web` scope; no server code changed.
- Explicit remaining drift after Batch 3:
  - `web/src/modules/user/pages/index.vue` still uses page-local permission booleans and dropdown disable states for some dangerous actions
  - `RBAC_PERMISSION_CODE.ROLE_PERMISSION_MANAGE` remains a frontend alias of `role.permission.assign`
- Validation completed for Batch 3:
  - `cd web && bun run check`
  - `git diff --check`

## Batch 4 Decision Record

- Executed the backend guard-consistency audit under Option A only and stayed inside the owned backend/recovery scope.
- Audited explicit permission registry declarations against registered management routes on:
  - `server/plugins/rbac/**`
  - `server/plugins/user/**`
  - `server/plugins/auth/**`
  - `server/internal/httpx/**`
- Confirmed the current backend permission-code coverage is already explicit and internally aligned:
  - RBAC role read/write routes all use declared `httpx.RequirePermission(...)` guards backed by registered RBAC permission codes.
  - RBAC permission-list and permission-detail routes are guarded by `permission.read`.
  - RBAC user-role snapshot and mutation routes are guarded by `user.role.read` and `user.role.assign`.
  - User management routes are guarded by the expected `user.read` / `user.create` / `user.update` / `user.disable` codes.
  - Admin user-session management routes are guarded by `user.session.read` and `user.session.revoke`.
  - Auth bootstrap and current-user session routes remain auth-owned and use authenticated/restricted-session guards rather than RBAC permission guards, which matches the current lifecycle split.
- Confirmed the suspected restricted-session/logout asymmetry is not a backend bug in this slice:
  - the logout route is registered under `auth` and does not pass through the management-only `restrictedSession` guard attached on `/users/**`
  - the existing README wording is therefore descriptive rather than evidence of an unguarded management route
- No real missing permission registry declaration, missing route guard, or naming drift requiring code changes was found in the audited scope.
- This batch therefore closes as audit-only with no server or web implementation edits.
