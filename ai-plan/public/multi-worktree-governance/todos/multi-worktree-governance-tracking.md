# Multi Worktree Governance Tracking

## Topic

- Topic: `multi-worktree-governance`
- Branch: `refactor/server-module-boundaries`
- Worktree: repository root only; no dedicated long-lived worktree exists yet
- Scope: keep `mvp-extension-path` archived, reconcile recovery truth with the current root-branch reality, freeze the
  final post-compatibility `web` ownership model, add the matching `server` compile-time modular-monolith ownership
  baseline, and prepare stable owned scopes before creating dedicated long-lived worktrees.

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
- The repository is currently running from the root worktree on branch `refactor/server-module-boundaries`; `git worktree list`
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
  - `server` long-term governance stays on compile-time modular monolith:
    - no runtime plugin loading
    - no runtime plugin discovery
    - no hot-load lifecycle
    - no generalized reflection plugin system
    - no generalized service locator
    - keep single-process deterministic startup
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
- The current shell-owned follow-up slice on `2026-05-18` closes the previously recorded web hotspot gaps:
  - module registration now auto-discovers `modules/*/index.ts` and rejects duplicate `moduleId`, `menuPath`,
    stable `routeName`, and derived child route names
  - login route name/path truth now lives in platform auth-route contracts instead of scattered shell literals
  - locale aggregation now deep-merges module catalogs so future module-owned namespaces do not overwrite unrelated
    shell/module message trees at the top level
- The first expected future long-lived feature directions are:
  - `RBAC`
  - `server-status-dashboard`
- Those directions must not be registered as active topics until each one has:
  - a real local branch
  - a real long-lived worktree
  - a declared owned scope
  - a clear shared-hotspot integration path
- The current `2026-05-18` `server` governance slice freezes the intended backend ownership baseline in
  `ai-plan/public/multi-worktree-governance/roadmap/server-module-boundaries-plan.md`:
  - `shared-stable-boundary`: `server/internal/pluginapi/**`, `server/internal/contract/**`
  - `generated-shared-hotspot`: `server/internal/pluginregistry/generated.go`
  - `plugin-owned`: `server/plugins/<name>/**`
  - `core-owned`: runtime/infrastructure packages only
  - `internal/store/**` and `internal/ent/**` are no longer valid steady-state landing zones for new business logic
    once the corresponding plugin-owned boundary exists
- The current `2026-05-18` Phase 2a service-decoupling slice has now landed on the same branch:
  - `server/internal/pluginapi/rbac.go` defines stable `PermissionSeed`, `RBACAccessService`, and
    `RBACBootstrapService`
  - `server/plugins/rbac/**` now registers RBAC access/bootstrap capabilities and no longer reads user existence
    through `ctx.Stores.Users()`
  - `server/plugins/user/**` now consumes deferred RBAC capabilities for bootstrap reads and default-admin bootstrapping
    instead of direct runtime `RBACRepository` calls
  - the remaining backend merge hotspots are therefore narrowed further to shared contracts plus still-centralized
    `internal/store/**`, `internal/ent/**`, and migration ownership
- The current `2026-05-18` Phase 2b RBAC private-store contract slice has now landed on the same branch:
  - `server/plugins/rbac/store/**` now owns RBAC plugin-private repository DTOs, inputs, errors, and repository
    contract shape
  - `server/plugins/rbac/**` runtime code now depends on the local RBAC store contract instead of
    `server/internal/store/rbac.go`
  - `server/plugins/rbac/storeadapter/internal_store.go` is the temporary compatibility seam that adapts
    `ctx.Stores.RBAC()` into the plugin-local repository contract without reopening direct `user` / `rbac`
    repository coupling
  - `server/internal/pluginapi/user.go` now defines `pluginapi.ErrUserNotFound` as the shared cross-plugin
    not-found semantic used by RBAC user-role routes
  - the remaining backend hotspot for RBAC persistence is therefore narrowed to the temporary adapter plus the still
    centralized `internal/store/**` / `internal/store/entstore/**` implementation ownership that must be migrated in a
    later slice
