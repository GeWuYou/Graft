# Multi Worktree Governance Tracking

## Topic

- Topic: `multi-worktree-governance`
- Branch: `refactor/web-module-boundaries`
- Worktree: repository root only; no dedicated long-lived worktree exists yet
- Scope: keep `mvp-extension-path` archived, reconcile recovery truth with the current root-branch reality, freeze the
  final post-compatibility `web` ownership model, and prepare stable owned scopes before creating dedicated long-lived
  worktrees.

## Goal

- Make the repository safe for multi-worktree execution by moving completed recovery state out of the active path,
  freezing shared governance on the current repository root, and defining which files must be integrated centrally
  before future long-lived worktrees are created.

## Repository Truth

- `AGENTS.md`
- `web/AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/roadmap/MVP实施计划.md`

## Current Recovery Point

- `mvp-extension-path` has been completed as the old long-lived MVP topic; its recovery materials now belong under
  `ai-plan/public/archive/mvp-extension-path/` and are no longer the default active entry.
- The repository is currently running from the root worktree on branch `refactor/web-module-boundaries`; `git worktree list`
  shows no additional worktrees.
- The immediate governance task on this branch is not to preserve compatibility bridges. It is to lock the final
  post-compatibility ownership model and recovery truth before the first dedicated long-lived worktree is created.
- Repository governance is now split into:
  - root `AGENTS.md` for startup, recovery, validation ownership, closeout/commit, and subagent rules
  - `web/AGENTS.md` for frontend execution truth
  - `server/AGENTS.md` for backend execution truth
- Current boundary facts are frozen as follows:
  - `server` is already close to plugin-oriented parallel execution, and future long-lived worktree ownership should be
    plugin-first.
  - `web` final ownership now follows three explicit layers:
    - `shell-owned`: `web/src/app/**`, `web/src/app/bootstrap/**`, `web/src/app/providers/**`, `web/src/layouts/**`,
      `web/src/router/**`, `web/src/config/**`, `web/src/utils/route/**`, `web/src/store/modules/user.ts`,
      `web/src/store/modules/permission.ts`, `web/src/locales/**`, platform `web/src/contracts/**`, and other platform
      bootstrap surfaces
    - `module-owned`: `web/src/modules/<name>/**` holds page, API, type, contract, locale, and bootstrap-route truth
      for one module
    - `shared-owned`: `web/src/shared/**` is reserved for business-agnostic reusable assets and is not feature-owned by
      any single module worktree
- The web boundary refactor has already landed on branch `refactor/web-module-boundaries`:
  - `web/src/modules/` is now the real feature registration layer
  - bootstrap dynamic route declarations now resolve through module registrations instead of feature truth living in shared shell code
  - the real `user` and `rbac` business surface now lives under `web/src/modules/<name>/`
  - reusable shell UI and composables now live under `web/src/shared/**`
  - module registration is now the only allowed new feature-to-shell integration path
  - root-level module files under `web/src/api/**`, `web/src/api/model/**`, and `web/src/contracts/{user,rbac}/**`
    have been removed from the codebase and are no longer valid steady-state ownership surfaces
  - cross-module stable DTOs now follow the final rule that they must live under `modules/<name>/contract/**`, while
    `types/**` stays private to the owning module
- This documentation/tracking-only migration slice on `2026-05-18` records the remaining shell-owned truth explicitly:
  - `web/src/config/**` is still present in the live tree and remains platform configuration owned by the shell layer
  - no runtime files were changed in this slice; it only updates governance, active tracking, and archived historical recovery notes
  - archived `mvp-extension-path/web` materials remain historical context only and must not be treated as the active recovery entry
- The first expected future long-lived feature directions are:
  - `RBAC`
  - `server-status-dashboard`
- Those directions must not be registered as active topics until each one has:
  - a real local branch
  - a real long-lived worktree
  - a declared owned scope
  - a clear shared-hotspot integration path

## Shared Hotspots

- `server/internal/app/runtime.go`
- `server/internal/store/factory.go`
- `server/internal/store/entstore/factory.go`
- `server/internal/pluginapi/**`
- `server/internal/ent/schema/**`
- migrations
- `web/src/utils/route/bootstrap.ts`
- `web/src/store/modules/user.ts`
- `web/src/store/modules/permission.ts`
- `web/src/layouts/**`
- `web/src/locales/lang/zh-CN.json`
- `web/src/locales/lang/en-US.json`
- `web/src/router/index.ts`
- `web/src/app/**`
- `web/src/shared/**`

## Guardrails Outside Final Web Standard

- do not recreate root-level module-specific files under `web/src/api/**`, `web/src/api/model/**`, or
  `web/src/contracts/{user,rbac}/**`
- do not recreate root-level reusable UI/composable/helper truth under `components/`, `hooks/`, or non-platform
  `utils/` when the asset belongs in `web/src/shared/**`

## Active Risks

- If a future long-lived worktree is created before shared hotspot ownership is frozen, the first merge wave will
  recreate hidden dual-truth and integration churn.
- If `web` continues to let module-specific truth linger under root `api/model/contracts` surfaces, future worktrees
  will keep competing over files that are no longer valid steady-state ownership boundaries.
