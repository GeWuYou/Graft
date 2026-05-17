# Multi Worktree Governance Trace

## 2026-05-17 topic bootstrap on main

- Confirmed the repository has returned to local `main` and no additional long-lived worktrees exist yet.
- Decided to stop carrying `mvp-extension-path` as an active topic on `main`; that topic has been completed and archived.
- Established `multi-worktree-governance` as the new active recovery entry on local `main`.
- Restricted this new topic to shared-baseline governance only:
  - archive completed topic recovery
  - define multi-worktree mapping rules
  - freeze shared hotspot ownership expectations
  - prepare future per-worktree active topics

## 2026-05-17 first web boundary refactor slice

- Created short branch `refactor/web-module-boundaries` from local `main` for the first frontend structure refactor.
- Started landing a real `web/src/modules/` registration layer instead of keeping bootstrap dynamic route declarations in
  shared shell code.
- Tightened the slice scope after rechecking the real runtime surface:
  - the real business pages are `user` and `rbac`
  - static shell pages still in runtime are `login` plus `result/403|404|500`
  - stale starter/demo result pages should be removed instead of kept as dormant runtime residue
  - module registrations remain the only allowed feature-to-shell integration path
  - subsequent code migration should move `user` and `rbac` page/api/type/contract ownership into `web/src/modules/<name>/`

## Next Step

- Continue governing shared repository truth on `main` until the first real long-lived worktree/topic pair is created,
  starting with exact owned-scope rules for `RBAC` and `server-status-dashboard`.
