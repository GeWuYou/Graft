# Multi Worktree Governance Tracking

## Topic

- Topic: `multi-worktree-governance`
- Branch: `main`
- Worktree: `primary-main`
- Scope: archive the completed `mvp-extension-path` topic, govern shared repository truth on local `main`, and prepare
  stable ownership boundaries before creating long-lived worktrees from local branches.

## Goal

- Make the repository safe for multi-worktree execution by moving completed recovery state out of the active path,
  freezing shared governance on `main`, and defining which shared files must be integrated centrally before future
  long-lived worktrees are created.

## Repository Truth

- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/roadmap/MVP实施计划.md`

## Current Recovery Point

- `mvp-extension-path` has been completed as the old long-lived MVP topic and is no longer the default recovery entry on
  `main`; its full recovery materials now belong under `ai-plan/public/archive/mvp-extension-path/`.
- The repository is currently running from the root worktree on local `main`, with no additional long-lived worktrees
  created yet.
- The immediate governance task on `main` is not feature implementation. It is shared-baseline preparation for future
  worktree splits.
- Current boundary facts are frozen as follows:
  - `server` is already close to plugin-oriented parallel execution, and future long-lived worktree ownership should be
    plugin-first.
  - `web` still has shared shell hotspots such as bootstrap route mapping, global stores, layout wiring, and locale
    catalogs; `app/**` is also shell-owned during the root-pages retirement migration, and future long-lived worktrees
    must not treat those files as freely owned by multiple topics at once.
- The first web-side mitigation slice has landed on short branch `refactor/web-module-boundaries`:
  - `web/src/modules/` is now the real feature registration layer
  - bootstrap dynamic route declarations now resolve through module registrations instead of feature truth living in shared shell code
  - the real `user` and `rbac` business surface now lives under `web/src/modules/<name>/`
  - compatibility re-exports remain in shared API/model/contract entrypoints where needed, but module-owned code now carries the feature truth
  - module registration is now the only allowed new feature-to-shell integration path
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
- `web/src/permission.ts`
- `web/src/layouts/**`
- `web/src/locales/lang/zh-CN.json`
- `web/src/locales/lang/en-US.json`
- `web/src/router/index.ts`
- `web/src/app/**`

## Active Risks

- If a future long-lived worktree is created before shared hotspot ownership is frozen, the first merge wave will
  recreate hidden dual-truth and integration churn.
- If `web` continues to treat shell wiring and locale catalogs as normal feature-owned files, long-lived parallel work
  will conflict even when business pages are otherwise disjoint.
- If old recovery materials remain active after the branch has returned to `main`, future boot/recovery flows will land
  on historical work instead of the current governance baseline.

## Latest Validation

- Recovery index and governance truth were updated on local `main` after confirming:
  - `git branch --all --verbose --no-abbrev`
  - `git worktree list --porcelain`
  - `git status --short --branch`
- Documentation consistency was checked with:
  - `rg -n "multi-worktree-governance|primary-main|mvp-extension-path|long-lived worktree|默认恢复入口" ai-plan`
- Current frontend refactor slice was grounded with:
  - `find web/src -maxdepth 3 -type d | sort`
  - `sed -n '1,260p' web/src/utils/route/bootstrap.ts`
  - `sed -n '1,260p' web/src/router/index.ts`
  - `sed -n '1,260p' ai-plan/design/前端架构设计.md`
- Focused frontend validation for the first module-boundary slice passed with host Windows Bun:
  - `cd web && /mnt/c/Users/gewuyou/.bun/bin/bun.exe run test:run -- src/utils/route/bootstrap.test.ts src/utils/route/index.test.ts`
  - `cd web && /mnt/c/Users/gewuyou/.bun/bin/bun.exe run typecheck`
  - `cd web && /mnt/c/Users/gewuyou/.bun/bin/bun.exe run lint -- src/utils/route/bootstrap.ts src/modules/index.ts src/modules/rbac/bootstrap-routes.ts src/modules/user/bootstrap-routes.ts src/modules/types.ts`
- The landed module-boundary branch currently has no uncommitted owned-scope changes:
  - `git status --short --branch`
  - `git diff -- web/src/modules/user web/src/modules/rbac web/src/modules/index.ts ai-plan/public/multi-worktree-governance`

## Immediate Next Step

- Keep using `multi-worktree-governance` on local `main` until the repository has explicit owned-scope rules for the
  first real long-lived worktrees.
- Keep the landed module-boundary refactor as the baseline for future `web` worktree ownership:
  - preserve `web/src/modules/user/**` and `web/src/modules/rbac/**` as module-owned feature truth
  - preserve `web/src/app/**` and other shared shell code as consumers of module registrations instead of holders of feature route truth
  - keep compatibility bridges narrow so future cleanup can remove them without reopening feature ownership
- Before creating the first additional worktree, decide the exact owned scope and shared-hotspot policy for:
  - `RBAC`
  - `server-status-dashboard`
- Once the first real worktree/topic pair exists, add it to `ai-plan/public/README.md` and create its dedicated
  tracking/trace files instead of continuing to stage feature recovery on `main`.

## Web Owned Scope Freeze

- Future `web` long-lived worktrees should own one module boundary at a time:
  - `web/src/modules/user/**`
  - `web/src/modules/rbac/**`
  - future `web/src/modules/server-status/**` or equivalent dashboard module path
- `shell-owned` directories must stay centrally integrated and are not long-lived feature-owned scope:
  - `web/src/app/**`
  - `web/src/layouts/**`
  - `web/src/router/**`
  - `web/src/utils/route/**`
  - `web/src/store/modules/user.ts`
  - `web/src/store/modules/permission.ts`
  - `web/src/permission.ts`
  - `web/src/locales/**`
  - platform `web/src/contracts/**`
