# Server Module Semantics And Live Migration Reset Tracking

## Topic

- Topic: `server-module-semantics-and-live-migration-reset`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
- Parent topic: `module-historical-plugin-naming-migration`
- Loop mode: `topic-completion-loop`
- Current batch: `none`

## Owned Scope

- `server/internal/module/**`
- `server/internal/moduleregistry/**`
- `server/internal/app/**`
- `server/modules/**`
- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md` only for terminology/reference repair
- active `ai-plan/design/**`
- active `ai-plan/public/**`

## Exclusions

- `ai-plan/public/archive/**`
- `openapi/**`
- `server/internal/contract/openapi/**`
- generated files
- shared wire/domain contract values such as `plugin_disabled`, `scopeKindPlugin`, and `PluginDependencyMissing`

## Frozen Decisions

- Batch 1 is limited to recovery truth and topic docs
- destructive rewrite of live module migrations is approved for a later batch because the user will delete the local database and rerun from scratch
- no compatibility bridge, fallback, alias, or adapter is allowed for active semantics cleanup by default
- older active topics remain archive-ready evidence only for their accepted scope; they do not prove current active semantics or live migration cleanup is complete

## Batch State

- completed:
  - `Batch 1: authority reset and inventory freeze`
  - `Batch 2: active design and governance authority cleanup`
  - `Batch 3: server internal non-wire naming cleanup`
  - `Batch 4: destructive live module migration reset`
  - `Batch 5: active recovery topic reconciliation`
  - `Batch 6: archive-readiness check and closeout`
- current:
  - none
- pending:
  - none

## Validation Record

- Batch 1:
  - `git diff --check`
- Batch 2:
  - `git diff --check`
  - active non-archive design-path rewiring searches
- Batch 3:
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go test ./internal/module ./internal/moduleregistry ./internal/app ./modules/auth ./modules/user ./modules/rbac ./modules/audit ./modules/monitor ./modules/scheduler`
  - `cd server && go build ./cmd/graft`
  - `git diff --check`
- Batch 4:
  - `atlas migrate hash --dir file://server/modules/user/migrations`
  - `atlas migrate hash --dir file://server/modules/rbac/migrations`
  - `atlas migrate hash --dir file://server/modules/audit/migrations`
  - `cd server && go run ./cmd/graft migrate up`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go test ./internal/cli ./modules/user ./modules/rbac ./modules/audit`
  - `cd server && go build ./cmd/graft`
  - `cd server && go run ./cmd/graft validate smoke`
    - counted as passed based on the later successful rerun after Redis recovery recorded on the main thread
- Batch 5:
  - `git diff --check`
  - active non-archive recovery/doc residual searches
- Batch 6:
  - `git diff --check`
  - targeted residual searches across `server` and active non-archive docs

## Notes

- `module-oriented-modular-monolith`, `module-symbol-and-path-authority-migration`, and `module-historical-plugin-naming-migration` remain useful archive-ready evidence for their accepted slices only.
- This topic exists because active docs, active recovery wording, and live module migration semantics still contain current-authority drift beyond those earlier bounded topics.
- Archive topics remain untouched in this loop unless a future batch explicitly changes only active index wording that references them.
- This topic is archive-ready after Batch 6; any future rename of shared/public `plugin` authority surfaces must open a new bounded topic rather than reopen this closeout.
