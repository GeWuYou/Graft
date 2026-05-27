# Audit Plugin MVP

## Topic

- Topic: `audit-plugin-mvp`
- Status: `active`
- Loop mode: `topic-completion-loop`
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-audit-plugin-mvp`
- Branch: `feat/wt-audit-plugin-mvp`

## Goal

- Build and close the audit plugin MVP topic as a bounded cross-boundary loop.
- Deliver audit log recording for key admin actions, a guarded query API, and a read-only web list page.
- Keep plugin boundaries, Ent/Atlas migration governance, OpenAPI/generated contract flow, and menu/permission/route
  alignment explicit.

## Current Recovery Point

- Batch 3 is complete.
- The audit plugin now exposes a guarded read/query API at `/api/audit/logs` with plugin-owned permission, menu, and
  OpenAPI contract closure.
- Backend write-path integration now emits richer active audit events from bounded `user` and `rbac` success paths while
  keeping audit publish failures non-blocking to the business write flow.
- User and RBAC write routes now propagate request ids into those active audit events without changing the settled read
  contract.
- Current focus moves to Batch 4:
  - build the frontend audit module and page on top of the settled `audit.read` permission and `/audit/logs` menu path
  - consume the existing read contract instead of redefining page-local backend semantics
  - keep Batch 4 inside owned frontend scope without widening backend API surface

## Owned Scope

- Recovery docs:
  - `ai-plan/public/audit-plugin-mvp/**`
  - `ai-plan/public/README.md`
- Server:
  - `server/plugins/audit/**`
  - `server/internal/pluginregistry/**`
  - `server/internal/plugin/**`
  - `server/internal/ent/**`
  - `server/internal/ent/schema/**`
  - `server/internal/ent/migrate/migrations/**`
  - `server/internal/httpx/**`
  - `server/internal/permission/**`
  - `server/internal/menu/**`
  - `openapi/**`
  - `server/cmd/**`
- Web:
  - `web/src/modules/audit/**`
  - `web/src/modules/index.ts`
  - module auto-registration files if directly required
  - `web/src/store/modules/permission.ts`
  - `web/src/utils/route/**`
  - `web/src/app/bootstrap/**`
  - `web/src/contracts/openapi/generated/**` only when produced by the contract workflow

## Shared Hotspots

- Shared hotspots may only be touched through bounded serialized slices:
  - `ai-plan/public/README.md`
  - `server/internal/pluginregistry/generated.go`
  - `server/internal/pluginapi/**`
  - `server/internal/contract/**`
  - `web/src/router/**`
  - `web/src/layouts/**`
  - `web/src/locales/**`

## Batch Plan

- Batch 0: exploration and worktree/topic setup
- Batch 1: backend audit domain design and schema
- Batch 2: backend API, permission, menu, OpenAPI contract
- Batch 3: backend recording integration for user and RBAC actions
- Batch 4: frontend audit module and page
- Batch 5: cross-boundary integration and regression
- Batch 6: archive-ready closeout
