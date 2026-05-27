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

## Batch Plan

1. Batch 1: baseline audit and visibility chain map.
2. Batch 2: canonical bootstrap menu and route alignment.
3. Batch 3: critical element permission coverage.
4. Batch 4: backend permission-guard consistency audit.
5. Batch 5: capability snapshot observability design.

## Immediate Next Step

- Run Batch 1 as a read-only delegated round under `graft-multi-agent-loop`.
- Record the current visibility chain map and concrete drift points before any behavior change.
