# Module Symbol And Path Authority Migration

## Status

- Topic: `module-symbol-and-path-authority-migration`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `module-oriented-modular-monolith`
- Loop mode: `topic-completion-loop`
- Current batch: `Batch 3: finalize topic docs and archive-readiness check`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/plugin/**` owns the historical compile-time module lifecycle package, exported symbol surface, and runtime metadata symbol authority currently expressed under `plugin` naming
  - `server/internal/pluginregistry/**` owns the generated registry, registry-facing builder/descriptors, generator constants, and migration directory aggregation authority
  - `server/plugins/*/descriptor.go` owns descriptor-local compile-time metadata symbol names for each backend business module
  - `server/internal/app/runtime.go` is the minimum additional runtime-consumer authority file required for truthful analysis because it consumes `pluginregistry.OrderedDescriptors`, `pluginregistry.BuildPlugins`, `plugin.NewRuntimeMetadata`, and `RuntimeMetadata.OrderedPluginDescriptors`
  - `ai-plan/design/项目设计.md` and `ai-plan/design/模块与依赖注入设计.md` remain the architecture narrative authority for deciding whether a symbol/path migration is still aligned with module-oriented modular monolith semantics
  - `ai-plan/public/module-symbol-and-path-authority-migration/**` owns recovery truth for this bounded topic

## Goal

Open a new bounded topic for module-symbol and path authority work without continuing the archived wording-only loop in place.

This topic must:

1. inventory exported symbol, package/path, import/generator, descriptor-local, and runtime-consumer rename classes
2. identify the minimum authority owners needed for each class
3. keep Batch 1 analysis-first and recovery-truthful
4. allow Batch 2 to take only tiny, authority-aligned, validated migration slices if they are proven safe

This topic must not:

- treat historical `plugin` wording cleanup as still active work
- introduce compatibility aliases, adapters, or fallback layers
- rename physical directories or import paths without explicit authority proof and validation
- broaden into `web` implementation work

## Scope

- included:
  - `server/internal/plugin/**`
  - `server/internal/pluginregistry/**`
  - `server/plugins/*/descriptor.go`
  - `server/internal/app/runtime.go`
  - `ai-plan/design/项目设计.md`
  - `ai-plan/design/模块与依赖注入设计.md`
  - `ai-plan/public/module-symbol-and-path-authority-migration/**`
  - `ai-plan/public/README.md`
- excluded unless later authority discovery proves they are required:
  - physical directory renames under `server/internal/plugin*` or `server/plugins/*`
  - import-path rewrites outside the owned scope
  - generated artifact shape changes that require new cross-boundary compatibility handling
  - `web/src/**` implementation changes

## Batch 1 Inventory

### Rename Classes

| Class | Current authority | Representative examples | Batch 1 verdict |
| --- | --- | --- | --- |
| exported symbol rename | `server/internal/plugin/**`, `server/internal/pluginregistry/**` | `plugin.Plugin`, `plugin.Descriptor`, `plugin.RuntimeMetadata.OrderedPluginDescriptors`, `pluginregistry.BuildPlugins` | defer; needs authority-aligned API migration analysis |
| package/path rename | `server/internal/plugin/**`, `server/internal/pluginregistry/**`, filesystem paths under `server/plugins/*` | package path `graft/server/internal/plugin`, directory `server/plugins/*` | defer; high-risk physical/import migration |
| import/generator rename | `server/internal/pluginregistry/cmd/pluginregistrygen/main.go`, `server/internal/pluginregistry/generated.go` | `pluginsDirName`, import alias suffix `plugin`, generated `[]plugin.Descriptor` | defer; generated output shape and directory truth are coupled |
| descriptor-local rename | `server/plugins/*/descriptor.go` | initially suspected candidates: `pluginID`, `pluginVersion`, `pluginDependencies` | Batch 2 validation disproved safety; these names are package-scoped and remain deferred |
| runtime-consumer rename | `server/internal/app/runtime.go` | `orderedDescriptors`, `pluginregistry.OrderedDescriptors`, `plugin.NewRuntimeMetadata`, `runtimeMetadata.OrderedPluginDescriptors()` | analyze first; runtime consumer names must follow the true exported authority |

### Minimum Additional Authority Files

- `server/internal/app/runtime.go`
  - required now because it is the bounded runtime consumer within owned scope that proves which exported registry/plugin names are actually consumed by runtime assembly
- none beyond `server/internal/app/runtime.go` were required for Batch 1
  - reason: current inventory could be completed from the owned plugin/pluginregistry/descriptor surfaces plus the direct runtime consumer edge, without touching OpenAPI, `web`, or unrelated server packages

## Loop State

- completed:
  - `Batch 1: inventory exported symbol and path authorities; create new bounded topic recovery materials`
  - `Batch 2: apply safe authority-aligned symbol/path migrations within bounded scope`
  - `Batch 3: finalize topic docs and archive-readiness check`
- current:
  - none
- pending:
  - none

## Final Topic State

- Outcome: `archive-ready`
- Archive-readiness basis:
  - Batch 1 established the bounded authority inventory and recovery materials for the new topic
  - Batch 2 directly tested the only pre-approved tiny candidate slice and proved it unsafe as a descriptor-local-only rename
  - no code rename landed, so the truthful end state is a docs-only authority conclusion rather than a partial migration
  - the remaining rename classes are now explicitly deferred and require a new bounded topic instead of further continuation inside this loop
- Deferred follow-up:
  - future work must open a new bounded topic if the repository later wants to pursue any of:
    - exported symbol renames
    - runtime-consumer renames
    - generator constant/output renames
    - package or physical path renames
    - migration string renames
    - package-scoped plugin-local constant renames across full plugin packages

## Validation Plan

- Batch 1 baseline:
  - `git diff --check`
- Batch 2 executed:
  - `git diff --check`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go test ./plugins/auth ./plugins/user ./plugins/rbac ./plugins/audit ./plugins/monitor`
  - `cd server && go build ./cmd/graft`
- Batch 2 result:
  - no safe descriptor-local-only rename landed after validation
  - the tested candidate names remain deferred because the direct package-scoped Go validation showed they are not descriptor-local-only
- conditional for future batches:
  - if Go files under `server/internal/plugin/**`, `server/internal/pluginregistry/**`, or `server/internal/app/runtime.go` change, keep the backend completion chain in order: `graft validate backend --stage lint`, minimum justified `go test`, then `go build ./cmd/graft`
  - if a batch changes shared contract consumers in `web`, add `cd web && bun run check` before claiming archive readiness

## Next-Session Prompt

`Re-run startup preflight from root AGENTS.md. Governance source: root AGENTS.md. Task class: cross-boundary. Recovery source: parent topic server-module-semantics-and-live-migration-reset. Owned scope: server/internal/module/**, server/internal/moduleregistry/**, server/internal/app/**, server/modules/**, AGENTS.md, server/AGENTS.md, web/AGENTS.md only for terminology/reference repair, active ai-plan/design/**, active ai-plan/public/**, and only the minimum additional authority files required by the next accepted batch. Treat module-symbol-and-path-authority-migration as archive-ready evidence only; do not resume this closed authority-inventory loop in place.`
