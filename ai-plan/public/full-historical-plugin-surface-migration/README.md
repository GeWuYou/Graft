# Full Historical Plugin Surface Migration

## Status

- Topic: `full-historical-plugin-surface-migration`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `server-module-semantics-and-live-migration-reset`
- Loop mode: `topic-completion-loop`
- Current batch: `none`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/pluginapi/**` is no longer treated as an intentionally retained stable boundary; the canonical target is `server/internal/moduleapi/**`
  - `server/internal/**` owns remaining backend runtime, exported-symbol, log field, component-name, and monitor-domain historical `plugin` semantics
  - `openapi/**` and generated Go/TS consumers own the shared wire/domain rename once canonical source changes
  - `web/src/modules/**`, `web/src/app/**`, and `web/src/router/**` own the remaining frontend consumer and visible-copy migration for repository-owned `plugin` wording
  - active `ai-plan/design/**`, active `ai-plan/public/**`, and repository `AGENTS.md` files own current governance and recovery truth for the migration

## Goal

Open a new bounded active topic for the full historical `plugin -> module` migration across repository-owned current-authority surfaces.

This topic must:

1. remove the prior exception that allowed retained shared/public `plugin` authority
2. freeze the exact rename map, scope, exclusions, validation chain, and batch plan for the full migration
3. keep Batch 1 limited to recovery materials and authority reset
4. migrate backend, OpenAPI, generated consumers, frontend visible copy, and active governance docs without compatibility layers

This topic must not:

- edit archive topics
- start code or contract renames in Batch 1
- add alias, adapter, fallback, dual field, or compatibility DTO behavior
- preserve historical `plugin` wording in active repository-owned surfaces just because it was previously marked stable

## Locked Decisions

- package/path rename target:
  - `server/internal/pluginapi` -> `server/internal/moduleapi`
- backend runtime/exported naming target:
  - repository-owned `Plugin*` runtime or lifecycle names that mean module must migrate to `Module*`
- monitor/shared contract rename targets:
  - `plugins` -> `modules`
  - `total_plugins` -> `total_modules`
  - `healthy_plugins` -> `healthy_modules`
  - `plugin_dependency_missing` -> `module_dependency_missing`
  - scope kind `plugin` -> `module`
- visible copy target:
  - `插件` / `Plugins` -> `模块` / `Modules`
- compatibility policy:
  - no alias, fallback, adapter, bridge, or dual-write/dual-read path is allowed
- archive policy:
  - `ai-plan/public/archive/**` remains untouched historical evidence

## Scope

- included:
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
  - root `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - active `ai-plan/design/**`
  - active `ai-plan/public/**`
- excluded:
  - `ai-plan/public/archive/**`
  - third-party/framework vocabulary such as Vue test `plugins` arrays or dependency package names
  - unrelated product expansion outside the rename authority chain

## Validation Chain

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
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
- Batch 6:
  - `git diff --check`
  - rerun the still-applicable server/web validation required by the accepted delta before archive readiness is claimed
- Batch 7:
  - `git diff --check`
  - rerun the last still-applicable server/web validation set before closeout
- Batch 8:
  - `git diff --check`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - rerun the non-archive residual scan and accept only allowed historical/third-party hits

## Batch Plan

- Batch 1: topic open + authority freeze
  - scope: `ai-plan/public/full-historical-plugin-surface-migration/**`, `ai-plan/public/README.md` only if needed
  - focus: freeze rename map, authority owners, exclusions, validation chain, and loop state
- Batch 2: backend stable package and exported-symbol rename
  - focus: `pluginapi -> moduleapi`, import repair, exported/runtime `Plugin* -> Module*`, registry generation alignment
- Batch 3: backend runtime/domain rename for monitor and logging semantics
  - focus: repository-owned runtime/domain/log naming that still uses `plugin` while meaning module
- Batch 4: OpenAPI source + generated artifact migration
  - focus: shared wire/domain rename and generated Go/TS consumer regeneration
- Batch 5: frontend consumer and visible-copy migration
  - focus: frontend field consumption, visible copy, and monitor/audit/RBAC downstream repair
- Batch 6: active governance/design doc reconciliation
  - focus: active docs and AGENTS truth stop presenting retained `plugin` authority as current acceptable wording
- Batch 7: archive-readiness scan and closeout
  - focus: verify remaining `plugin` hits are limited to archive/history or third-party vocabulary only
- Batch 8: residual live plugin drift cleanup
  - focus: finish remaining log-field/comment/test-helper/doc cleanup, then rerun archive-readiness scan

## Loop State

- completed batches:
  - `Batch 1: topic open + authority freeze`
  - `Batch 2: backend stable package and exported-symbol rename`
  - `Batch 3: backend runtime/domain rename for monitor and logging semantics`
  - `Batch 4: OpenAPI source + generated artifact migration`
  - `Batch 5: frontend consumer and visible-copy migration`
- `Batch 6: active governance/design doc reconciliation`
- `Batch 7: archive-readiness scan and closeout`
- `Batch 8: residual live plugin drift cleanup`
- current batch:
  - `none`
- pending batches:
  - none

## Progress Record

- Batch 1:
  - opened `full-historical-plugin-surface-migration` as a new active bounded topic for the full removal of repository-owned historical `plugin` authority
  - froze the rename map, exclusions, validation chain, and seven-batch loop
- Batch 2:
  - migrated backend stable package authority from `internal/pluginapi` to `internal/moduleapi`
  - repaired imports and repository-owned `Plugin*` runtime/lifecycle naming in bounded backend scope
- Batch 3:
  - repaired backend runtime/domain/log naming that still used `plugin` while meaning module
- Batch 4:
  - migrated OpenAPI source plus generated Go/TS artifacts from repository-owned `plugin` wire semantics to `module`
- Batch 5:
  - migrated frontend consumers, locale keys, visible copy, and downstream tests from repository-owned `plugin` semantics to `module`
- Batch 6:
  - reconciled active governance/design/recovery docs so current-authority files stop treating retained `plugin` authority as acceptable live truth
- Batch 7:
  - ran the final non-archive scan across active governance/docs plus live `server` / `web` / `openapi` surfaces
  - found remaining repository-owned live drift, so the topic is not yet `archive-ready`
  - identified one additional bounded follow-up batch for residual log-field/comment/doc cleanup rather than falsely closing the loop
- Batch 8:
  - removed the remaining repository-owned live `plugin` drift from active non-archive server/runtime/doc surfaces where `plugin` still meant current `module` semantics
  - updated topic/recovery truth and reran archive-readiness validation
  - confirmed remaining `plugin` hits are limited to allowed classes: historical evidence, anti-goal wording, third-party/framework vocabulary, and archived/trace materials

## Batch 7 Archive-Readiness Result

- status: `continue-required`
- conclusion:
  - archive readiness failed because active non-archive surfaces still contain repository-owned `plugin` terminology outside approved residual classes
- approved residual classes:
  - archive/history/traces
  - third-party/framework vocabulary such as Vue test `plugins` arrays
  - external dependency or package names containing `plugin`
- remaining live drift found in active scope:
  - `server/modules/**`
    - structured log fields still use `zap.String("plugin", ...)` in active runtime code
    - helper/test names such as `assertUserModuleRegistry`
    - package comments and migration READMEs still describe module-owned Ent/runtime surfaces as `plugin`
  - `server/internal/**`
    - scheduler/logging examples and generated-contract package docs still describe current runtime/module semantics as `plugin`
  - active `ai-plan/design/**`
    - several current governance/design docs still point to `server/modules/**` predecessors as live authority instead of historical evidence only
  - `ai-plan/public/README.md`
    - active recovery entry still says the topic is at Batch 6 and does not reflect the Batch 7 residual-drift finding

## Batch 8 Archive-Readiness Result

- status: `archive-ready`
- conclusion:
  - active non-archive repository-owned current-authority surfaces no longer use `plugin` as the live term for current `module` semantics
- remaining allowed residual classes:
  - historical naming explanations and rename-history evidence
  - anti-goal or prohibited-architecture wording such as `runtime plugin platform`
  - third-party/framework vocabulary such as Vue test `plugins` arrays or package names like `eslint-plugin-unused-imports`
  - archive/history/trace materials and explicit historical owner references

## Next Batch

- none; the bounded loop is complete and this topic is `archive-ready`

Next-session startup prompt: `Re-run startup preflight from root AGENTS.md. Governance source: root AGENTS.md. Task class: cross-boundary. Recovery source: parent topic server-module-semantics-and-live-migration-reset. Owned scope: treat ai-plan/public/full-historical-plugin-surface-migration/** as archive-ready evidence only and open a new bounded topic only if the repository explicitly decides to rename currently allowed historical/plugin-literal surfaces outside this loop's accepted residual classes.`
