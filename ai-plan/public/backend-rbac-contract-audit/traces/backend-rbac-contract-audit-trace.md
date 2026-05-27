# Backend RBAC Contract Audit Trace

## 2026-05-27 Batch 0 initialized topic docs and recorded the initial audit inventory

- Reused the inherited startup context under root `AGENTS.md` for a `cross-boundary` retry worker round in
  `graft-multi-agent-loop`.
- Re-read the current-turn startup minimums:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `.ai/environment/tools.ai.yaml`
  - `ai-plan/public/README.md`
- Reused the required recovery sources from the inherited loop context:
  - archived `rbac-visibility-governance`
  - archived `user-page-permission-governance`
  - archived `frontend-permission-code-cleanup`
  - current RBAC backend implementation
  - current RBAC frontend implementation
- Confirmed the worktree was clean before Batch 0 writes.
- Created the new topic document set:
  - `ai-plan/public/backend-rbac-contract-audit/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/todos/backend-rbac-contract-audit-tracking.md`
  - `ai-plan/public/backend-rbac-contract-audit/traces/backend-rbac-contract-audit-trace.md`
- Updated `ai-plan/public/README.md` to register `backend-rbac-contract-audit` as the active recovery topic for the
  current branch/worktree.
- Audited the owned backend RBAC contract surfaces:
  - `server/plugins/rbac/contract/permission.go`
  - `server/plugins/rbac/contract/route.go`
  - `server/plugins/rbac/plugin_registration.go`
  - `server/plugins/rbac/route_registration.go`
  - `server/internal/permission/registry.go`
  - `server/internal/menu/registry.go`
  - `server/internal/httpx/authz.go`
- Audited the owned frontend RBAC/user visibility surfaces:
  - `web/src/modules/rbac/contract/permissions.ts`
  - `web/src/modules/user/contract/permissions.ts`
  - `web/src/modules/rbac/bootstrap-routes.ts`
  - `web/src/modules/user/bootstrap-routes.ts`
  - `web/src/store/modules/permission.ts`
  - `web/src/utils/route/bootstrap.ts`
  - `web/src/modules/rbac/pages/index.vue`
  - `web/src/modules/rbac/pages/permissions/index.vue`
  - `web/src/modules/user/pages/index.vue`
- Recorded the initial inventory required by Batch 0:
  - backend permission registry inventory
  - backend menu declaration inventory
  - backend RBAC API route inventory
  - backend guard inventory
  - frontend permission constant inventory
  - frontend route/menu visibility inventory
  - frontend page/action permission usage inventory
- Drafted the initial RBAC contract audit matrix inside the topic README instead of scattering the first conclusions
  across trace-only notes.
- Recorded two bounded follow-up questions for later batches without widening scope:
  - backend menu `/access-control/overview` currently has no owned frontend page registration in the current read scope
  - frontend route `/access-control/users` is owned in current frontend scope, but its backend menu declaration owner
    is outside the current Batch 0 RBAC backend read scope
- Kept Batch 0 docs-only; no runtime or test code was changed.

## 2026-05-27 Batch 1 completed the backend permission, menu, API, and guard audit

- Reused the inherited startup context under root `AGENTS.md` for the `cross-boundary` Batch 1 worker round.
- Re-read the required startup and recovery sources for the current round:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `.ai/environment/tools.ai.yaml`
  - `ai-plan/public/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/todos/backend-rbac-contract-audit-tracking.md`
  - `ai-plan/public/backend-rbac-contract-audit/traces/backend-rbac-contract-audit-trace.md`
  - `ai-plan/design/契约治理与魔法值治理规范.md`
  - `ai-plan/design/AI任务追踪与恢复设计.md`
- Confirmed the worktree was clean before Batch 1 writes.
- Audited the owned backend RBAC contract and guard surfaces in detail:
  - `server/plugins/rbac/contract/permission.go`
  - `server/plugins/rbac/contract/route.go`
  - `server/plugins/rbac/plugin_registration.go`
  - `server/plugins/rbac/route_registration.go`
  - `server/plugins/rbac/route_read_handlers.go`
  - `server/plugins/rbac/route_write_handlers.go`
  - `server/plugins/rbac/route_errors.go`
  - `server/plugins/rbac/read_service.go`
  - `server/plugins/rbac/write_service.go`
  - `server/plugins/rbac/store/rbac.go`
  - `server/internal/permission/registry.go`
  - `server/internal/menu/registry.go`
  - `server/internal/httpx/authz.go`
- Audited existing tests as evidence instead of widening scope into speculative fixes:
  - `server/plugins/rbac/plugin_test.go`
  - `server/internal/httpx/authz_test.go`
- Confirmed these backend closure properties from code plus tests:
  - all owned RBAC write routes have explicit non-blank guards
  - all owned RBAC read routes have explicit expected guards
  - menu permission references used by RBAC-owned menus exist in the RBAC permission registry
  - denied requests return standard `403 auth.forbidden` semantics with `details.permission`
  - resource misses return dedicated `404` contracts
  - malformed request input and stale referenced IDs normalize to `400 common.invalid_argument`
  - builtin role rename and actor self-lockout protection remain enforced below the page layer
