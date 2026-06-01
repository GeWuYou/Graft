# Module-Oriented Modular Monolith Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md` and kept the slice classified as `cross-boundary`.
- Accepted Batch 1 inventory as the authority input for the active loop.
- Confirmed the current branch is already `feat/module-oriented-modular-monolith`; the remaining drift is wording-only, not a pending branch rename.
- Continued Batch 2 on the bounded server/docs wording migration while keeping symbols, paths, imports, and exported names stable.
- Updated recovery materials so they reflect the active topic-completion loop instead of an already closed branch-rename slice.
- Advanced to Batch 3 and classified the remaining symbol candidates by rename safety instead of assuming a broad wording migration can cross into exported API renames.
- Implemented only a local/private helper cleanup slice: unexported runtime metadata storage and package-private generator helper names now use module-first wording without changing package names, paths, or exported APIs.
- Ran the terminal Batch 4 archive-readiness audit across the owned governance, design, comment, and recovery surfaces.
- Confirmed the remaining in-scope `plugin`-named surfaces are either intentional historical authority-preserving names or deferred exported/path-like rename classes that require a separate future topic.
- Closed the topic as `archive-ready` and recorded a new-topic-only follow-up for exported symbol or path migration analysis.

## Authority Summary

- root `AGENTS.md` and `server/AGENTS.md` remain the execution-governance authority
- `ai-plan/design/项目设计.md` and `ai-plan/design/插件与依赖注入设计.md` remain the architecture authority
- `server/internal/plugin/**` and `server/internal/pluginregistry/**` remain the canonical code-comment surfaces for historical backend naming

## Validation Record

- Batch 4 executed:
  - `git diff --check`
  - `cd server && go test ./internal/plugin/... ./internal/pluginregistry/...`
  - consistency search across touched governance and design docs

## Batch 3 Classification

- safe local/private rename candidates:
  - `server/internal/plugin/runtime_metadata.go`
    - unexported field `orderedPluginDescriptors`
  - `server/internal/pluginregistry/cmd/pluginregistrygen/main.go`
    - package-private type `pluginPackage`
    - package-private helper `collectPluginPackages`
  - `server/plugins/*/descriptor.go`
    - file-local constants and vars such as `pluginID`, `pluginVersion`, `pluginDependencies`
- exported/public API candidates deferred:
  - `server/internal/plugin/plugin.go`
    - `type Plugin`
    - `type Descriptor`
  - `server/internal/plugin/runtime_metadata.go`
    - `OrderedPluginDescriptors`
  - `server/internal/pluginregistry/registry.go`
    - `BuildPlugins`
- blocked by path/generator/runtime authority:
  - `server/internal/pluginregistry/cmd/pluginregistrygen/main.go`
    - `pluginsDirName`
    - generated import alias suffix `plugin`
  - any physical path or package rename under `server/internal/plugin*` and `server/plugins/*`
  - migration path strings such as `plugins/<name>/migrations`

## Deferral Rationale

- Exported names remain part of the current compile-time/runtime authority surface and are referenced outside the immediate files; renaming them would broaden scope into cross-package API churn without a compatibility bridge, which this batch forbids.
- Path-like and generator constants remain tied to real directories and generated output shape, so changing them would become a physical-path or import-path migration rather than a bounded wording correction.
- Descriptor-local constants and vars are technically safe but were deferred in this batch because the smaller coherent cleanup slice was already satisfied by internal runtime/generator helper names; broadening across every descriptor file would add churn without changing the public authority stance.

## Closeout

- current loop state: `archive-ready`
- terminal batch: `Batch 4: archive-readiness check for this topic slice`
- final accepted outcome:
  - safe wording migration is complete in owned scope
  - Batch 3 kept exported names, path-like names, generator constants, and migration paths deferred
  - Batch 4 confirmed those deferred classes require a separate future topic instead of continuation inside this loop
- deferred future topic:
  - `module-symbol-and-path-authority-migration`
