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

- Batch 5 is complete.
- Cross-boundary integration and regression confirmed the settled audit MVP closure without widening into unrelated scope:
  - backend plugin registration still exposes canonical `audit.read` permission, `/audit/logs` menu path, and guarded
    `/api/audit/logs` read route
  - web bootstrap recovery still mounts `/audit/logs` through `modules/index.ts + bootstrap-routes.ts + dynamic routes
    + permission store`
  - the audit module continues to consume checked-in generated OpenAPI DTOs and the existing request-envelope adapter
    rather than page-local API types
- Existing bounded tests now act as the regression proof set for the closure points:
  - backend audit plugin read-surface coverage for permission, menu, and guarded route registration
  - frontend bootstrap-route recovery coverage for `/audit/logs`
  - frontend audit page smoke coverage for generated-contract-backed list rendering
- Batch 5 validation passed without requiring owned-scope regression fixes:
  - `cd server && go test ./...`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
  - `git diff --check`
- Current focus moves to Batch 6:
  - run archive-ready closeout for the topic
  - decide whether any final archive docs or governance notes are still required before archiving

## Owned Scope

- Interpretation rule:
  - owned scope records standing responsibility and bounded execution surface
  - bounded scope forbids unrelated expansion, not required authority repair
  - if future audit drift is traced to upstream authority such as plugin contract, OpenAPI source, or shared bootstrap
    semantics, escalate and repair there instead of adding local compatibility by default

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
