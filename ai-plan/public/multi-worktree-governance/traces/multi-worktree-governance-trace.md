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
  - continue Phase 2 by removing centralized business-store entry from plugin runtime wiring before opening the first
    long-lived backend feature worktrees

## 2026-05-18 server readiness review for parallel worktrees

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and reviewed the live backend wiring against the topic plan and
  `server/AGENTS.md`.
- Confirmed the compile-time registry baseline is live:
  - `plugin.Descriptor.Build()` is the canonical runtime construction boundary
  - `serve` now builds plugins from `pluginregistry.BuildPlugins()`
  - `migrate` now resolves default migration directories through `pluginregistry.MigrationDirs()`
  - `server/internal/pluginregistry/generated.go` is the only centralized plugin list
- Confirmed the service/store decoupling is only partial:
  - `server/plugins/rbac/store/**` and `server/plugins/user/store/**` now own plugin-private repository contracts
  - both plugins still reach persistence through temporary `storeadapter/internal_store.go` seams over
    `server/internal/store/**`
  - `plugin.Context` still exposes `Stores store.Factory`, so runtime plugin wiring still offers a centralized
    business-store entrypoint
  - `server/plugins/user/bootstrap_admin.go` still contains a repository-backed RBAC bootstrap compatibility helper
    over `internalstore.RBACRepository`
- Confirmed the largest merge hotspots remain unresolved:
  - business Ent schema and generated code still live in `server/internal/ent/**`
  - default Atlas migration truth still starts from `internal/ent/migrate/migrations`
  - CLI dev/reset flow still depends on centralized `internal/store/**`
- Conclusion:
  - the branch can already support limited parallel feature work inside plugin-local HTTP/service logic
  - it does not yet meet the stricter goal of low-conflict long-lived parallel worktrees for persistence/schema-heavy
    plugin development
- Immediate next step:
  - complete Phase 2 runtime decoupling first, then start Phase 3 Ent/migration ownership migration before declaring
    backend worktree readiness

## 2026-05-18 server Phase 2d builder-wiring decoupling

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and implemented the next backend decoupling slice locally without
  multi-agent work because the write scope stayed tightly coupled across plugin/runtime assembly boundaries.
- Moved runtime plugin construction behind explicit core-owned build context:
  - `plugin.Builder` now consumes `plugin.BuildContext`

## 2026-05-19 server Phase 3d user plugin-owned Ent path

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and executed the bounded round through `graft-multi-agent-task`.
- Kept the immediate blocking work local; `graft-multi-agent-batch` was not used because the RBAC write narrowing,
  plugin-owned user Ent import path, and migration checkpoint stayed tightly coupled inside one allowed-scope slice.
- Narrowed the remaining RBAC shared generated-user dependency without changing stable write semantics:
  - `server/internal/store/entstore/rbac_repository.go` now checks user existence for `AssignRoleToUser` and
    `ReplaceRolesForUser` through direct shared-client `User.Get` lookups instead of importing the shared generated
    `internal/ent/user` package
  - added focused regression coverage in `server/internal/store/entstore/rbac_repository_test.go` for the missing-user
    write path
- Introduced the first plugin-owned generated Ent import surface under `server/plugins/user/ent/**`:
  - added `go generate ./plugins/user/ent` with a local alias generator that emits plugin-owned wrappers for the
    user-owned `user` and `refreshsession` generated packages
  - rewired `server/plugins/user/storeent/{user_repository,auth_repository}.go` to consume those plugin-owned import
    paths instead of directly importing `server/internal/ent/user` and `server/internal/ent/refreshsession`
- Introduced the first runnable forward-only user migration state under `server/plugins/user/migrations/**`:
  - added no-op checkpoint `202605190001_user_plugin_boundary_checkpoint.sql`
  - generated `plugins/user/migrations/atlas.sum` so the plugin-owned migration directory now participates in the
    default registry-driven migration chain without claiming ownership of the historical mixed Atlas files
- Validation completed with:
  - `cd server && go generate ./plugins/user/ent`
  - `cd server && atlas migrate hash --dir file://plugins/user/migrations`
  - `cd server && go test ./internal/store/entstore ./plugins/user/storeent ./internal/cli`
  - `cd server && go test ./plugins/user/...`
