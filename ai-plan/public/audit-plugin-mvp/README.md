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

- Batch 2 is complete.
- The audit plugin now exposes a guarded read/query API at `/api/audit/logs` with plugin-owned permission, menu, and
  OpenAPI contract closure.
- Backend generated contract artifacts were refreshed from the canonical root OpenAPI workflow and validated against the
  existing HTTP envelope, request-id, and authz chain.
- Current focus moves to Batch 3:
  - integrate richer audit recording into user and RBAC write paths
  - reuse the settled audit read contract instead of widening the API surface
  - keep request-level auto audit as fallback while adding explicit domain events at write success points

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
