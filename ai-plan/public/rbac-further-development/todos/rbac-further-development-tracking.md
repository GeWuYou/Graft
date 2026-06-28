# RBAC Further Development Tracking

## Topic

- Topic: `rbac-further-development`
- Status: `active recovery entry`
- Goal: keep the dedicated recovery入口 aligned with the active RBAC feature worktree/topic pair without reopening shared-baseline governance history.
- Recovery source: parent topic archived -> new standalone topic
- Worktree: `/mnt/f/gewuyou/Project/Go/Graft-WorkTree/Graft-wt-rbac-further-development`
- Branch: `feat/wt-rbac-further-development`

## Scope

- Owned scope:
  - `server/modules/rbac/**`
  - `web/src/modules/rbac/**`
- This topic now has a dedicated long-lived worktree/topic pair.
- The topic remains explicit about standing ownership and recovery truth; it is not a standing permission to edit
  shared hotspots outside serialized bounded slices.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- `ai-plan/design/architecture/项目设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/roadmap/MVP实施计划.md`

## Current Recovery Point

- `multi-worktree-governance` no longer needs to carry RBAC-specific follow-up as shared-baseline history.
- The dedicated RBAC implementation workspace is now
  `/mnt/f/gewuyou/Project/Go/Graft-WorkTree/Graft-wt-rbac-further-development` on
  `feat/wt-rbac-further-development`.
- The repository root remains a shared-governance coordination point on `main`; RBAC recovery should enter through the
  dedicated worktree/topic mapping instead of the old branch-only fallback.
- The current shared-hotspot exception is limited to serialized recovery-doc reconciliation under `ai-plan/public` so
  that the public mapping matches the real worktree/branch pair.

## Shared Hotspots

- The following paths may be touched only through explicit bounded coordination slices; they are not long-term standing
  ownership for this topic:
  - `ai-plan/public/README.md`
  - `server/internal/pluginregistry/generated.go`
  - `server/internal/pluginapi/**`
  - `server/internal/contract/**`
  - `web/src/router/**`
  - `web/src/layouts/**`
  - `web/src/locales/**`

## Dedicated Pair Guardrails

- Keep the RBAC worktree primarily owned by `server/modules/rbac/**` and/or `web/src/modules/rbac/**`, rather than by
  shared-baseline governance.
- Keep the worktree identity, branch name, owned scope, and any temporary hotspot exceptions updated in the same slice
  that changes the active-topic mapping or ownership truth.
- Keep shared hotspot edits serialized and minimal; if the next slice is mostly about shared router/layout/contract or
  registry coordination, it should stay a bounded integration slice instead of becoming standing RBAC ownership.

## Immediate Next Step

- Use this topic as the recovery entry for the active RBAC dedicated worktree/topic pair.
- Do not fall back to `main` for RBAC feature recovery while the dedicated pair above remains active.