- Recorded one bounded governance note rather than treating it as a Batch 1 code fix:
  - `server/internal/permission.Registry` and `server/internal/menu.Registry` currently append declarations without
    duplicate or cross-reference validation
  - current safety for uniqueness and closure comes from canonical contract ownership plus tests, not registry runtime
    hard enforcement
- Did not find a clear low-risk backend runtime gap in owned scope, so no runtime or test code was changed.
- Updated the topic README and tracking docs to carry the Batch 1 backend audit matrix and conclusions forward to
  Batch 2.

## 2026-05-27 Batch 2 completed the frontend permission, route, and action audit

- Reused the inherited startup context under root `AGENTS.md` for the `cross-boundary` Batch 2 worker round.
- Re-read the required startup and recovery sources for the current round:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `.ai/environment/tools.ai.yaml`
  - `ai-plan/public/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/todos/backend-rbac-contract-audit-tracking.md`
  - `ai-plan/public/backend-rbac-contract-audit/traces/backend-rbac-contract-audit-trace.md`
  - `ai-plan/design/契约治理与魔法值治理规范.md`
  - `ai-plan/design/AI任务追踪与恢复设计.md`
  - `ai-plan/design/前端架构设计.md`
  - `ai-plan/design/前端视觉设计规范.md`
  - `ai-plan/design/TDesign-MCP-辅助开发规范.md`
- Confirmed the worktree was clean before Batch 2 writes.
- Audited the owned frontend permission and route surfaces in detail:
  - `web/src/modules/rbac/contract/permissions.ts`
  - `web/src/modules/user/contract/permissions.ts`
  - `web/src/store/modules/permission.ts`
  - `web/src/utils/route/bootstrap.ts`
  - `web/src/utils/route/bootstrap.test.ts`
  - `web/src/modules/rbac/bootstrap-routes.ts`
  - `web/src/modules/rbac/contract/bootstrap.ts`
  - `web/src/modules/user/bootstrap-routes.ts`
  - `web/src/modules/user/contract/paths.ts`
  - `web/src/modules/rbac/pages/index.vue`
  - `web/src/modules/rbac/pages/permissions/index.vue`
  - `web/src/modules/user/pages/index.vue`
- Read adjacent consistency context without widening write scope:
  - `web/src/modules/access-control/contract/bootstrap.ts`
  - `web/src/modules/access-control/bootstrap-routes.ts`
  - `web/src/modules/access-control/index.ts`
  - `server/plugins/rbac/contract/permission.go`
  - `server/plugins/user/contract/permission.go`
  - `server/plugins/user/plugin_registration.go`
  - `server/plugins/user/bootstrap.go`
- Confirmed these frontend closure properties from code plus tests:
  - all owned frontend permission constants map to canonical backend-owned wire-format codes
  - `/access-control/users`, `/access-control/roles`, and `/access-control/permissions` all have aligned backend menu
    owners and frontend bootstrap route registrations
  - the previously open `/access-control/overview` question is closed by the `access-control` module registration
  - role-permission and user-role action entrypoints require the same multi-permission combinations as the backend
    read/write guard split
  - user page privileged handlers keep runtime permission re-checks before opening dialogs or mutating state
- Ran TDesign MCP preflight because the owned frontend fix touches a `Dropdown` usage surface:
  - framework: `vue-next`
  - components: `Dropdown`
  - docs checked: `get_component_docs`
  - fallback: none
- Found one clear low-risk owned-scope frontend drift:
  - `web/src/modules/rbac/pages/index.vue` rendered RBAC row `More` dropdown items as disabled when the viewer lacked
    `role.status.update` or `role.delete`
  - this violated the already-governed frontend rule that permission-missing actions should be hidden rather than shown
    as permission-only disabled affordances
- Applied the smallest fix in owned scope:
  - changed `roleRowMoreOptions(...)` to emit only permission-authorized actions
  - kept `builtin` lifecycle-driven disabled behavior intact for authorized users
  - hid the `More` dropdown button entirely when no authorized row action remains
  - added a focused page test proving permission-missing `More` actions no longer render
- Updated topic docs and recovery metadata to carry the Batch 2 audit matrix, findings, and next-step pointer into
  Batch 3.

## 2026-05-27 Batch 3 completed the cross-boundary consistency audit

- Reused the inherited startup context under root `AGENTS.md` for the `cross-boundary` Batch 3 worker round.
- Re-read the required startup and recovery sources for the current round:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `.ai/environment/tools.ai.yaml`
  - `ai-plan/public/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/todos/backend-rbac-contract-audit-tracking.md`
  - `ai-plan/public/backend-rbac-contract-audit/traces/backend-rbac-contract-audit-trace.md`
  - `ai-plan/design/契约治理与魔法值治理规范.md`
  - `ai-plan/design/AI任务追踪与恢复设计.md`