- The current `2026-05-18` Phase 2c user private-store contract slice has now landed on the same branch:
  - `server/plugins/user/store/**` now owns the user plugin's private user/auth/session repository contract surface
  - `server/plugins/user/storeadapter/internal_store.go` is the temporary compatibility seam that adapts
    `ctx.Stores.Users()` and `ctx.Stores.Auth()` into that plugin-local contract without changing runtime behavior
  - `server/plugins/user/**` runtime code now depends on the local user store contract instead of directly importing
    `server/internal/store/{user,auth}.go`
  - the remaining direct `internal/store` dependency inside `server/plugins/user/**` is now limited to the
    dev-only RBAC bootstrap compatibility helper used by `ResetDefaultAdminForDevelopment`
  - the remaining backend hotspots are therefore narrowed further to shared contracts plus still-centralized
    `internal/store/**`, `internal/store/entstore/**`, `internal/ent/**`, and migration ownership
- The current `2026-05-18` readiness review for parallel `server` worktrees concludes:
  - Phase 1 is effectively complete: compile-time descriptors, generated registry, registry-driven `serve`, and
    registry-derived migration directory resolution are all live
  - Phase 2 is only partially complete: `rbac` and `user` now own private `store/**` contracts, but `plugin.Context`
    still exposes `Stores store.Factory`, runtime plugins still enter persistence through compatibility adapters, and
    `server/internal/cli/dev_reset.go` still coordinates reset behavior through centralized `internal/store/**`
  - Phase 3 has not started in earnest: business Ent schema, generated Ent code, and Atlas migration truth are still
    centralized under `server/internal/ent/**` and `internal/ent/migrate/migrations`
  - the branch is therefore suitable for limited plugin-feature parallelism on top of the current seams, but not yet
    for low-conflict long-lived parallel worktrees that modify plugin persistence, schema, or migrations independently
- The current `2026-05-18` Phase 2d builder-wiring slice has now landed on the same branch:
  - `plugin.Builder` now receives explicit `plugin.BuildContext`, and `app.NewRuntime()` now builds runtime plugins only
    after core services and `store.Factory` are available
  - `server/internal/cli/serve.go` no longer builds plugin instances directly; plugin construction now happens inside
    runtime assembly where core resources exist
  - `server/plugins/user/**`, `server/plugins/rbac/**`, and `server/plugins/audit/**` now receive plugin-private
    repository adapters during construction instead of reading repositories from `plugin.Context.Stores`
  - `plugin.Context` no longer exposes `Stores store.Factory`, so active plugin runtime paths cannot reopen the
    centralized business-store entrypoint by default
  - `server/plugins/user/dev_reset.go` and `server/internal/cli/dev_reset.go` now consume the stable
    `pluginapi.RBACBootstrapService` path instead of the removed repository-backed compatibility bootstrap seam
  - the remaining Phase 2 hotspot is therefore narrowed further to core still registering `store.Factory` for
    transitional service/container needs, while active plugin lifecycle code no longer consumes it directly
- The current `2026-05-18` Phase 2e explicit-repository builder slice has now landed on the same branch:
  - `plugin.BuildContext` no longer exposes `store.Factory`
  - `server/internal/app/runtime.go` now registers explicit `store.{Audit,User,Auth,RBAC}Repository` singletons
    instead of re-exporting the generalized `store.Factory` through the runtime container
  - `server/plugins/{audit,rbac,user}/descriptor.go` now resolve only the repository boundary each plugin needs during
    Builder wiring, then adapt that repository into the plugin-local store contract when needed
  - `store.Factory` remains an internal runtime helper for assembly and dev CLI flows, but it is no longer a
    container-visible or builder-visible general business-store backdoor
  - the remaining backend hotspot is therefore narrowed to the still-centralized `internal/store/**` /
    `internal/store/entstore/**` implementation ownership plus Phase 3 `internal/ent/**` and migration ownership
  - that earlier sequencing note only described the first extraction order; it is no longer the final ownership truth
    for `user_roles`