- Honest remaining gap after this slice:
  - the runtime still shares `server/internal/ent.Client`; a separate plugin-owned user Ent client would require
    out-of-scope runtime/container changes
  - the mixed historical Atlas root remains immutable and `user_roles` ownership is still shared with `rbac`
  - `pluginregistry.BuildPlugins(...)` now requires that build context
  - `app.NewRuntime()` now constructs plugins only after core services and `store.Factory` exist
  - `serve` no longer constructs plugin instances directly
- Narrowed active plugin runtime dependencies:
  - `user`, `rbac`, and `audit` plugins now receive their repository adapters during construction
  - active plugin lifecycle code no longer reads repositories from `plugin.Context.Stores`
  - `plugin.Context` no longer exposes `Stores`, reducing the chance of new runtime business-store backflow
- Closed the remaining default-admin compatibility seam introduced by earlier slices:
  - removed the repository-backed RBAC bootstrap helper from `server/plugins/user/bootstrap_admin.go`
  - `user` dev reset now consumes the stable `pluginapi.RBACBootstrapService`
  - `server/internal/cli/dev_reset.go` now builds that capability through the RBAC plugin-local adapter path
- Validation for the slice finished with:
  - `cd server && go test ./internal/plugin ./internal/pluginregistry/... ./internal/app ./internal/cli ./plugins/user ./plugins/rbac ./plugins/audit ./plugins/scheduler`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
- Resulting readiness update:
  - Phase 2 is still not fully complete because `store.Factory` remains registered inside core and Ent/migration
    ownership is still centralized
  - but active plugin lifecycle code is now materially closer to the target multi-worktree model because repository
    wiring moved from runtime context access into compile-time builder/runtime assembly
- Immediate next step:
  - begin Phase 3 by defining the first plugin-owned Ent/migration split, starting with the smaller/lower-risk owner
    between `rbac` and `user`
  - preserve the new compile-time registry seam as the only central wiring path
  - choose one Phase 2 boundary for plugin-private store/capability migration instead of reopening core-owned wiring

## 2026-05-19 server Phase 3 owner-decision governance checkpoint

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and executed the bounded round through `graft-multi-agent-task`.
- Kept the slice local; `graft-multi-agent-batch` was not used because the change was a tightly coupled governance
  update across `server/AGENTS.md` and active `ai-plan/**` truth.
- Converted the newly approved Phase 3 owner decision into repository truth without touching production code:
  - `server/AGENTS.md` now states that `user_roles` has one final owner, `rbac`
  - backend governance now explicitly assigns `user_roles` Ent schema, repository, migration, and test ownership to
    `rbac`
  - backend governance now explicitly forbids `user` from owning `user_roles` or exposing role-assignment
    implementation details
  - the topic roadmap and plugin/DI design now require `rbac` to validate `user_id` through stable `user` capability
    or contract surfaces instead of importing `user` Ent packages
  - multi-worktree rules now explicitly allow `RBAC` worktrees to modify `user_roles` while forbidding direct
    `user_roles` edits from `User` worktrees
  - the active topic tracking file now records that the historical mixed Atlas root remains immutable and that any
    ownership checkpoint must land later as an `rbac` forward-only migration
  - this checkpoint supersedes the earlier historical note from the Phase 3d trace that still described `user_roles`
    ownership as shared
- Validation stayed documentation-only and intentionally narrow:
  - no backend code validation was required because no production code, generated code, or migration files changed in
    this round
  - consistency was checked with targeted repository search over the changed governance and tracking files
- Immediate next step:
  - execute the remaining `rbac` persistence extraction and ownership-checkpoint slice under the now-frozen
    `user_roles -> rbac` truth

## 2026-05-18 server Phase 2a service-capability decoupling

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and executed the slice through `graft-multi-agent-task`.
- Used a bounded multi-agent wave for the initial split, but kept acceptance and final implementation on the local
  critical path after both worker slices stopped before landing code:
  - one worker confirmed the `pluginapi + rbac` contract shape and hit only an `apply_patch` context mismatch
  - one worker confirmed the `user` call sites and correctly reported that the shared `pluginapi` contracts had to
    land before a user-only refactor could complete
