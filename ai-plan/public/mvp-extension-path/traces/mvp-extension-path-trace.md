# MVP Extension Path Trace

## 2026-05-12

- Established `mvp-extension-path` as the first long-lived active topic for Graft.
- Bound the topic to branch `feat/mvp-extension-path` so future MVP work has a stable recovery entrypoint.
- Migrated repository-wide design documents from `plan/` into `ai-plan/design/`.
- Migrated the MVP execution document from `plan/` into `ai-plan/roadmap/`.
- Added `ai-plan/design/AI任务追踪与恢复设计.md` to define the boundary between repository truth and topic recovery
  documents.
- Updated `AGENTS.md`, `README.md`, and `graft-boot` so boot and implementation rules now point at `ai-plan/`.
- Validation target for this change is documentation governance consistency rather than runtime compilation.

## 2026-05-12 planned ORM migration docs

- Updated repository design truth so the backend ORM baseline is now Ent instead of GORM.
- Updated repository design truth so schema changes use Atlas versioned migrations executed through an explicit CLI
  step before application startup, rather than through runtime startup execution.
- Narrowed the plugin-facing data boundary: `plugin.Context` and cross-plugin public services should expose a neutral
  repository / store factory contract instead of a concrete ORM handle.
- Kept this change documentation-only and limited it to repository truth plus active-topic recovery material.
- Current risk: follow-up `server` work may preserve old startup-time migration wiring or leak `*ent.Client` across
  plugin boundaries unless the implementation contract is tightened first.
- Validation target for this change is cross-document consistency across the owned `ai-plan/` files.

## 2026-05-12 Ent runtime migration

- Replaced the `server` database bootstrap from GORM to an Ent-backed client built on the pgx `database/sql` driver.
- Changed `server` database config from `GRAFT_DATABASE_DSN` to `GRAFT_DATABASE_URL` so runtime and Atlas CLI can share
  one PostgreSQL connection format.
- Removed `*gorm.DB` and the runtime migration registry from `plugin.Context`, and replaced the plugin-facing data
  boundary with a repository / store factory.
- Added a Cobra-based CLI entrypoint with explicit `graft migrate up` and `graft serve` commands.
- Added the initial Ent user schema, generated client code, Atlas-versioned SQL migration file, and `atlas.sum`
  baseline under `server/internal/ent/`.
- Updated the sample `user` plugin to resolve its data through the new repository boundary instead of returning a
  hard-coded shell payload.
- Direct validation completed for code generation, module tidy, `go build ./cmd/graft`, `go test ./...`, and CLI help
  output for both the root command and the `migrate` subtree.
- Remaining validation gap: Atlas apply was not executed against a live PostgreSQL database in this environment because
  the `atlas` CLI is not installed and no disposable database target was provisioned.

## 2026-05-13 PR review skill absorption

- Added a repository-local `graft-pr-review` skill under `.agents/skills/` so Graft can inspect the current branch PR
  through the GitHub API instead of depending on a GFramework-specific workflow.
- Absorbed the proven PR parsing logic from the external `gframework-pr-review` helper, but rewired the repository
  constants, environment-variable prefix, and skill metadata for `GeWuYou/Graft`.
- Simplified repository resolution so the helper uses normal `git` context by default and only falls back to explicit
  `GRAFT_GIT_DIR` and `GRAFT_WORK_TREE` bindings when needed.
- Registered the new skill in `AGENTS.md` so repository-maintained skill truth and the actual `.agents/skills/`
  contents stay aligned.
- Validated the helper locally with its Python regression tests and against the real public PR `GeWuYou/Graft#1`,
  including text output, JSON output, and `jq` narrowing of high-signal fields.

## 2026-05-13 CI and docs consistency fixes

- Removed invalid job-level `hashFiles()` gating from the pull request validation workflow so both smoke jobs rely on
  the workflow trigger configuration instead of unsupported runtime conditions.
- Updated repository-facing stack documentation from GORM to Ent where repository truth had already moved.
- Standardized the remaining trace wording from `end to end` to `end-to-end`.