- The current `2026-05-18` Phase 3a user storeent extraction slice has now landed on the same branch:
  - `server/plugins/user/storeent/**` now owns the live Ent-backed user/auth/session repository implementation that the
    runtime `user` plugin consumes
  - `server/plugins/user/descriptor.go` now resolves the shared `*ent.Client` from the runtime container and builds
    plugin-owned `user` repositories directly, instead of adapting shared `internal/store` repositories into the
    plugin-local contract
  - `server/internal/app/runtime.go` now exposes the shared `*ent.Client` as an explicit builder-visible singleton so
    plugin-owned persistence implementations can be wired without reopening `store.Factory`
  - `server/internal/cli/dev_reset.go` now enters the user plugin through the plugin-owned auth repository contract for
    the dev-only default-admin reset path, while keeping the RBAC side transitional
  - `server/plugins/user/migrations/README.md` now freezes `plugins/user/migrations/**` as the plugin-owned migration
    boundary for future user-only versions, without rewriting the existing mixed Atlas history yet
  - the remaining Phase 3 blockers are now explicit:
    - `server/internal/ent/schema/{user,refresh_session,user_role,role}.go` still form one shared Ent graph
    - historical Atlas file `server/internal/ent/migrate/migrations/202605140001_auth_rbac_foundation.sql` still mixes
      `users`, `refresh_sessions`, `user_roles`, `roles`, `permissions`, and `role_permissions`
    - `user_roles` remains outside this first extraction and must stay out until the user/RBAC bridge is split safely
- The current `2026-05-18` Phase 3b migration-directory gating slice has now landed on the same branch:
  - `server/internal/cli/migrate.go` now treats compile-time registry migration directories differently from
    user-specified `--dir` input:
    - explicit `--dir` paths still run exactly as requested
    - the default registry-driven migration chain now skips plugin-owned directories that do not yet contain
      `atlas.sum`
  - this keeps `server/plugins/user/migrations/**` declared as the plugin-owned future boundary without pretending the
    directory already owns live Atlas history
  - the slice intentionally does not rewrite `202605140001_auth_rbac_foundation.sql`, does not generate
    `plugins/user/migrations/atlas.sum`, and does not claim the shared Ent graph around `user_roles` is already split
  - the remaining honest Phase 3 gap is therefore unchanged in structure but narrower operationally:
    - default migrate wiring no longer assumes every declared plugin migration directory is immediately runnable
    - the actual `users` / `refresh_sessions` vs `user_roles` / `rbac` ownership split still requires a later schema
      and Atlas-history migration slice
- The current `2026-05-19` backend wiring-hardening follow-up slice has now landed on the same branch:
  - `server/plugins/{audit,rbac,user}/descriptor.go` now declares stable compile-time plugin metadata directly instead
    of instantiating runtime plugin values just to derive IDs, versions, or dependencies
  - `server/internal/app/runtime.go` now rejects missing repository singletons explicitly when transitional
    `store.Factory` wiring is unavailable, preventing silent nil resolutions at runtime assembly boundaries
  - `server/internal/cli/dev_reset.go` now depends on plugin-root reset helpers plus
    `pluginapi.RBACBootstrapService`, keeping private adapter ownership inside `server/plugins/user/**`
  - `server/internal/cli/migrate.go` now fails fast when the compile-time registry produces no Atlas-state migration
    directories, instead of silently continuing with an empty automatic apply chain
  - focused tests now cover the hardened resolution and ordering paths in `internal/app`, `internal/cli`,
    `internal/plugin`, and `internal/pluginregistry`, while nil-safe helpers were added to the RBAC transitional
    bootstrap/adapter seams