- Landed the decoupling locally with these boundary changes:
  - added `server/internal/pluginapi/rbac.go` with stable `PermissionSeed`, `RBACAccessService`, and
    `RBACBootstrapService`
  - registered RBAC access/bootstrap services in `server/plugins/rbac/**`
  - removed runtime `rbac -> ctx.Stores.Users()` coupling by switching read-management existence checks to
    `pluginapi.UserService`
  - removed runtime `user -> RBACRepository` coupling in boot/bootstrap paths by introducing deferred RBAC access
    binding plus RBAC bootstrap capability consumption
  - kept the dev-only `ResetDefaultAdminForDevelopment` CLI shape stable by adapting the repository input behind a
    private compatibility adapter instead of broadening the core slice
- Revalidated the slice with:
  - `cd server && go test ./plugins/rbac ./plugins/user ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend`
- Immediate next step after this slice:
  - keep the new RBAC capability seam as the only allowed user/rbac cross-plugin path
  - move the next Phase 2 slice to plugin-private `store/**` / `storeent/**` ownership instead of touching runtime
    capability wiring again

## 2026-05-18 server Phase 2b RBAC private-store contract migration

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and executed the slice through `graft-multi-agent-task`.
- Used a bounded multi-agent wave for two read-only explorer sidecars to compare `user` vs `rbac` as the next
  plugin-private store target, then kept implementation, review, and validation on the local critical path.
- Chose `rbac` for this slice because the owned scope excludes `server/internal/store/**`, and `rbac` had the smaller
  plugin-local contract move that could land without reopening auth/session or direct `user` / `rbac` repository
  coupling.
- Landed the migration with these boundary changes:
  - added `server/plugins/rbac/store/**` as the RBAC plugin-owned repository contract surface
  - rewired `server/plugins/rbac/**` runtime code to consume that local contract instead of
    `server/internal/store/rbac.go`
  - added `server/plugins/rbac/storeadapter/internal_store.go` as the temporary adapter from the shared
    `ctx.Stores.RBAC()` seam into the plugin-local repository contract
  - added `pluginapi.ErrUserNotFound` in `server/internal/pluginapi/user.go` and updated `server/plugins/user/**` to
    map user-not-found reads onto that shared cross-plugin semantic
  - kept `pluginapi.RBACAccessService` / `RBACBootstrapService` unchanged, so cross-plugin behavior still flows only
    through the already-landed service-capability seam
- Revalidated the slice with:
  - `cd server && go test ./plugins/rbac`
  - `cd server && go test ./plugins/rbac ./plugins/user ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend`
- Immediate next step after this slice:
  - keep the new plugin-local RBAC contract as the only allowed RBAC repository dependency inside
    `server/plugins/rbac/**`
  - migrate the remaining RBAC persistence implementation ownership out of `internal/store/**` and
    `internal/store/entstore/**` in a later slice, or start the separate `user` private-store migration without
    reopening direct repository coupling

## 2026-05-18 server Phase 2c user private-store contract migration

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and executed the slice through `graft-multi-agent-task`.
- Used a bounded multi-agent wave for two read-only explorer sidecars to confirm the narrowest compatibility seam, then
  kept acceptance and final implementation on the local critical path.
- Landed the migration with these boundary changes:
  - added `server/plugins/user/store/**` as the user plugin-owned user/auth/session repository contract surface
  - added `server/plugins/user/storeadapter/internal_store.go` as the temporary adapter from
    `ctx.Stores.{Users,Auth}` into the plugin-local repository contract
  - rewired `server/plugins/user/**` runtime code to consume the local contract instead of directly importing
    `server/internal/store/{user,auth}.go`
  - preserved the dev-only `graft dev reset-admin` command shape by adapting the shared auth repository input inside
    `server/internal/cli/dev_reset.go`
  - kept the remaining `internal/store` dependency inside `server/plugins/user/**` limited to the dev-only RBAC
    bootstrap compatibility helper instead of reopening runtime `user -> rbac` repository coupling
