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
- Landed a real `web/src/modules/` registration layer instead of keeping bootstrap dynamic route declarations in shared
  shell code.
- Moved the real `user` and `rbac` page/api/type/contract surface under `web/src/modules/<name>/`, while keeping
  narrow compatibility re-exports at shared entrypoints that are still consumed elsewhere in `web`.
- Confirmed the shell/module boundary now follows the intended direction:
  - shared shell code consumes module registrations
  - module directories hold feature bootstrap route truth
  - future long-lived `web` owned scope can freeze on module boundaries instead of technical-layer directories
- Rechecked the branch after landing the slice and found no remaining uncommitted owned-scope changes in
  `web/src/modules/user/**`, `web/src/modules/rbac/**`, `web/src/modules/index.ts`, or the active topic docs.

## Historical Next Step At That Time

- Continue governing shared repository truth on `main` until the first real long-lived worktree/topic pair is created,
  starting with exact owned-scope rules and shared-hotspot integration policy for `RBAC` and `server-status-dashboard`.

## 2026-05-18 documentation reconciliation on refactor/web-module-boundaries

- Rechecked the live repository state and confirmed the current active recovery path is the repository root on branch
  `refactor/web-module-boundaries`, with no additional worktrees reported by `git worktree list`.
- Updated `ai-plan/public/README.md` and the topic tracking file so they no longer assume the root worktree is
  `primary-main` on local `main`.
- Updated `ai-plan/design/前端架构设计.md` to the final post-compatibility ownership model:
  - `app/**` is shell-owned
  - `modules/<name>/**` is the only valid module-owned feature truth
  - `shared/**` is the only valid cross-module business-agnostic reuse layer
  - root-level module-specific `api/model/contract` files are no longer part of the steady-state standard
- Recorded the remaining structural gap honestly at that time: code cleanup still had to remove root-level
  module-specific files and land `web/src/shared/**`.

## 2026-05-18 final web code cleanup on refactor/web-module-boundaries

- Confirmed the shared-layer migration landed as scoped commit `71ed60d` after passing the full host-Bun completion
  chain.
- Removed the remaining root-level module-specific compatibility files:
  - `web/src/api/user.ts`
  - `web/src/api/rbac.ts`
  - `web/src/api/model/userModel.ts`
  - `web/src/api/model/rbacModel.ts`
  - `web/src/contracts/user/**`
  - `web/src/contracts/rbac/**`
  - `web/src/constants/index.ts`
- With this cleanup, the codebase now matches the final ownership standard:
  - module truth lives under `web/src/modules/**`
  - shared reusable shell assets live under `web/src/shared/**`
  - platform contracts remain under `web/src/contracts/**`
  - root-level business compatibility bridges are no longer part of the runtime surface

## 2026-05-18 AGENTS split and contract-boundary follow-up

- Split repository governance into one root startup-governance document plus two subdomain execution-truth documents:
  - root `AGENTS.md` now keeps repository-only governance such as startup, recovery, validation ownership, commit flow,
    CI/CD, subagent rules, and done-state rules
  - `web/AGENTS.md` now owns frontend execution truth
  - `server/AGENTS.md` now owns backend execution truth
- Updated `ai-plan/design/AI任务追踪与恢复设计.md` so recovery design explicitly describes the new layering and keeps
  `ai-plan/` out of daily execution-rule ownership.
- Updated `ai-plan/design/前端架构设计.md` and `ai-plan/design/插件与依赖注入设计.md` so they remain architecture-truth
  documents while deferring execution-level rule lists to the new subdomain `AGENTS.md` files.
- Narrowed the remaining `web` module boundary by moving `rbac` DTOs that are consumed outside the module into
  `web/src/modules/rbac/contract/**`, then updating `user` and internal `rbac` call sites to stop using
  cross-module `types/**` imports.
- The resulting frontend baseline is now explicit:
  - shell code consumes module registrations and stable module contracts
  - cross-module stable DTOs live in `modules/<name>/contract/**`
  - `modules/<name>/types/**` stays private to the owning module

## Next Step

- Keep the recovery mapping aligned with the current root branch until the repository returns to `main` or the first
  dedicated worktree/topic pair is created.
- Freeze the first additional `web` worktree owned scopes against the now-landed final ownership baseline.