## 2026-05-13 PR review follow-up fixes

- Fixed the sample `user` plugin route so backend access now requires an explicit MVP permission guard instead of
  relying on frontend route metadata alone.
- Added request-header-based session parsing in `server/internal/httpx` as a temporary server-side authorization gate
  until the real auth + RBAC plugins land.
- Updated the runtime shell so startup failures, normal shutdown, and plugin stop order all release plugin and core
  resources explicitly.
- Hardened `graft migrate up` so the default Atlas migration directory resolves from either the repository root or the
  `server` module root.
- Added focused tests for the new permission guard, reverse plugin shutdown ordering, and migration directory
  resolution.
- Direct validation completed with `cd server && go test ./...`.
- Remaining gap: the Atlas apply path still has not been exercised against a live PostgreSQL target, and the current
  permission gate is a deliberate MVP placeholder rather than the final RBAC implementation.

## 2026-05-13 PR review follow-up hardening

- Hardened `server/internal/container` singleton resolution so concurrent callers now share one in-flight provider
  build instead of racing to construct duplicate instances.
- Tightened `server/internal/httpx` shutdown sequencing so the runtime still waits for the server goroutine to finish
  even when graceful shutdown returns an error.
- Completed the remaining `server/internal/cli` review follow-ups by documenting `NewRootCommand` contract semantics
  and adding Go-style intent comments for the migration directory tests.
- Hardened the mock-backed `web` auth store to reject malformed persisted session payloads before they reach
  permission checks, and restricted unauthorized-page fallback navigation to safe in-app paths only.
- Fixed `graft-pr-review` helper gaps around explicit git bindings, optional GitHub token authentication, and grouped
  CodeRabbit path parsing so review extraction stays stable in mixed WSL/Windows and extensionless-file cases.
- Corrected the `AGENTS.md` environment inventory paths so repository startup guidance points at valid repo-relative
  files.
- Direct validation for this hardening batch includes `cd server && GOCACHE=/tmp/go-build-cache go test ./internal/container ./internal/httpx ./internal/cli`, `cd web && bun run typecheck`, and the helper's Python regression tests.

## 2026-05-13 PR review correctness follow-up

- Serialized `server/internal/httpx` lifecycle ownership with a mutex-guarded running-server slot so concurrent
  `Run` / `Shutdown` transitions no longer race on `s.server`.
- Added `server/internal/httpx/server_test.go` to lock the lifecycle contract down with direct coverage for
  concurrent start rejection and one-time detach semantics.
- Restored `graft-pr-review` default shell compatibility by preferring native `git` over the repository's Windows
  fallback unless an explicit override is configured.
- Tightened review-thread classification so visible `✅ Addressed in commit ...` markers close CodeRabbit threads and
  non-CodeRabbit threads without a reliable resolution signal stay conservative instead of being mislabeled as open.
- Fixed `--format json --json-output` so the helper still writes machine-readable JSON to stdout while also persisting
  the same payload to disk.
- Direct validation for this batch includes `cd server && go test -race ./internal/httpx`, the helper's Python
  regression tests, `fetch_current_pr_review.py --section pr` on the checked-out branch, and
  `fetch_current_pr_review.py --pr 1 --format json --json-output /tmp/graft-pr1-review.json`.

## 2026-05-13 PR review warning follow-up

- Fixed the remaining `graft-pr-review` warning gate so parsed `major` / `minor` / `duplicate` / `outside-diff`
  groups suppress the fallback "actionable comments block was not found" warning even when `nitpick` is empty.
- Added a focused `build_result()` regression test that covers a CodeRabbit latest-review body with only `major`
  grouped comments.

## 2026-05-13 comment governance baseline

- Added `ai-plan/design/代码注释与模块文档规范.md` as repository truth for Chinese comments, module navigation
  READMEs, comment priority ordering, and exemption boundaries.
- Updated `AGENTS.md` so repository execution rules now require Chinese documentation for hand-written Go comments,
  reject mechanical comments, and keep module `README.md` scoped to navigation rather than detailed design.
