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