- Revalidated the slice with:
  - `cd server && go test ./plugins/user ./internal/cli`
  - `cd server && go test ./plugins/rbac ./plugins/user ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend`
- Immediate next step after this slice:
  - keep the new plugin-local user contract as the only allowed direct user/auth/session repository dependency inside
    `server/plugins/user/**`
  - move the next Phase 2/3 slice to plugin-owned persistence implementation ownership under `storeent/**`, `ent/**`,
    and `migrations/**` instead of reopening direct shared-store imports

## 2026-05-18 server Phase 2e explicit-repository builder decoupling

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and executed the slice through `graft-multi-agent-task`.
- Used a bounded multi-agent wave for two read-only explorer sidecars while keeping the actual implementation on the
  local critical path because the write scope stayed tightly coupled across `plugin.BuildContext`, runtime service
  registration, and plugin descriptors.
- Closed the remaining generalized builder/container backdoor without starting Phase 3 prematurely:
  - removed `store.Factory` from `plugin.BuildContext`
  - added `plugin.ResolveService(...)` as the explicit typed singleton resolution helper for builder wiring
  - rewired `server/internal/app/runtime.go` to register `store.{Audit,User,Auth,RBAC}Repository` singletons instead
    of exporting `store.Factory` through the shared runtime container
  - rewired `server/plugins/{audit,rbac,user}/descriptor.go` so each Builder resolves only the repository boundary it
    actually needs, then adapts that repository into plugin-local store contracts
- Updated the audit plugin README and active topic tracking so repository truth no longer claims that plugins receive
  `Stores` through `plugin.Context`.
- The read-only Phase 3 sidecar recommended `user` as the first plugin-owned Ent/migration owner:
  - `users` plus `refresh_sessions` are the smaller extraction than `rbac`
  - the mixed `user_roles` bridge should stay out of the first user-owned Ent/migration cut
- Validation for the slice finished with:
  - `cd server && go test ./internal/plugin ./internal/pluginregistry/... ./internal/app ./plugins/user ./plugins/rbac ./plugins/audit`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
- Resulting readiness update:
  - active plugin lifecycle code and compile-time builders no longer expose a generalized `store.Factory` backdoor
  - the next honest backend hotspot is Phase 3 ownership migration for `internal/store/**`, `internal/store/entstore/**`,
    `internal/ent/**`, and Atlas migrations, starting with the `user` plugin's independent `users` /
    `refresh_sessions` slice or a deliberately scoped dev CLI cleanup if that path is chosen first

## 2026-05-18 server Phase 3a user storeent extraction

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and executed the slice through `graft-multi-agent-task`.
- Used a bounded multi-agent wave for two read-only explorer sidecars to map:
  - the `user` vs `user_roles` / `rbac` Ent coupling points
  - the runtime, builder, dev CLI, and migrate entrypoints that would be touched by the first `user` persistence move
- Kept the implementation critical path local and chose the smallest honest Phase 3 slice:
  - move live `user` persistence implementation ownership into `server/plugins/user/storeent/**`
  - expose the shared `*ent.Client` explicitly to builders
  - avoid pretending the mixed historical Atlas chain or shared Ent graph has already been fully split
- Landed the runtime ownership move with these boundary changes:
  - added `server/plugins/user/storeent/**` as the plugin-owned Ent-backed implementation of the user/auth/session
    repository contract
  - rewired `server/plugins/user/descriptor.go` to resolve the shared `*ent.Client` from the runtime container and
    construct plugin-owned repositories directly instead of adapting shared `internal/store` repositories
  - rewired `server/internal/app/runtime.go` to register the shared `*ent.Client` as an explicit singleton service for
    compile-time builder wiring
  - rewired `server/internal/cli/dev_reset.go` so the dev-only default-admin reset path now enters the user plugin
    through the plugin-owned auth repository contract instead of the old user auth compatibility adapter
