# Audit Plugin MVP Trace

## 2026-05-27 Batch 0 started

- Received a bounded `graft-multi-agent-loop` startup prompt for topic `audit-plugin-mvp`.
- Confirmed the task must not start runtime implementation before exploration of plugin, migration, OpenAPI, frontend
  bootstrap, and RBAC guard patterns.

## 2026-05-27 first audit worktree attempt corrected

- Verified the source RBAC worktree was clean before any topic split work.
- The first audit worktree attempt was incorrectly created from `main`, which produced an older code baseline missing
  the expected OpenAPI/generated contract chain.
- Stopped using that incorrect worktree, removed it, and recreated `feat/wt-audit-plugin-mvp` from the clean RBAC
  worktree baseline commit `4cd907e`.
- Re-ran startup preflight inside the corrected audit worktree before continuing.

## 2026-05-27 startup receipt re-established

- Governance source: `root AGENTS.md`
- Task class: `cross-boundary`
- Recovery source:
  - `ai-plan/public/README.md`
  - archived `backend-rbac-contract-audit`
  - current plugin registry implementation
  - current user plugin implementation
  - current rbac plugin implementation
  - current OpenAPI/generated contract workflow
  - current web module/bootstrap/route implementation
- Loop mode: `topic-completion-loop`

## 2026-05-27 exploration findings

- The repository already contains a minimal `server/plugins/audit` plugin.
- The current audit plugin registers request-level middleware and subscribes to `pluginapi.AuditRecordEventName`.
- The current audit DTO and repository are write-only; no query contract or web module exists yet.
- Plugin descriptors declare plugin-owned migration paths, and `pluginregistry` states default migration selection
  expands plugin-owned directories rather than the historical shared migration directory.
- `graft migrate up` synthesizes the default Atlas chain from ordered plugin-owned migration directories.
- `server/internal/httpx` is the canonical envelope and authz boundary:
  - request ids come from `EnsureRequestID`
  - localized error `messageKey` is stored in request context
  - `RequirePermission` injects `pluginapi.RequestAuthContext`
- Canonical OpenAPI and generated contract workflow exists in this corrected baseline:
  - source spec in `openapi/openapi.yaml` plus `openapi/paths/**`
  - backend generated types under `server/internal/contract/openapi/**`
  - web generated schema at `web/src/contracts/openapi/generated/schema.ts`
- Frontend module routing remains bootstrap-driven:
  - `web/src/modules/index.ts` auto-discovers module registrations
  - `permission` store converts bootstrap menus to dynamic routes
  - global route guards restore bootstrap and mount routes at runtime
  - UI visibility uses `v-permission` plus page-local computed guards
- The best future richer-audit insertion points are business success paths in:
  - `server/plugins/user/plugin.go`
  - `server/plugins/rbac/write_service.go`

## Recovery Notes

- Batch 0 remains docs-and-exploration only until the docs slice is validated and committed.
- Shared hotspots remain serialized exceptions; no standing ownership is assumed outside the declared topic scope.