- The current `2026-05-19` Phase 3c user-role reverse-edge narrowing slice has now landed on the same branch:
  - `server/internal/ent/schema/user.go` no longer declares the unused reverse `user_roles` edge from `User`
  - `server/internal/ent/schema/user_role.go` now keeps the `user_id` foreign-key relation as a one-way `UserRole ->
    User` Ent edge instead of depending on the removed `User.user_roles` back-reference
  - regenerated shared `server/internal/ent/**` code now drops the generated `User` API surface that only existed to
    support the reverse `user_roles` traversal, while keeping runtime behavior and the shared `*ent.Client` shape
    otherwise intact
  - this slice intentionally does not create `server/plugins/user/ent/**`, does not introduce plugin-owned user Ent
    generation yet, and does not rewrite the mixed Atlas history or generate `plugins/user/migrations/atlas.sum`
  - the remaining honest Phase 3 blockers are therefore narrower and more explicit:
    - `server/internal/store/entstore/rbac_repository.go` still imports shared generated `internal/ent/user` and still
      checks user existence through the shared Ent client for `user_roles` writes
    - `server/plugins/user/**` still consumes the shared generated `internal/ent/**` client because no plugin-owned
      `user/ent/**` generation path exists yet
    - historical Atlas file `server/internal/ent/migrate/migrations/202605140001_auth_rbac_foundation.sql` remains the
      immutable mixed history root for `users`, `refresh_sessions`, `user_roles`, `roles`, `permissions`, and
      `role_permissions`
- The current `2026-05-19` Phase 3d user plugin-owned Ent-path and migration-checkpoint slice has now landed on the same branch:
  - `server/internal/store/entstore/rbac_repository.go` no longer imports shared generated `internal/ent/user` for
    `user_roles` writes; user existence checks now narrow to direct `User.Get` lookups while preserving stable
    `store.ErrUserNotFound` semantics
  - `server/plugins/user/ent/**` now provides the first plugin-owned generated import surface for the user-owned Ent
    packages consumed by `server/plugins/user/storeent/**`
  - `server/plugins/user/storeent/**` now imports plugin-owned `plugins/user/ent/{user,refreshsession}` packages
    instead of directly importing shared generated `server/internal/ent/user` and
    `server/internal/ent/refreshsession`
  - `server/plugins/user/migrations/**` is now runnable with a forward-only no-op checkpoint
    `202605190001_user_plugin_boundary_checkpoint.sql` plus `atlas.sum`, without rewriting or deleting any historical
    mixed Atlas files under `server/internal/ent/migrate/migrations/**`
  - the remaining honest Phase 3 blockers are therefore narrowed again:
    - the runtime still uses the shared `server/internal/ent.Client` because introducing a separate plugin-owned Ent
      client would require out-of-scope runtime/container wiring changes
    - historical Atlas file `server/internal/ent/migrate/migrations/202605140001_auth_rbac_foundation.sql` remains the
      immutable mixed history root for `users`, `refresh_sessions`, `user_roles`, `roles`, `permissions`, and
      `role_permissions`
- The current `2026-05-19` Phase 3 owner-decision governance checkpoint supersedes the older shared-ownership wording:
  - `user_roles` now has one final owner: `rbac`
  - `rbac` owns the future `user_roles` Ent schema, repository, migration, and test truth
  - `user` does not own `user_roles` and does not expose role-assignment implementation details
  - `user` only exposes stable user capability surfaces needed by other plugins:
    - user existence checks
    - basic identity lookup
    - pre-delete constraint checks
  - `rbac` must validate `user_id` through those stable `user` capabilities or contracts, not by importing `user`
    plugin-private Ent packages
  - the historical mixed Atlas root remains immutable; the ownership checkpoint must be recorded later only through an
    `rbac` forward-only migration
  - multi-worktree ownership is now explicit:
    - `RBAC` worktree may modify `user_roles`
    - `User` worktree must not modify `user_roles` directly