- Landed the migration-boundary groundwork without rewriting Atlas history:
  - added `server/plugins/user/migrations/README.md` as the plugin-owned migration boundary marker for future
    user-only versions
  - intentionally kept the mixed historical `202605140001_auth_rbac_foundation.sql` chain under
    `server/internal/ent/migrate/migrations/**` because it still owns `users`, `refresh_sessions`, `user_roles`,
    `roles`, `permissions`, and `role_permissions` together
- Explicit non-goals left for the next Phase 3 slice:
  - no plugin-owned `server/plugins/user/ent/**` generate path yet
  - no split of the shared Ent graph around `user_roles`
  - no rewrite of the existing mixed Atlas revision history
- Focused validation for the slice finished with:
  - `cd server && go test ./internal/app ./internal/cli ./plugins/user ./plugins/rbac ./plugins/user/storeent`
- Immediate next step after this slice:
  - split the remaining shared Ent/schema and migration history along the `users` / `refresh_sessions` vs
    `user_roles` / `rbac` boundary without reopening runtime store-factory coupling

## 2026-05-18 docs automation multi-agent loop orchestration

- Re-ran startup preflight on `refactor/server-module-boundaries`, classified the work as `docs/automation`, recovered
  through the active `multi-worktree-governance` parent topic, and implemented the looped orchestration slice locally
  without subagents because the write scope stayed tightly coupled across repository skill text, a new automation
  runner, and recovery/governance docs.
- Added `.agents/skills/graft-multi-agent-loop/**` as a new repository skill that initially wrapped repeated fresh-session
  `graft-multi-agent-task` execution behind an explicit budget:
  - `max_rounds`
  - `max_files_changed`
  - `max_commits`
  - `max_runtime_minutes`
  - `allowed_scopes`
  - validation-failure stop policy
- Added `scripts/run_loop.py` as the first standard-library Python runner prototype that:
  - launches `codex exec --ephemeral` child sessions
  - injects inherited startup context plus remaining budget into each round prompt
  - parses the child closeout JSON first and falls back to `Next-session startup prompt:` only when JSON is missing
  - stops on repeated prompts, scope expansion, high risk, validation failure, or budget exhaustion
- Added focused unit coverage in `scripts/test_run_loop.py` for:
  - JSON closeout parsing
  - keyword fallback parsing
  - malformed or contradictory closeout rejection
  - stop-condition evaluation
  - dirty-worktree refusal
  - two-round stubbed loop execution
- Updated `graft-multi-agent-task` and `graft-task-closeout` so loop-orchestrated runs now have one dual-channel
  closeout contract:
  - human-readable closeout first
  - `Next-session startup prompt:` only when another round is required
  - one trailing fenced JSON block with status, validation, commit, budget, scope, and risk fields
- Updated root `AGENTS.md` and the active topic tracking so repository governance now recognizes
  `graft-multi-agent-loop` as a repository skill without treating it as a second startup or closeout system.

## 2026-05-18 server Phase 3b migration-directory gating

- Re-ran startup preflight on `refactor/server-module-boundaries`, recovered through the active
  `multi-worktree-governance` parent topic, and attempted to execute the next Phase 3 slice through
  `graft-multi-agent-loop`.
- The loop orchestration itself was not reliable enough to trust as the execution record for this server slice:
  - the first fresh-session child stalled without emitting the required closeout JSON
  - the stalled child still left one owned diff in `server/internal/cli/migrate.go`
  - after reviewing that diff locally, kept the coherent part and completed the slice directly on the main critical path
- Chose the smallest honest follow-up after Phase 3a:
  - do not pretend the shared Ent graph or mixed Atlas revision history has already been split
  - instead, make the default migrate chain stop treating empty plugin-owned migration directories as runnable history
- Landed the slice in `server/internal/cli/migrate.go` by:
  - keeping explicit `graft migrate up --dir ...` behavior unchanged
  - filtering the default registry-driven migration directory list so only directories containing `atlas.sum` are
    included in the automatic apply chain
  - preserving `plugins/user/migrations/**` as the declared plugin-owned future boundary without forcing it into the
    live Atlas execution path before the directory actually owns versioned history
- Added focused regression coverage in `server/internal/cli/migrate_test.go` for:
  - skipping registry-declared migration directories that do not yet contain Atlas state
  - preserving explicit `--dir` execution even when the target directory has no `atlas.sum`
  - keeping the existing sequential registry-apply behavior once both core and plugin directories have Atlas state
