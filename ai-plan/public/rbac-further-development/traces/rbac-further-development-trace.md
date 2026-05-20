# RBAC Further Development Trace

## 2026-05-20 dedicated worktree mapping reconciled

- Verified the active RBAC implementation workspace already exists at
  `/mnt/f/gewuyou/Project/Go/Graft-WorkTree/Graft-wt-rbac-further-development`.
- Verified the paired branch is already `feat/wt-rbac-further-development`.
- Updated the public recovery index and topic tracking so RBAC recovery no longer points at the stale `main`
  branch-only fallback.
- Recorded the current bounded shared-hotspot exception explicitly:
  - `ai-plan/public/README.md`
  - `ai-plan/public/rbac-further-development/**`

## 2026-05-20 standalone RBAC recovery entry created

- Split RBAC follow-up out of `multi-worktree-governance` because the parent topic should stay focused on shared
  baseline governance, hotspot policy, and worktree mapping truth rather than carrying plugin-specific continuation
  notes.
- Opened `rbac-further-development` as a standalone active recovery entry so the next RBAC slice can recover quickly
  without reviving archived or over-broad governance history.
- Recorded the initial neutral startup state explicitly:
  - `Worktree: none yet`
  - `Branch: none yet`
  - the topic is recovery-only and no dedicated long-lived worktree exists yet
- Recorded that the repository root has already returned to `main`, which means the root is not the standing RBAC
  implementation workspace for this topic.
- This initial placeholder state was superseded later on 2026-05-20 when the dedicated RBAC worktree/branch pair was
  created and the public recovery mapping was reconciled.

## Dedicated Pair Guardrails

- Keep this topic aligned with a real implementation workspace only while all of the following remain true:
  - the dedicated RBAC worktree stays explicit
  - its branch stays explicit
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