- If the recovery index is not refreshed when the repository root branch changes again, future boot/recovery flows will
  land on stale branch/worktree assumptions instead of current governance truth.

## Latest Validation

- Recovery truth was grounded against the current repository state with:
  - `pwd`
  - `git branch --show-current`
  - `git worktree list`
  - `git status --short`
- Documentation consistency was checked with:
  - `rg -n "multi-worktree|worktree|兼容|compat|shared/|app/|modules/|refactor/web-module-boundaries|primary-main|main" ai-plan/public/multi-worktree-governance ai-plan/design/前端架构设计.md ai-plan/public/README.md`
- AGENTS split consistency was checked with:
  - `rg -n "web/AGENTS.md|server/AGENTS.md|Subdomain governance|执行真值|前端执行级治理真值|后端执行级治理真值" AGENTS.md web/AGENTS.md server/AGENTS.md ai-plan/design/AI任务追踪与恢复设计.md ai-plan/design/前端架构设计.md ai-plan/design/插件与依赖注入设计.md`
- Current frontend structure and ownership surfaces were grounded with:
  - `find web/src -maxdepth 3 -type d | sort`
  - `rg --files web/src | rg "^(web/src/(api|contracts|app|modules|components|store|router|shared|pages|hooks|utils))"`
  - `rg --files web/src | rg '(^web/src/config/|/config/)'`
  - `sed -n '1,260p' ai-plan/design/前端架构设计.md`
- Cross-module DTO boundary cleanup was checked with:
  - `rg -n "from '@/modules/[^']+/types/|from \\\"@/modules/[^\\\"]+/types/\" web/src/modules`
- This slice used targeted consistency searches plus documentation/tracking updates only; no full `web` runtime validation
  was run because the owned scope is docs/tracking and this change did not modify runtime code.
- Validation expectation for eventual runtime completion remains unchanged:
  - `cd web && bun run check`
- Later runtime migration on the same branch completed the behavior-preserving shell/module cleanup and revalidated with:
  - `rg -n '"user"|"rbac"' web/src/locales/lang/zh-CN.json web/src/locales/lang/en-US.json`
  - `rg -n "from '@/modules/[^']+/(types|api|pages|locales)|from \\\"@/modules/[^\\\"]+/(types|api|pages|locales)\\\"" web/src`
  - `rg -n \"'/users'|\\\"/users\\\"|'/api/users'|\\\"/api/users\\\"\" web/src --glob '!web/src/modules/user/contract/paths.ts' --glob '!web/src/**/*.test.ts'`
  - `cd web && /mnt/c/Users/gewuyou/.bun/bin/bun.exe run test:run -- src/permission.test.ts src/locales/index.test.ts src/modules/user/pages/index.test.ts`
  - `cd web && /mnt/c/Users/gewuyou/.bun/bin/bun.exe run typecheck`
  - `cd web && /mnt/c/Users/gewuyou/.bun/bin/bun.exe run check`

## Immediate Next Step

- Keep using `multi-worktree-governance` on the current root branch until the repository either returns to `main` with
  the same baseline or creates the first dedicated worktree/topic pair.
- Keep the landed module-boundary refactor as the baseline for future `web` worktree ownership:
  - preserve `web/src/modules/user/**` and `web/src/modules/rbac/**` as module-owned feature truth
  - preserve `web/src/app/**` and other shell-owned code as consumers of module registrations instead of holders of
    feature route truth
  - preserve `web/src/shared/**` as the only valid cross-module reusable asset layer
- Before creating the first additional worktree, decide the exact owned scope and shared-hotspot policy for:
  - `RBAC`
  - `server-status-dashboard`
- Once the first real worktree/topic pair exists, add it to `ai-plan/public/README.md` and create its dedicated
  tracking/trace files instead of continuing to stage feature recovery on the root branch.
- Use the new split governance layout as the baseline for future worktree setup:
  - root `AGENTS.md` remains the only startup-governance source
  - `web/AGENTS.md` and `server/AGENTS.md` own daily execution rules inside their boundaries

## Web Owned Scope Freeze

- Future `web` long-lived worktrees should own one module boundary at a time:
  - `web/src/modules/user/**`
  - `web/src/modules/rbac/**`
  - future `web/src/modules/server-status/**` or equivalent dashboard module path
- `shell-owned` directories must stay centrally integrated and are not long-lived feature-owned scope:
  - `web/src/app/**`
  - `web/src/layouts/**`
  - `web/src/router/**`
  - `web/src/config/**`
  - `web/src/utils/route/**`
  - `web/src/store/modules/user.ts`
  - `web/src/store/modules/permission.ts`
  - `web/src/permission.ts`
  - `web/src/locales/**`
  - platform `web/src/contracts/**`
- `shared-owned` runtime assets must stay business-agnostic and centrally integrated:
  - `web/src/shared/**`
- Root-level module-specific files under `web/src/api/**`, `web/src/api/model/**`, and `web/src/contracts/{user,rbac}/**`
  are not valid long-lived owned scope and must not be reintroduced by a feature worktree.