- Validation for the slice finished with:
  - `cd server && go test ./internal/cli`

## 2026-05-18 docs automation multi-agent loop contract shift

- Re-ran startup preflight on `refactor/server-module-boundaries`, classified the work as `docs/automation`, and
  returned to the active `multi-worktree-governance` parent topic to replace the unstable fresh-session loop contract.
- Promoted the observed failure mode into the repository skill design instead of leaving it as an ad-hoc recovery step:
  - the old `codex exec --ephemeral` child could stall without emitting the required closeout JSON
  - the outer Python runner then lost its machine-readable control surface
  - the human operator had to manually recover the coherent partial diff on the main critical path
- Reframed `graft-multi-agent-loop` as a same-session main-agent delegation loop:
  - the main agent now owns budget tracking, stop conditions, validation planning, and final acceptance
  - delegated rounds still run through `graft-multi-agent-task` and close out through `graft-task-closeout`
  - the JSON closeout remains the only loop control surface, while `Next-session startup prompt:` stays a
    human-readable mirror for future-turn handoff
- Removed the prototype `run_loop.py` / `test_run_loop.py` implementation and cleaned live governance, skill text, and
  prompt references so active repository truth no longer describes `graft-multi-agent-loop` as a fresh-session script.
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
- Immediate next step after this slice:
  - continue the real Phase 3 ownership split by separating the shared Ent/schema and mixed Atlas history around the
    `users` / `refresh_sessions` vs `user_roles` / `rbac` boundary, now that the default migrate path no longer
    assumes every declared plugin migration directory is already active

## 2026-05-19 backend wiring-hardening follow-up

- Re-ran startup preflight on `refactor/server-module-boundaries`, classified the work as `server`, and recovered
  through the active `multi-worktree-governance` parent topic before executing an explicit commit/push request.
- Reviewed the dirty worktree and kept the owned scope narrow:
  - included the coherent backend hardening slice under `server/**`
  - updated the active topic tracking/trace files for recovery honesty
  - excluded the unrelated `.agents/skills/graft-task-closeout/SKILL.md` wording edit from commit scope
- Hardened the compile-time/runtime wiring edges without reopening centralized business-store coupling:
  - `server/plugins/{audit,rbac,user}/descriptor.go` now defines stable plugin IDs, versions, and dependencies
    directly instead of constructing runtime plugin instances for metadata
  - `server/internal/app/runtime.go` now fails explicitly when transitional repository providers are unavailable,
    and `server/internal/app/runtime_test.go` now resolves non-nil repository singletons in coverage
  - `server/internal/cli/dev_reset.go` now routes dev-only default-admin reset wiring through
    `user.NewAuthRepositoryForReset`, `user.NewRBACBootstrapServiceForReset`, and `pluginapi.RBACBootstrapService`
  - `server/plugins/rbac/bootstrap_service.go` and `server/plugins/rbac/storeadapter/internal_store.go` now return
    `nil` for nil inputs so transitional helpers fail cleanly instead of wrapping absent repositories
  - `server/internal/cli/migrate.go` now rejects a compile-time registry whose default migration chain has no
    Atlas-state directories, and `server/internal/cli/migrate_test.go` covers that failure mode
- Tightened regression coverage and test hygiene around the same slice:
  - removed the obsolete repository-shaped RBAC dev-reset stub after the reset path moved to the bootstrap capability
  - simplified the plugin dependency-cycle assertion to check the emitted error text directly
  - added ordering and determinism assertions to `pluginregistrygen` descriptor tests
