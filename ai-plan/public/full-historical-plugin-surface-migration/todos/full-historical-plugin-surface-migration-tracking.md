# Full Historical Plugin Surface Migration Tracking

## Topic

- Topic: `full-historical-plugin-surface-migration`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
- Parent topic: `server-module-semantics-and-live-migration-reset`
- Loop mode: `topic-completion-loop`
- Current batch: `none`

## Owned Scope

- `server/internal/pluginapi/**`
- `server/internal/moduleapi/**`
- `server/internal/module/**`
- `server/internal/moduleregistry/**`
- `server/internal/app/**`
- `server/internal/httpx/**`
- `server/internal/cli/**`
- `server/modules/**`
- `openapi/**`
- `server/internal/contract/openapi/**`
- generated Go/TS consumers coupled to the renamed authority
- `web/src/modules/**`
- `web/src/app/**`
- `web/src/router/**`
- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- active `ai-plan/design/**`
- active `ai-plan/public/**`

## Exclusions

- `ai-plan/public/archive/**`
- third-party/framework vocabulary such as Vue test `plugins` arrays
- external dependency/package names containing `plugin`
- unrelated product or architecture expansion beyond the rename authority chain

## Frozen Decisions

- Batch 1 is limited to recovery truth and topic docs
- prior “retained stable/public plugin authority” exceptions are removed for active repository-owned surfaces
- no compatibility bridge, fallback, alias, adapter, or dual field/path is allowed
- archive topics remain untouched
- OpenAPI and generated artifacts must migrate in the same topic after canonical source changes; they are not acceptable residual drift

## Rename Map

- `server/internal/pluginapi` -> `server/internal/moduleapi`
- exported/runtime `Plugin*` -> `Module*` where repository semantics mean module
- monitor payload `plugins` -> `modules`
- monitor summary `total_plugins` -> `total_modules`
- monitor summary `healthy_plugins` -> `healthy_modules`
- anomaly value `plugin_dependency_missing` -> `module_dependency_missing`
- scope kind `plugin` -> `module`
- visible copy `插件` / `Plugins` -> `模块` / `Modules`

## Batch State

- completed:
  - `Batch 1: topic open + authority freeze`
  - `Batch 2: backend stable package and exported-symbol rename`
  - `Batch 3: backend runtime/domain rename for monitor and logging semantics`
  - `Batch 4: OpenAPI source + generated artifact migration`
  - `Batch 5: frontend consumer and visible-copy migration`
  - `Batch 6: active governance/design doc reconciliation`
- current:
  - `none`
- pending:
  - none

## Validation Record

- Batch 1:
  - `git diff --check`
- Batch 2:
  - `cd server && go generate ./internal/moduleregistry`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - minimum justified `go test`
  - `cd server && go build ./cmd/graft`
- Batch 3:
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - minimum justified `go test`
  - `cd server && go build ./cmd/graft`
- Batch 4:
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd web && bun run openapi:types`
  - `cd web && bun run openapi:types:check`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go build ./cmd/graft`
- Batch 5:
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run check`
- Batch 6:
  - `git diff --check`
- Batch 7:
  - `git diff --check`
- Batch 8:
  - `git diff --check`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `rg -n "plugin|plugins|Plugin" server/internal server/modules ai-plan/design ai-plan/public AGENTS.md server/AGENTS.md web/AGENTS.md --glob '!ai-plan/public/archive/**'`

## Notes

- `module-physical-path-migration`, `module-historical-plugin-naming-migration`, and `server-module-semantics-and-live-migration-reset` remain archive-ready evidence only for their accepted narrower scopes.
- This topic exists because the repository now wants full migration of backend/public/shared/frontend repository-owned `plugin` authority with no compatibility exception.
- Future batches must keep authority-first repair order: canonical backend/package/runtime authority before downstream generated and frontend consumer repair.
- Batch 7 archive-readiness scan found remaining live drift in active `server` comments/log fields/test helpers, active generated/openapi package docs, and current governance/design docs that still present `plugin` wording as live repository authority.
- Batch 8 removed those residual live drifts and closed the loop as `archive-ready`.