## Shared Hotspots

- `server/internal/pluginapi/**`
- `server/internal/contract/**`
- `server/internal/pluginregistry/generated.go`
- `server/cmd/graft/**`
- `server/AGENTS.md`
- `AGENTS.md`
- `ai-plan/**`
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

- If a future `server` long-lived worktree still depends on shared edits in `internal/store/**`, `internal/ent/**`, or
  hand-written plugin registration lists, the first merge wave will recreate backend hotspot churn.
- If plugin dependency rules are not enforced, future plugins will quietly re-form source-level coupling through
  cross-plugin `service` / `storeent` / `ent/schema` imports.
- If migration ownership is not enforced, table ownership will drift back into shared migration hotspots.
- If `web` continues to let module-specific truth linger under root `api/model/contracts` surfaces, future worktrees
  will keep competing over files that are no longer valid steady-state ownership boundaries.
- If the recovery index is not refreshed when the repository root branch changes again, future boot/recovery flows will
  land on stale branch/worktree assumptions instead of current governance truth.
- If future looped multi-session automation relies on free-form closeout only, the runner will be brittle against
  wording drift and may continue or stop on the wrong round.

## Latest Validation

- Commit-scope diagnosis on `2026-05-18` confirmed one recurrent Git/IDE ambiguity:
  - `git status --short` showed the current `web` slice as ` M` on five files, which means modified but unstaged
  - `git diff --cached --name-only` was therefore correctly empty because the Git index had no staged entries
  - JetBrains changelist checkboxes or selected-file UI state must not be treated as Git staged proof in repository
    commit workflows; the `graft-commit` skill now calls this out explicitly
- Recovery truth was grounded against the current repository state with:
  - `pwd`
  - `git branch --show-current`
  - `git worktree list`
  - `git status --short`
- The current `server` governance baseline was grounded with:
  - `sed -n '1,320p' server/AGENTS.md`
  - `sed -n '1,260p' ai-plan/design/插件与依赖注入设计.md`
  - `sed -n '1,260p' ai-plan/design/项目设计.md`
  - `find server -maxdepth 3 -type d | sort`
  - `rg --files server | sort`
- Documentation consistency was checked with:
  - `rg -n "multi-worktree|worktree|兼容|compat|shared/|app/|modules/|refactor/web-module-boundaries|primary-main|main" ai-plan/public/multi-worktree-governance ai-plan/design/前端架构设计.md ai-plan/public/README.md`
- AGENTS split consistency was checked with:
  - `rg -n "web/AGENTS.md|server/AGENTS.md|Subdomain governance|执行真值|前端执行级治理真值|后端执行级治理真值" AGENTS.md web/AGENTS.md server/AGENTS.md ai-plan/design/AI任务追踪与恢复设计.md ai-plan/design/前端架构设计.md ai-plan/design/插件与依赖注入设计.md`
- The current `server` readiness review for multi-worktree execution was grounded with:
  - `sed -n '1,260p' server/internal/plugin/plugin.go`
  - `sed -n '1,260p' server/internal/cli/serve.go`
  - `sed -n '1,260p' server/internal/cli/migrate.go`
  - `sed -n '1,240p' server/internal/pluginregistry/registry.go`
  - `sed -n '1,240p' server/internal/pluginregistry/generated.go`
  - `sed -n '1,260p' server/internal/pluginapi/rbac.go`
  - `sed -n '1,260p' server/plugins/rbac/storeadapter/internal_store.go`
  - `sed -n '1,260p' server/plugins/user/storeadapter/internal_store.go`
  - `sed -n '1,220p' server/plugins/user/bootstrap_admin.go`
  - `find server/internal/ent -maxdepth 3 -type d | sort`
  - `find server/plugins -maxdepth 3 -type d | sort`
  - `rg -n 'ctx\\.Stores\\(|internal/store|internal/ent/migrate|internal/ent/schema|pluginregistry|Descriptor|Builder' server/internal server/plugins --glob '!server/internal/ent/**'`