- Started the first implementation wave on `server/internal/container`, `server/internal/plugin`,
  `server/internal/httpx`, `server/internal/app`, and `server/plugins/user` so the new rules have concrete examples in
  the codebase instead of remaining documentation-only.
- Validation target for this wave is direct package-level `go test` coverage on the touched `server` packages plus
  consistency across the owned `ai-plan/` and `AGENTS.md` documents.

## 2026-05-13 comment governance wave 2

- Extended the comment-governance sweep from the initial lifecycle packages into more hand-written `server` modules:
  `config`, `database`, `menu`, `permission`, `cronx`, `store`, `pluginapi`, and `redisx`.
- Converted the remaining hand-written English package and exported-symbol comments in those modules into Chinese
  Go-style documentation, while preserving boundary semantics around core-owned resources, plugin contracts, and MVP
  repository access surfaces.
- Added Chinese test-intent comments to the touched `server` test files so validation targets and lifecycle
  assumptions remain readable during future recovery.
- Converted the current `web` shell's route/store/setup block comments to Chinese in the places where backend menu,
  permission, session, and shared-state contracts are intentionally staged ahead of dynamic plugin data.
- Direct validation target for this wave is the smallest compile-oriented check that covers the touched `server`
  packages plus `web` type checking, without widening into unrelated runtime integration work.

## 2026-05-12 `.ai/environment`

- Introduced `.ai/environment/tools.raw.yaml` and `.ai/environment/tools.ai.yaml` as repository-wide environment truth.
- Added `scripts/collect-dev-environment.sh` and `scripts/generate-ai-environment.py` so the inventory can be
  regenerated instead of hand-maintained.
- Updated `README.md`, `AGENTS.md`, `graft-boot`, and the AI governance docs so startup flow reads environment truth
  before making toolchain assumptions.
- Captured the current reality that `web` bootstrap files existed before the first substantive MVP shell work landed.

## 2026-05-12 MVP shell scaffold

- Confirmed that `https://tdesign.tencent.com/vue-next/getting-started` is the official TDesign Vue Next
  documentation, which matches the locked frontend stack in `AGENTS.md`.
- Classified the task as `cross-boundary` because the MVP extension path needed both a backend runtime shell and a
  frontend admin shell to become executable.
- Added a first-pass `server` runtime shell with explicit plugin ordering, registries, sample routes, lightweight DI,
  and a sample `user` plugin.
- Added a first-pass `web` admin shell with Vue 3, TypeScript, Vite, TDesign Vue Next, static auth, baseline layouts,
  and route/store scaffolding that preserves the future `menu + route + page + api + permission` contract.
- Installed Go 1.26.1 so the `server` shell could be validated locally.
- Switched frontend validation from the WSL Bun binary to the host Windows Bun at
  `C:\Users\gewuyou\.bun\bin\bun.exe`, which matches the user's WSL-based development setup.
- Fixed invalid UnoCSS package versions and a `vue-router` module augmentation bug so the frontend shell could pass
  typecheck and production build.
- Validated the current shell end-to-end with host Bun for `web` and Go toolchain validation for `server`.
- Fixed the environment regeneration workflow so both scripts resolve the repository root through Git and can regenerate
  `.ai/environment/` again after the shell bootstrap work.

## 2026-05-12 server env configuration

- Changed the planned server configuration path from YAML files to env-first loading with a committed
  `server/.env.example` template.
- Added PostgreSQL and Redis as required core infrastructure defaults for the server runtime, matching the platform
  stack while keeping business logic in plugins.
- Kept SQLite and PostgreSQL-compatible SQLite layers out of the runtime matrix; lightweight database alternatives
  remain a future testing convenience question rather than a production dependency choice.

## 2026-05-12 server dependency hygiene

- Updated flagged transitive server dependencies for `pgx`, `fsnotify`, `mapstructure`, and `locafero`.
- Kept `server/go.mod` readable by documenting only direct dependencies and leaving indirect dependencies in standard
  Go tool format.
- Treated this as dependency governance only; no architecture, plugin lifecycle, or frontend module convention changed.