- Validation for the slice finished with:
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend`
- Immediate next step after this slice:
  - continue the real Phase 3 ownership split by separating the shared Ent/schema and mixed Atlas history around the
    `users` / `refresh_sessions` vs `user_roles` / `rbac` boundary, now that builder/runtime seams fail earlier and
    the default migrate chain reports empty registry state explicitly

## 2026-05-19 server Phase 3c user-role reverse-edge narrowing

- Re-ran startup preflight on `refactor/server-module-boundaries`, classified the work as `server`, recovered through
  the active `multi-worktree-governance` parent topic, and executed the slice through `graft-multi-agent-loop` as one
  bounded delegated round with two read-only discovery sidecars.
- Kept the critical path local after the sidecars confirmed the same blocking constraints:
  - the live Ent generation path is still a single shared `server/internal/ent/generate.go`
  - the mixed Atlas revision `202605140001_auth_rbac_foundation.sql` remains immutable live history because
    `internal/ent/migrate/migrations/atlas.sum` already records it
  - RBAC still depends on shared generated `internal/ent/user` for `user_roles` existence checks
- Chose the smallest honest Phase 3 schema slice that reduces shared coupling without claiming the ownership split is
  complete:
  - removed the unused reverse `User -> user_roles` Ent edge from `server/internal/ent/schema/user.go`
  - rewired `server/internal/ent/schema/user_role.go` so the `user_id` foreign-key stays modeled as a one-way
    `UserRole -> User` Ent edge instead of an inverse edge that requires `User.user_roles`
  - regenerated shared `server/internal/ent/**` so the generated reverse-traversal helpers disappeared while the live
    shared `*ent.Client` runtime surface remained behavior-compatible
- Kept explicit non-goals for this round:
  - no plugin-owned `server/plugins/user/ent/**` generation path yet
  - no rewrite of the mixed Atlas revision chain
  - no `plugins/user/migrations/atlas.sum`
  - no attempt to remove RBAC's remaining direct shared-Ent `User` dependency yet
- Focused validation for the slice finished with:
  - `cd server && go test ./internal/store/entstore ./plugins/user ./plugins/rbac`
  - `cd server && go test ./internal/ent/...`
- Immediate next step after this slice:
  - continue Phase 3 by replacing RBAC's remaining shared `internal/ent/user` dependency around `user_roles` writes,
    then introduce the first plugin-owned `server/plugins/user/ent/**` generation path and forward-only user migration
    state without rewriting the historical mixed Atlas chain

## 2026-05-19 docs automation loop serial-subagent contract correction

- Re-ran startup preflight on `refactor/server-module-boundaries`, classified the work as `docs/automation`, and
  recovered through the active `multi-worktree-governance` parent topic before correcting the approved
  `graft-multi-agent-loop` contract.
- Corrected the active governance truth away from the still-conflicting "outer main agent keeps the implementation
  critical path local" wording:
  - root `AGENTS.md` now documents the explicit `graft-multi-agent-loop` exception under subagent rules
  - `graft-multi-agent-loop` now defines a same-session serial subagent orchestrator where the outer main agent owns
    orchestration, budget tracking, stop conditions, closeout parsing, acceptance, and next-round dispatch
  - each implementation round is now documented as delegated to exactly one worker subagent by default
  - the outer main agent is now explicitly forbidden from editing repo-tracked implementation files during active loop
    rounds
- Tightened the failure contract instead of leaving local recovery ambiguous:
  - missing, malformed, or contradictory worker closeout now retries once with a fresh worker subagent
  - the second closeout failure now fails closed as `blocked`
  - active governance no longer describes local main-agent recovery as a valid fallback for malformed round closeout
- Kept the loop correction constrained to governance/design/tracking only:
  - did not restore `run_loop.py`, `test_run_loop.py`, or `codex exec --ephemeral` style external fresh-session
    runners
  - did not change `graft-task-closeout` JSON fields
  - did not modify production runtime code, `server` code, `web` code, or CLI behavior
  - left the existing dirty `server/internal/ent/**` worktree changes untouched because they are outside this slice
- Consistency validation for the correction used targeted searches only:
  - `rg -n "critical path local|critical path in the main agent|same-session main-agent delegation loop|fresh-session|codex exec --ephemeral|run_loop.py|graft-multi-agent-loop|retry|blocked" AGENTS.md .agents/skills ai-plan/design/AI任务追踪与恢复设计.md ai-plan/public/multi-worktree-governance`
- Immediate next step after this slice:
  - when `graft-multi-agent-loop` is used again, the outer main agent should dispatch the first bounded round to one
    worker subagent, require the round closeout contract, retry once on malformed/missing closeout, and stop as
    `blocked` on the second failure instead of taking over implementation locally

## 2026-05-19 server Phase 3e RBAC plugin-local persistence slice

- Re-ran startup preflight on `refactor/server-module-boundaries`, classified the work as `server`, recovered through
  the active `multi-worktree-governance` parent topic, and executed this bounded round under
  `graft-multi-agent-loop` without `graft-multi-agent-batch` because the write scope stayed tightly coupled across
  descriptor wiring, dev CLI reset wiring, RBAC service semantics, and migration ownership.
- Moved the live RBAC persistence entrypoint onto the plugin-owned boundary:
  - added `server/plugins/rbac/storeent/**` as the live Ent-backed RBAC repository implementation
  - changed `server/plugins/rbac/descriptor.go` to resolve shared `*ent.Client` and build the plugin-owned RBAC
    repository directly
  - removed live runtime reliance on `server/plugins/rbac/storeadapter/internal_store.go`
- Finished the dev-only reset decoupling on the same boundary:
  - `server/internal/cli/dev_reset.go` now creates the RBAC bootstrap service from
    `rbac.NewRepositoryForReset(resources.Client)`
  - `server/plugins/rbac/dev_reset.go` now exposes the plugin-owned repository helper for this reset path
- Kept `user_roles -> rbac` ownership honest without expanding `pluginapi`:
  - `server/plugins/rbac/write_service.go` now checks target user existence via stable
    `pluginapi.UserService.GetUserByID` before user-role replacement writes
  - repository-local RBAC writes no longer depend on a shared-store adapter layer for user-not-found semantics
- Added the first honest RBAC migration-boundary checkpoint without rewriting mixed history:
  - `server/plugins/rbac/migrations/202605190002_rbac_plugin_boundary_checkpoint.sql`
  - `server/plugins/rbac/migrations/atlas.sum`
  - `server/plugins/rbac/migrations/README.md`
- Removed transitional dead paths once the live wiring no longer referenced them:
  - deleted `server/internal/store/entstore/rbac_repository.go`
  - deleted `server/internal/store/entstore/rbac_repository_test.go`
  - removed `store.Factory.RBAC()` and the runtime/container-visible `store.RBACRepository` singleton
- Validation for the slice finished with:
  - focused direct tests: `cd server && go test ./plugins/rbac ./plugins/rbac/storeent ./internal/app ./internal/cli ./internal/store/entstore ./internal/store/...`
  - backend lint gate execution slice: `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
- Immediate next step after this slice:
  - continue the ownership split by reducing or deleting the remaining shared `internal/store/**` surface for
    non-RBAC paths, while keeping future `user_roles` schema/migration truth inside `server/plugins/rbac/**`

## 2026-05-19 server Phase 3f RBAC compatibility cleanup slice

- Continued the same bounded `server` round under `graft-multi-agent-loop` without `graft-multi-agent-batch`; the
  remaining work was a tightly coupled compatibility cleanup across RBAC tests, user plugin embedded-RBAC tests, and
  active tracking truth.
- Removed the last runtime-dead shared RBAC compatibility layer:
  - deleted `server/plugins/rbac/storeadapter/internal_store.go`
  - deleted `server/internal/store/rbac.go`
- Moved the remaining test-only callers onto the plugin-local RBAC store boundary:
  - `server/plugins/rbac/plugin_test.go` now uses `server/plugins/rbac/store.Repository` test doubles directly
  - `server/plugins/user/plugin_test.go` and `server/plugins/user/dev_reset_test.go` now model embedded RBAC plugin
    behavior with `server/plugins/rbac/store/**` DTOs and inputs instead of shared-store RBAC types
- Kept active recovery truth honest:
  - updated `ai-plan/public/multi-worktree-governance/todos/multi-worktree-governance-tracking.md` to record that the
    test layer no longer depends on the temporary adapter or shared RBAC store file
- Validation for this cleanup slice finished with:
  - `cd server && go test ./plugins/rbac ./plugins/user`
  - `cd server && go test ./internal/store/...`