- The current `server` Phase 2d builder-wiring slice was validated with:
  - `cd server && go test ./internal/plugin ./internal/pluginregistry/... ./internal/app ./internal/cli ./plugins/user ./plugins/rbac ./plugins/audit ./plugins/scheduler`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
- The current `server` Phase 2e explicit-repository builder slice was validated with:
  - `cd server && go test ./internal/plugin ./internal/pluginregistry/... ./internal/app ./plugins/user ./plugins/rbac ./plugins/audit`
- The current `server` Phase 3b migration-directory gating slice was validated with:
  - `cd server && go test ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
- The current `2026-05-19` backend wiring-hardening follow-up slice was validated with:
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend`
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
  - `cd web && bun run test:run -- src/permission.test.ts src/locales/index.test.ts src/modules/user/pages/index.test.ts`
  - `cd web && bun run typecheck`
  - `cd web && bun run check`
- This hotspot review rechecked the post-migration shell-owned surfaces with:
  - `git status --short`
  - `find web/src/app -maxdepth 3 -type f | sort`
  - `find web/src/locales -maxdepth 3 -type f | sort`
  - `find web/src/modules -maxdepth 3 -type f | sort | rg 'index\\.ts$|bootstrap-routes\\.ts$|locales/|contract/'`
  - `cd web && bun run test:run -- src/utils/route/bootstrap.test.ts src/utils/route/title.test.ts src/locales/index.test.ts`
  - `cd web && bun run typecheck`
- The follow-up shell-owned gap-closure slice revalidated with:
  - `rg -n "from '@/modules/[^']+/(types|api|pages|locales)|from \\\"@/modules/[^\\\"]+/(types|api|pages|locales)\\\"" web/src`
  - `rg -n "'/login'|\"/login\"|'login'|\"login\"" web/src --glob '!web/src/app/auth/index.vue' --glob '!web/src/locales/**'`
  - `cd web && bun run test:run -- src/modules/index.test.ts src/router/index.test.ts src/locales/index.test.ts src/permission.test.ts`
  - `cd web && bun run typecheck`
  - `cd web && bun run check`
- The current `server` Phase 1 registry-wiring slice revalidated the landed implementation with:
  - `cd server && go test ./internal/plugin ./internal/pluginregistry/... ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage buildtest`
- The current `server` runtime baseline now matches the Phase 1 ownership direction:
  - `serve` consumes compile-time generated plugin descriptors instead of a hand-written plugin list
  - `migrate` consumes registry-derived default migration directories instead of a hard-coded single core path
  - `server/internal/pluginregistry/generated.go` is now the only centralized plugin wiring hotspot
  - each current plugin exports its own `NewDescriptor()` shim under `server/plugins/<name>/descriptor.go`
- The current `server` Phase 2a service-decoupling slice revalidated the landed implementation with:
  - `cd server && go test ./plugins/rbac ./plugins/user ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend`
- The current `server` Phase 2b RBAC private-store contract slice revalidated the landed implementation with:
  - `cd server && go test ./plugins/rbac`
  - `cd server && go test ./plugins/rbac ./plugins/user ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend`
