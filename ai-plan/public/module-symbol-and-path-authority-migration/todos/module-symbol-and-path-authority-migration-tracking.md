# Module Symbol And Path Authority Migration Tracking

## Topic

- Topic: `module-symbol-and-path-authority-migration`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
- Parent topic: `module-oriented-modular-monolith`
- Loop mode: `topic-completion-loop`
- Current batch: `Batch 3: finalize topic docs and archive-readiness check`

## Owned Scope

- `server/internal/plugin/**`
- `server/internal/pluginregistry/**`
- `server/plugins/*/descriptor.go`
- `server/internal/app/runtime.go`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/public/module-symbol-and-path-authority-migration/**`
- `ai-plan/public/README.md`

## Batch State

- completed:
  - `Batch 1: inventory exported symbol and path authorities; create new bounded topic recovery materials`
  - `Batch 2: apply safe authority-aligned symbol/path migrations within bounded scope`
  - `Batch 3: finalize topic docs and archive-readiness check`
- current:
  - none
- pending:
  - none

## Inventory Checklist

- exported symbol authority
  - `plugin.Plugin`
  - `plugin.Descriptor`
  - `RuntimeMetadata.OrderedPluginDescriptors`
  - `pluginregistry.BuildPlugins`
- package/path authority
  - `server/internal/plugin`
  - `server/internal/pluginregistry`
  - `server/plugins/<name>`
  - `plugins/<name>/migrations`
- import/generator authority
  - `pluginsDirName`
  - generated import alias suffix `plugin`
  - generated `[]plugin.Descriptor`
- descriptor-local authority
  - descriptor-local `plugin*` symbol names in `server/plugins/*/descriptor.go`
- runtime-consumer authority
  - `server/internal/app/runtime.go`

## Batch 2 Entry Rules

- prefer descriptor-local or similarly tiny migrations first
- direct validation disproved the initial `pluginID` / `pluginVersion` / `pluginDependencies` candidate set in the five target files; they are package-scoped within each plugin package
- do not rename exported symbols unless the slice proves:
  - the authority owner is within owned scope
  - the runtime consumer chain inside owned scope can be updated in the same slice
  - direct Go validation covers the touched authority
- do not rename package paths, physical directories, or migration directory strings in Batch 2 unless a narrower safe slice becomes explicit

## Validation Record

- Batch 1:
  - `git diff --check`
- Batch 2:
  - `git diff --check`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go test ./plugins/auth ./plugins/user ./plugins/rbac ./plugins/audit ./plugins/monitor`
  - `cd server && go build ./cmd/graft`
- Batch 2 landed state:
  - docs plus authority-analysis closeout
  - no safe descriptor-local-only code rename accepted from the attempted candidate list; runtime-linked symbol changes such as `moduleID` / `NewModuleSpec` had already landed in concrete descriptor files and were not reopened here
- future conditional:
  - keep `graft validate backend --stage lint`, minimum justified `go test`, and `go build ./cmd/graft` aligned with any future Go authority edits

## Notes

- `module-oriented-modular-monolith` remains archive-ready evidence only
- this topic exists specifically to avoid continuing the wording-migration loop in place
- Batch 2 intentionally left exported symbols, runtime-consumer names, generator constants, package paths, physical directories, import paths, migration strings, and the package-scoped plugin-local constants deferred
- the topic is now archive-ready because no additional bounded safe batch remains without opening a new rename-class topic
