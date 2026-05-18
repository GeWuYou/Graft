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

## 2026-05-18 web migration documentation slice

- Rechecked the live `web` tree for the documentation-only migration slice and confirmed `web/src/config/**` still
  exists alongside the landed `app/modules/shared` ownership baseline.
- Updated `web/AGENTS.md` and `ai-plan/design/前端架构设计.md` so `config/**` is now explicit shell-owned platform
  configuration instead of an implied leftover directory.
- Updated the active `multi-worktree-governance` tracking file to record:
  - what already moved into `modules/**` and `shared/**`
  - what remains shell-owned, including `config/**`
  - that archived `mvp-extension-path/web` materials are historical recovery context only
- Left validation reporting intentionally narrow and honest:
  - expected final frontend completion entry remains `cd web && bun run check`
  - no runtime validation result is recorded for this slice because only documentation/tracking files changed

## 2026-05-18 shell-owned runtime migration slice

- Moved the real web startup path into `web/src/app/bootstrap/**`, including app creation, route-guard wiring,
  restricted-session recovery, and the shell-owned permission directive, while reducing root `main.ts` to a thin
  bootstrap entrypoint.
- Moved provider composition into `web/src/app/providers/**`, reducing root `App.vue` to a thin shell-owned wrapper and
  keeping the existing `router-view + locale + theme workbench` runtime behavior unchanged.
- Added `web/src/modules/user/contract/paths.ts` as the single truth for `/users` and `/api/users`, rewired the user
  bootstrap route, user API client, header navigation, and user page endpoint display to consume that contract.
- Moved `user.*` and `rbac.*` locale catalogs into `web/src/modules/{user,rbac}/locales/**`, updated the root locale
  aggregator to merge module-owned catalogs, and removed the empty historical `components/`, `hooks/`, `contracts/user`,
  `contracts/rbac`, `directives/`, `constants/`, and `router/modules/` directories.
- Revalidated the runtime slice with targeted search checks, focused Vitest/typecheck, and one full host Windows Bun
  `bun run check` pass with the Vite warning surface back to zero.

## 2026-05-18 post-migration shell hotspot review

- Re-ran startup preflight on `refactor/web-module-boundaries`, then recovered through the active
  `multi-worktree-governance` topic plus archived `mvp-extension-path/web` historical tracking before reviewing the
  live shell-owned web surfaces.
- Confirmed the current worktree was clean and kept the slice review-only for `web/src/app/**`, `web/src/locales/**`,
  and module registration surfaces; multi-agent work was not justified because the scope stayed small and shell-owned.
- Focused validation stayed green with:
  - `cd web && bun run test:run -- src/utils/route/bootstrap.test.ts src/utils/route/title.test.ts src/locales/index.test.ts`
  - `cd web && bun run typecheck`
- The review found three shared-hotspot follow-ups to schedule before future long-lived web worktrees start depending on
  these shell surfaces:
  - module registration enforces duplicate `menuPath` only and still allows duplicate stable `routeName` values
  - login route name/path truth is still fragmented across shell-owned runtime files as bare literals instead of a
    canonical auth-route contract
  - locale aggregation still does a shallow top-level merge for module catalogs, so future module-owned `menu` or
    other shared namespaces can overwrite earlier shell/module messages

## 2026-05-18 shell hotspot closure for parallel web worktrees

- Reworked `web/src/modules/index.ts` from hand-written module imports into eager module auto-discovery over
  `modules/*/index.ts`, so future module onboarding no longer defaults to a shared registry edit.
- Extended registration validation so the shell now rejects duplicate `moduleId`, duplicate `menuPath`, duplicate stable
  parent route names, and duplicate derived child route names before route bootstrap can continue.
- Promoted login route name/path into the platform auth-route contract and replaced shell-owned `'/login'` / `'login'`
  literals across router, guards, request, permission, tabs-router, header logout, and theme-workbench visibility.
- Hardened locale aggregation with recursive deep merge semantics so module-owned locale trees can extend shared
  namespaces without top-level overwrite drift.
- Added focused regression coverage for:
  - module registration collision handling
  - router login route canonical contract usage
  - locale deep-merge semantics
- Revalidated the slice with repository-portable Bun command forms:
  - `cd web && bun run test:run -- src/modules/index.test.ts src/router/index.test.ts src/locales/index.test.ts src/permission.test.ts`
  - `cd web && bun run typecheck`
  - `cd web && bun run check`

