# RBAC Further Development Tracking

## Topic

- Topic: `rbac-further-development`
- Status: `active recovery entry`
- Goal: continue RBAC work with durable topic recovery while using reusable temporary Agent worktrees.
- Recovery source: parent topic archived -> new standalone topic

## Scope

- Owned scope:
  - `server/modules/rbac/**`
  - `web/src/modules/rbac/**`
- A worktree directory and task branch are not standing ownership. Each task declares its bounded scope and returns the
  directory to the reusable pool after developer-confirmed integration.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/architecture/前端架构设计.md`

## Current Recovery Point

- Recover through this topic, not through a fixed local directory or branch.
- Before implementation, use Worktree Manager to acquire a clean unique task branch from `main`.
- Shared registry, contract, router, layout, locale, and migration work remains an explicit bounded integration slice.

## Shared Hotspots

- `ai-plan/public/README.md`
- `server/internal/moduleregistry/generated.go`
- `server/internal/moduleapi/**`
- `server/internal/contract/**`
- `web/src/router/**`
- `web/src/layouts/**`
- `web/src/locales/**`

## Immediate Next Step

- Start from root startup preflight, recover this topic, then acquire a reusable worktree for the bounded RBAC task.
