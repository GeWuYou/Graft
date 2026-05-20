# RBAC Further Development Trace

## 2026-05-20 standalone RBAC recovery entry created

- Split RBAC follow-up out of `multi-worktree-governance` because the parent topic should stay focused on shared
  baseline governance, hotspot policy, and worktree mapping truth rather than carrying plugin-specific continuation
  notes.
- Opened `rbac-further-development` as a standalone active recovery entry so the next RBAC slice can recover quickly
  without reviving archived or over-broad governance history.
- Recorded the current neutral startup state explicitly:
  - `Worktree: none yet`
  - `Branch: none yet`
  - the topic is recovery-only and no dedicated long-lived worktree exists yet
- Recorded that the repository root has already returned to `main`, which means the root is not the standing RBAC
  implementation workspace for this topic.

## Dedicated Pair Admission Rule

- Promote this topic from recovery-only to a real implementation topic only after all of the following are true:
  - a dedicated worktree is created for the RBAC slice
  - that worktree has an explicit branch name
  - the owned scope remains centered on `server/plugins/rbac/**` and/or `web/src/modules/rbac/**`
  - any required shared-hotspot touches are declared as bounded exceptions rather than assumed standing ownership
- If the next slice is still dominated by `ai-plan/public/README.md`, plugin registry wiring, shared contracts, router,
  layouts, or locales coordination, keep it as an integration/governance slice instead of treating it as the RBAC
  worktree baseline.

## Recovery Notes

- Shared hotspots for future RBAC slices stay opt-in only:
  - `ai-plan/public/README.md`
  - `server/internal/pluginregistry/generated.go`
  - `server/internal/pluginapi/**`
  - `server/internal/contract/**`
  - `web/src/router/**`
  - `web/src/layouts/**`
  - `web/src/locales/**`
- The standing owned scope remains:
  - `server/plugins/rbac/**`
  - `web/src/modules/rbac/**`