## 2026-05-18 graft-commit staging diagnosis

- Re-ran startup preflight on `refactor/web-module-boundaries` for a docs/automation investigation into why
  `$graft-commit` appeared to miss files after a prior `$graft-push`.
- Confirmed the reported symptom was not a Git index bug:
  - `git status --short` showed five `web` files as ` M`
  - `git diff --cached --name-only` returned empty
  - that combination means the files were modified in the working tree but never staged into the Git index
- Root cause was workflow ambiguity, not command failure: the repository skill text did not state explicitly that IDE
  changelist checkboxes are not authoritative proof of Git staging state.
- Updated `.agents/skills/graft-commit/SKILL.md` so the workflow now requires explicit interpretation of `git status`
  columns, explains why `git diff --cached --name-only` can be empty, and forbids treating IDE selection UI as staged
  proof.

## 2026-05-18 server module-boundary governance baseline

- Re-ran startup preflight on `refactor/server-module-boundaries` for a docs/automation slice that converts the
  drafted `server` multi-worktree plan into repository truth.
- Added `graft-multi-agent-task` as a repository skill and created its skill folder as a thin wrapper around:
  - `graft-multi-agent-batch`
  - `graft-task-closeout`
  - `graft-commit`
- Corrected active recovery mapping so `ai-plan/public/README.md` and the tracking file now point at the live root
  branch `refactor/server-module-boundaries` instead of the earlier web refactor branch.
- Added `ai-plan/public/multi-worktree-governance/roadmap/server-module-boundaries-plan.md` as the topic-local formal
  plan for backend ownership freeze, phase-by-phase execution, and future plugin onboarding.
- Updated `server/AGENTS.md` and `ai-plan/design/插件与依赖注入设计.md` so backend governance now explicitly freezes:
  - compile-time modular monolith direction
  - plugin dependency restrictions
  - migration ownership rules
  - per-plugin Ent generation direction
  - shared-hotspot whitelist
  - no-business-logic-backflow constraints
  - future third-party compatibility boundaries without implementing runtime plugins now
- Recorded the resulting backend owned-scope baseline in the active topic tracking file so future worktree creation can
  reference one repository-local truth instead of chat-only planning output.

## 2026-05-18 server Phase 1 registry wiring

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and executed the slice through `graft-multi-agent-task`.
- Used a bounded multi-agent wave for two read-only explorer sidecars plus one worker-owned descriptor shim slice, while
  keeping `internal/plugin`, `internal/pluginregistry`, CLI integration, review, and validation on the local critical path.
- Added `plugin.Descriptor` and `plugin.Builder` to `server/internal/plugin`, keeping descriptor metadata as the
  canonical compile-time truth while still validating runtime plugin metadata drift during `Build()`.
- Added `server/internal/pluginregistry/**` with:
  - `registry.go` for descriptor snapshots, ordered runtime plugin construction, and default migration directory
    aggregation
  - `generate.go` plus `cmd/pluginregistrygen/main.go` for deterministic registry generation
  - generated hotspot `generated.go` as the only centralized plugin list
- Added `server/plugins/{audit,user,rbac,scheduler}/descriptor.go` so each existing plugin now owns its compile-time
  descriptor shim locally instead of relying on `serve.go` imports.
- Rewired `server/internal/cli/serve.go` to consume `pluginregistry.BuildPlugins()` and rewired
  `server/internal/cli/migrate.go` so the default migration path now resolves through registry-derived directory lists.
- Added focused regression coverage for:
  - descriptor and dependency ordering in `server/internal/plugin/plugin_test.go`
  - registry generator determinism and missing-descriptor failure in
    `server/internal/pluginregistry/cmd/pluginregistrygen/main_test.go`
  - registry-driven serve and migrate behavior in `server/internal/cli/serve_test.go` and
    `server/internal/cli/migrate_test.go`
- Validation for the slice finished with:
  - `cd server && go test ./internal/plugin ./internal/pluginregistry/... ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage buildtest`
- Immediate next step after this slice:
  - preserve the new compile-time registry seam as the only central wiring path
  - choose one Phase 2 boundary for plugin-private store/capability migration instead of reopening core-owned wiring