- Confirmed the worktree was clean before Batch 3 writes.
- Re-verified the backend-to-frontend closure surfaces instead of assuming Batch 1 and Batch 2 conclusions:
  - `server/plugins/rbac/contract/permission.go`
  - `server/plugins/rbac/plugin_registration.go`
  - `server/plugins/rbac/route_registration.go`
  - `server/internal/httpx/authz.go`
  - `server/internal/httpx/authz_test.go`
  - `server/plugins/user/plugin_registration.go`
  - `server/plugins/user/bootstrap.go`
  - `web/src/modules/rbac/contract/permissions.ts`
  - `web/src/modules/user/contract/permissions.ts`
  - `web/src/modules/rbac/bootstrap-routes.ts`
  - `web/src/modules/user/bootstrap-routes.ts`
  - `web/src/modules/access-control/bootstrap-routes.ts`
  - `web/src/store/modules/permission.ts`
  - `web/src/utils/route/bootstrap.ts`
  - `web/src/utils/route/bootstrap.test.ts`
  - `web/src/modules/rbac/pages/index.vue`
  - `web/src/modules/rbac/pages/index.test.ts`
  - `web/src/modules/user/pages/index.vue`
  - `web/src/modules/user/pages/index.test.ts`
- Merged Batch 1 backend evidence and Batch 2 frontend evidence into one Batch 3 cross-boundary matrix covering:
  - role CRUD surfaces
  - permission list/detail/read-only surfaces
  - role-permission snapshot and mutation surfaces
  - user-role snapshot and single-user or batch mutation surfaces
  - user manage-roles entry semantics
  - bootstrap menu ordering and dynamic route assembly
  - route visibility versus action visibility separation
  - builtin protection and authn/authz error separation
- Confirmed the current owned runtime scope is aligned and did not justify a runtime fix:
  - no backend-only gap
  - no frontend-only gap
  - no blocker within current MVP scope
- Found one bounded documentation drift in owned scope:
  - `server/plugins/rbac/README.md` still described stale `.../roles/assign` wording and replace-only write semantics
- Applied the smallest fix:
  - updated `server/plugins/rbac/README.md` to match current `replace | add | remove` contracts
- Updated:
  - `ai-plan/public/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/todos/backend-rbac-contract-audit-tracking.md`
  - `ai-plan/public/backend-rbac-contract-audit/traces/backend-rbac-contract-audit-trace.md`
- Kept Batch 3 effectively docs-only for validation purposes:
  - relied on prior Batch 1 backend validation and Batch 2 web validation for unchanged runtime code
  - ran `git diff --check` for the current owned docs changes

## 2026-05-27 Batch 4 completed the terminal archive closeout

- Reused the inherited startup context under root `AGENTS.md` for the `cross-boundary`
  `batch-4-mvp-stable-decision-and-archive-closeout` worker round.
- Re-read the required startup and recovery sources for the current round:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `.ai/environment/tools.ai.yaml`
  - `ai-plan/public/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/todos/backend-rbac-contract-audit-tracking.md`
  - `ai-plan/public/backend-rbac-contract-audit/traces/backend-rbac-contract-audit-trace.md`
  - `ai-plan/design/契约治理与魔法值治理规范.md`
  - `ai-plan/design/AI任务追踪与恢复设计.md`
- Confirmed the worktree was clean before Batch 4 archive writes.
- Reused accepted evidence from prior completed batches:
  - Batch 1 backend permission/menu/API/guard audit
  - Batch 2 frontend permission/route/action audit
  - Batch 3 cross-boundary closure matrix
- Chose the terminal MVP decision as `mvp-stable-with-risks` instead of `mvp-stable`:
  - current MVP closure is validated and archive-ready
  - remaining known issues are governance-hardening risks rather than proven runtime mismatches
  - the most concrete residual note is that permission/menu registries still rely on tests plus canonical ownership
    instead of runtime duplicate/reference enforcement
- Updated archive and recovery materials:
  - `ai-plan/public/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/README.md`
  - `ai-plan/public/backend-rbac-contract-audit/todos/backend-rbac-contract-audit-tracking.md`
  - `ai-plan/public/backend-rbac-contract-audit/traces/backend-rbac-contract-audit-trace.md`
- Recorded the follow-up policy explicitly:
  - no proactive RBAC feature expansion remains in this topic line
  - later work is bugfix-only unless a new topic is opened
  - data permission / row-level permission, organization permission, tenant permission, and observability remain
    separate future topics
- Ran the final required validation chain for this terminal closeout:
  - `git status --short`
  - `git branch --show-current`
  - `cd web && bun run check`
  - `cd server && go run ./cmd/graft validate backend`
  - `git diff --check`
- Prepared the terminal archive-ready closeout and the final scoped docs commit for the owned archive materials.