- The current `server` Phase 2c user private-store contract slice revalidated the landed implementation with:
  - `cd server && go test ./plugins/user ./internal/cli`
  - `cd server && go test ./plugins/rbac ./plugins/user ./internal/cli`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend`
- The current `2026-05-18` docs/automation loop-orchestrator slice was grounded with:
  - `sed -n '1,220p' .agents/skills/graft-multi-agent-task/SKILL.md`
  - `sed -n '1,260p' .agents/skills/graft-task-closeout/SKILL.md`
  - `sed -n '190,215p' AGENTS.md`
  - `sed -n '1,260p' ai-plan/design/AI任务追踪与恢复设计.md`
  - `python3 /root/.codex/skills/.system/skill-creator/scripts/init_skill.py --help`
  - `python3 /root/.codex/skills/.system/skill-creator/scripts/generate_openai_yaml.py --help`
- This slice adds a new repository skill plus local automation runner only; runtime validation expectations for actual
  `server` and `web` feature work remain unchanged.
- The current `2026-05-19` docs/automation loop contract-correction slice was grounded with:
  - `sed -n '1,260p' AGENTS.md`
  - `sed -n '1,260p' .agents/skills/graft-multi-agent-loop/SKILL.md`
  - `sed -n '1,220p' .agents/skills/graft-multi-agent-task/SKILL.md`
  - `sed -n '1,220p' .agents/skills/graft-boot/SKILL.md`
  - `sed -n '1,220p' .agents/skills/graft-multi-agent-batch/SKILL.md`
  - `sed -n '1,260p' ai-plan/design/AI任务追踪与恢复设计.md`
  - `rg -n "critical path local|critical path in the main agent|same-session main-agent delegation loop|fresh-session|codex exec --ephemeral|run_loop.py|graft-multi-agent-loop|retry|blocked" AGENTS.md .agents/skills ai-plan/design/AI任务追踪与恢复设计.md ai-plan/public/multi-worktree-governance`
- This slice changes governance text only; it does not restore any external fresh-session runner, does not change
  `graft-task-closeout` JSON fields, and does not modify production runtime code, `server` code, `web` code, or CLI behavior.

## Immediate Next Step

- Keep using `multi-worktree-governance` on the current root branch until the repository either returns to `main` with
  the same baseline or creates the first dedicated worktree/topic pair.
- Keep the landed `server` governance baseline as the default for future backend worktree ownership:
  - plugin-first owned scope under `server/plugins/<name>/**`
  - compile-time generated plugin registry as the only allowed central plugin wiring artifact
  - `internal/pluginapi/**` and `internal/contract/**` as the only stable shared backend API boundary
  - no new business logic backflow into `internal/store/**`, `internal/ent/**`, or core runtime packages
- Keep the landed module-boundary refactor as the baseline for future `web` worktree ownership:
  - preserve `web/src/modules/user/**` and `web/src/modules/rbac/**` as module-owned feature truth
  - preserve `web/src/app/**` and other shell-owned code as consumers of module registrations instead of holders of
    feature route truth
  - preserve `web/src/shared/**` as the only valid cross-module reusable asset layer
- Keep the current shell-owned baseline for future `web` worktree ownership:
  - auto-discovered module registration remains the only module-to-shell integration path
  - login route name/path remains a platform contract instead of a scattered shell literal
  - locale aggregation remains deep-merge based for module-owned catalogs
- Before creating the first additional worktree, decide the exact owned scope and shared-hotspot policy for:
  - `RBAC`
  - `server-status-dashboard`
- Once the first real worktree/topic pair exists, add it to `ai-plan/public/README.md` and create its dedicated
  tracking/trace files instead of continuing to stage feature recovery on the root branch.
- Use the new split governance layout as the baseline for future worktree setup:
  - root `AGENTS.md` remains the only startup-governance source
  - `web/AGENTS.md` and `server/AGENTS.md` own daily execution rules inside their boundaries
- Continue backend work from the landed Phase 1 baseline instead of reopening hand-written wiring:
  - preserve `plugin.Descriptor` / `plugin.Builder` as the only new plugin onboarding seam
  - keep `generated.go` as the sole centralized plugin-list artifact
  - continue Phase 2 from the landed service-capability seam instead of reintroducing direct user/rbac repository
    coupling
  - continue after the landed RBAC and user contract moves by migrating the remaining persistence implementation
    ownership out of `internal/store/**` / `internal/store/entstore/**`, starting with the remaining
    `rbac/storeent/**` extraction plus the `user_roles` ownership checkpoint that must land as `rbac`-owned
    `ent/schema` + forward-only Atlas migration truth
- Keep the landed docs/automation loop workflow aligned before relying on it for long-running repository work:
  - keep `graft-multi-agent-loop` as an outer wrapper only; do not let it redefine startup, closeout, or commit rules
  - keep `graft-multi-agent-task` and `graft-task-closeout` aligned on the dual-channel closeout contract
  - keep `graft-multi-agent-loop` documented as a same-session serial subagent orchestrator:
    - outer main agent owns orchestration, budget, stop conditions, closeout parsing, acceptance, and next-round dispatch
    - each implementation round is delegated to exactly one worker subagent by default
    - the outer main agent must not edit repo-tracked implementation files during active rounds
    - malformed or missing worker closeout follows `retry_once_then_blocked`
  - do not restore `run_loop.py`, `test_run_loop.py`, or `codex exec --ephemeral`-style external fresh-session runners

## Server Owned Scope Freeze

- Future `server` long-lived worktrees should own one plugin boundary at a time:
  - `server/plugins/user/**`
  - `server/plugins/rbac/**`
  - future `server/plugins/server-status/**` or equivalent plugin path
- `shared-stable-boundary` directories stay centrally integrated:
  - `server/internal/pluginapi/**`
  - `server/internal/contract/**`
- `generated-shared-hotspot` stays centrally integrated:
  - `server/internal/pluginregistry/generated.go`
- `core-owned` runtime assets stay centrally integrated and are not long-lived feature-owned scope:
  - `server/internal/app/**`
  - `server/internal/plugin/**`
  - `server/internal/httpx/**`
  - `server/internal/config/**`
  - `server/internal/logger/**`
  - `server/internal/database/**`
  - `server/internal/container/**`
  - `server/internal/eventbus/**`
  - `server/internal/menu/**`
  - `server/internal/permission/**`
  - `server/internal/cronx/**`
  - `server/internal/redisx/**`
  - `server/internal/migration/**`
  - `server/internal/ent/**` 仅限 core-owned schema
- `server/internal/store/**` 与 `server/internal/ent/**` 不是未来业务插件 steady-state owned scope，业务真相迁出后不得重新回流

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

## 2026-05-19 server Phase 3e RBAC plugin-local persistence slice

- `server/plugins/rbac/descriptor.go` now resolves the shared `*ent.Client` and builds plugin-owned
  `server/plugins/rbac/storeent/**` repositories directly instead of adapting `internal/store.RBACRepository`.
- live `server/plugins/rbac/**` runtime wiring and `server/internal/cli/dev_reset.go` no longer depend on
  `server/plugins/rbac/storeadapter/internal_store.go` or `internal/store/entstore.NewFactory(...).RBAC()`.
- `server/plugins/rbac/write_service.go` now performs `user_id` existence checks through stable
  `pluginapi.UserService.GetUserByID`, keeping the frozen `user_roles -> rbac` ownership truth without expanding
  `pluginapi`.
- `server/plugins/rbac/migrations/**` now contains a forward-only no-op checkpoint plus `atlas.sum`, making the
  plugin-owned RBAC migration directory runnable without pretending the mixed historical Atlas chain was rewritten.
- the transitional shared RBAC store implementation and `store.Factory.RBAC()` outlet have been removed because they
  became dead paths once runtime and dev-reset wiring moved to the plugin-owned boundary.
- `server/plugins/rbac/plugin_test.go`, `server/plugins/user/plugin_test.go`, and
  `server/plugins/user/dev_reset_test.go` now exercise the plugin through `server/plugins/rbac/store/**` directly,
  which made `server/plugins/rbac/storeadapter/internal_store.go` and `server/internal/store/rbac.go` removable.