## Next Step

- Run the first end-to-end Atlas migration against a disposable PostgreSQL database and replace the temporary
  header-based authorization gate with the real auth + RBAC plugin chain.

## 2026-05-13 local startup ergonomics

- Confirmed a real local startup failure mode: IDEs that run `cmd/graft` without subcommands only print root help,
  which looks like a silent startup failure even though the process exits normally.
- Added a regression test that locks in repository-root loading of `server/.env`, so the supported local config path is
  covered directly by tests.
- Clarified the root CLI help and repository README so local startup now points contributors to the explicit
  `graft migrate up` then `graft serve` flow.
- Added a minimal `scripts/dev-server.sh` helper that runs migration first and then starts the server, while failing
  early when `atlas` is missing.

## 2026-05-13 local startup CLI unification

- Replaced the Bash-owned startup flow with a first-class `graft dev` command that runs explicit migrations before
  server startup.
- Kept `graft serve` as the pure runtime entrypoint and `graft migrate up` as the standalone migration entrypoint, so
  schema changes remain explicit instead of being hidden inside normal boot.
- Updated the Atlas lookup failure path to explain that `graft dev` and `graft migrate up` require Atlas, while
  `graft serve` is only safe when the schema is already current.
- Reduced `scripts/dev-server.sh` to a compatibility wrapper that forwards to `go run ./cmd/graft dev`, so Windows and
  IDE users no longer depend on Bash logic for the actual startup sequence.
- Updated the repository README and active-topic tracking so local development now centers on one IDE-friendly command.

## 2026-05-13 remove startup wrapper

- Removed `scripts/dev-server.sh` after the Go CLI entrypoint became the only supported local startup path.
- Updated the README so Windows PowerShell and CMD users can start the backend directly with `go run ./cmd/graft dev`
  or a prebuilt `graft.exe dev` binary.

## 2026-05-13 logger and i18n extension hooks

- Added a first-class `server/internal/logger` module so the runtime now constructs one shared Zap logger instead of
  leaving logging as a config-only placeholder.
- Added a first-class `server/internal/i18n` module so the runtime can resolve locales, localize platform error
  messages, and fall back to `zh-CN` by default.
- Extended `plugin.Context` and the core service container with shared logger and i18n access, keeping those
  capabilities in core rather than letting plugins create incompatible side paths.
- Reserved a stable HTTP error contract with `message_key`, localized `message`, and `locale`, and updated the sample
  `user` plugin plus permission middleware to use it.
- Updated repository design truth so both `server` and `web` explicitly reserve i18n extension points in the MVP
  shell, while still treating full translation coverage as follow-up work.
- Validation target for this slice is direct compile-oriented coverage across touched `server` packages plus `web`
  typecheck/build, without claiming live PostgreSQL or Atlas execution.

## 2026-05-13 comment governance sweep completion

- Extended the hand-written `server` Go comment-governance sweep through the remaining CLI, runtime shell, registry,
  repository, resource-bootstrap, and sample plugin packages.
- Kept the final rule set conservative: exported Go symbols require Chinese Go doc comments, while complex
  orchestration or lifecycle functions use `参数：` / `返回值：` sections only when the signature alone is not enough.
- Updated `AGENTS.md` and `ai-plan/design/代码注释与模块文档规范.md` so repository truth now explains when to use the
  `server/internal/cli/dev.go` style and when to prefer shorter responsibility-and-boundary comments.
- Used a bounded multi-agent wave to split comment work across disjoint `server` package sets while keeping docs,
  validation, and final integration on the main agent.
- Direct validation for this sweep is `cd server && go test ./internal/app ./internal/cli ./internal/config ./internal/container ./internal/cronx ./internal/database ./internal/httpx ./internal/menu ./internal/permission ./internal/plugin ./internal/pluginapi ./internal/redisx ./internal/store ./plugins/user && go build ./cmd/graft`.

## Next Step

- Exercise `graft dev` against a disposable PostgreSQL database with a real Atlas installation, then replace the
  temporary header-based authorization gate with the real auth + RBAC plugin chain.
