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
- Validated the current shell end to end with host Bun for `web` and Go toolchain validation for `server`.
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

- Define the concrete `server` migration plan for Ent integration, Atlas CLI execution, and the repository / store
  factory contract before touching runtime implementation.
