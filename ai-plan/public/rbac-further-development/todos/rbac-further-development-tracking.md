# RBAC Further Development Tracking

## Topic

- Topic: `rbac-further-development`
- Status: `active recovery entry`
- Goal: keep a dedicated recovery入口 for the next RBAC feature worktree/topic pair without reopening shared-baseline governance history.
- Recovery source: parent topic archived -> new standalone topic
- Worktree: none yet
- Branch: none yet

## Scope

- Owned scope:
  - `server/plugins/rbac/**`
  - `web/src/modules/rbac/**`
- This topic is currently recovery-only. No dedicated long-lived worktree has been created yet.
- Until a dedicated worktree/topic pair exists, this topic records entry conditions and ownership truth only; it is not
  a standing permission to edit shared hotspots from the repository root.

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

## Current Recovery Point

- `multi-worktree-governance` no longer needs to carry RBAC-specific follow-up as shared-baseline history.
- The repository root has already returned to `main`; this RBAC topic should not treat the root worktree as its
  dedicated implementation workspace.
- A future RBAC implementation round should start by creating an explicit dedicated worktree/topic pair, then updating
  the public recovery mapping in the owning slice instead of extending this placeholder entry indefinitely.

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

## Admission Conditions For A Dedicated Worktree/Topic Pair

- Create the RBAC worktree only when the next slice is primarily owned by `server/plugins/rbac/**` and/or
  `web/src/modules/rbac/**`, rather than by shared-baseline governance.
- Record the new worktree identity, branch name, owned scope, and any temporary hotspot exceptions in the same slice
  that activates the dedicated pair.
- Keep shared hotspot edits serialized and minimal; if the next slice is mostly about shared router/layout/contract or
  registry coordination, it should stay a bounded integration slice instead of becoming standing RBAC ownership.

## Immediate Next Step

- Use this topic only as the recovery entry for the upcoming RBAC dedicated worktree/topic pair.
- Do not assume `main` is the RBAC worktree. Create the dedicated pair first, then continue feature implementation.
