# MVP Extension Path Tracking

## Topic

- Topic: `mvp-extension-path`
- Branch: `feat/mvp-extension-path`
- Scope: `server/core`, platform registries, initial plugins, and the `web` shell required by the MVP path

## Goal

- Keep one long-lived recovery entrypoint for the MVP extension path while the repository is still stabilizing its core
  architecture and implementation sequence.

## Repository Truth

- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/roadmap/MVP实施计划.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`

## Stages

- Stage A: core runtime
- Stage B: platform registries
- Stage C: initial plugins
- Stage D: web shell and dynamic menu path

## Current Recovery Point

- The repository AI workflow has been upgraded from `plan/` to `ai-plan/`.
- Repository-wide design truth now lives in `ai-plan/design/`.
- Repository-wide implementation sequencing now lives in `ai-plan/roadmap/`.
- The long-lived branch `feat/mvp-extension-path` has been created and is now the default execution branch for this
  topic.
- Repository-wide environment truth now lives under `.ai/environment/`, with scripts that regenerate raw and AI-facing
  inventories.
- This topic is the default recovery entrypoint for future MVP-path work.
- The repository now contains the first substantive MVP shell implementation across both `server` and `web`.
- `server` has a minimal runtime shell with explicit plugin registration, lifecycle ordering, registries, and a sample
  `user` plugin.
- `server` now uses an env-first configuration path with PostgreSQL and Redis as required core infrastructure.
- `server/go.mod` keeps direct dependencies documented and leaves indirect dependencies in standard Go tool format.
- `web` has a minimal Vue 3 + TDesign admin shell with `AuthLayout`, `BasicLayout`, static routing, mock auth, and a
  navigation store reserved for backend-driven menu metadata.
- Repository truth for the planned server-wide ORM migration now uses Ent as the backend ORM baseline.
- Repository truth for schema changes now uses Atlas versioned migrations executed through an explicit CLI step before
  application startup, not during runtime boot.
- Plugin-facing design truth now reserves a neutral repository / store factory boundary instead of exposing a concrete
  ORM handle through `plugin.Context` or cross-plugin public services.

## Active Risks

- Future work must keep repository-wide design truth and topic-level recovery documents aligned instead of creating a
  second source of truth.
- The main implementation risk is replacing existing runtime assumptions around startup-time migrations and direct DB
  handle access without leaking Ent-specific details across plugin boundaries.

## Latest Validation

- `rg -n -P "(?<!ai-)plan/" AGENTS.md README.md .gitignore .agents/skills -S`
- `rg -n "ai-plan/" AGENTS.md README.md .gitignore .agents/skills ai-plan -S`
- `bash scripts/collect-dev-environment.sh --check`
- `bash scripts/collect-dev-environment.sh --write`
- `python3 scripts/generate-ai-environment.py`
- `python3 -c 'import yaml; yaml.safe_load(open(".ai/environment/tools.raw.yaml", "r", encoding="utf-8")); yaml.safe_load(open(".ai/environment/tools.ai.yaml", "r", encoding="utf-8")); print("ok")'`
- `cmd.exe /C "cd /d F:\\gewuyou\\project\\go\\graft\\web && C:\\Users\\gewuyou\\.bun\\bin\\bun.exe install"`
- `cmd.exe /C "cd /d F:\\gewuyou\\project\\go\\graft\\web && C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run typecheck"`
- `cmd.exe /C "cd /d F:\\gewuyou\\project\\go\\graft\\web && C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run build"`
- `cd server && go mod tidy`
- `cd server && go list -m -u all`
- `cd server && go build ./cmd/graft`
- `cd server && go test ./...`
- `bash scripts/collect-dev-environment.sh --write`
- `python3 scripts/generate-ai-environment.py`
- `rm -rf web/dist`
- Documentation-only validation target for this update: owned `ai-plan/` files stay mutually consistent on Ent,
  Atlas versioned migrations, explicit migration CLI flow, and repository / store factory boundaries.

## Immediate Next Step

- Translate the updated repository truth into concrete `server` implementation work: define the Ent integration point,
  the Atlas CLI workflow, and the repository / store factory contract before changing runtime code.
