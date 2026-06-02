# Module-Oriented Modular Monolith Tracking

## Topic

- Topic: `module-oriented-modular-monolith`
- Status: `archive-ready`
- Branch: `feat/module-oriented-modular-monolith`
- Worktree: `feat/wt-audit-plugin-mvp`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
- Loop mode: `topic-completion-loop`
- Current batch: `Batch 4: archive-readiness check for this topic slice`

## Scope

- Owned scope:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/design/项目设计.md`
  - `ai-plan/design/模块与依赖注入设计.md`
  - `ai-plan/design/前端架构设计.md`
  - `server/internal/plugin/**`
  - `server/internal/pluginregistry/**`
  - `server/plugins/*/descriptor.go`
  - `ai-plan/public/module-oriented-modular-monolith/**`
  - `ai-plan/public/README.md`

## Current Recovery Point

- Current slice: final archive-readiness verification for the bounded wording-migration topic.
- Batch 1 completed the inventory/classification pass and is accepted input for final closeout.
- Batch 3 narrowed the remaining symbol work to explicit safety classes before any further identifier edits.
- Historical archived topics under this worktree remain archive evidence only and must not be reused as the current recovery entry.

## Batch Inventory

- Batch 1 findings accepted for this loop:
  - safe wording-only drift remains in authority docs such as `ai-plan/design/模块与依赖注入设计.md` and `ai-plan/design/项目设计.md`
  - doc-only frontend wording in `ai-plan/design/前端架构设计.md` is in scope only when it is not a canonical runtime symbol
  - symbol/type renames under `server/internal/plugin/**`, `server/internal/pluginregistry/**`, and descriptor-local symbol names are out of scope unless a comment or README wording-only edit is sufficient
  - physical path, import path, generator constant, migration path, and exported symbol renames remain blocked/high-risk for later batches
- Batch 3 classification:
  - safe local/private rename candidates:
    - `orderedModuleDescriptors`
    - `modulePackage`
    - `collectModulePackages`
    - descriptor-local `pluginID` / `pluginVersion` / `pluginDependencies`
  - deferred exported/public candidates:
    - `plugin.Plugin`
    - `plugin.Descriptor`
    - `RuntimeMetadata.OrderedPluginDescriptors`
    - `pluginregistry.BuildPlugins`
  - blocked path/generator/runtime candidates:
    - `pluginsDirName`
    - generated import alias suffix `plugin`
    - migration path strings under `plugins/<name>/migrations`
    - any package or physical directory rename

## Batch Acceptance Target

- repository governance docs describe backend `plugin` naming as historical module naming
- `server/internal/plugin/**` and `server/internal/pluginregistry/**` comments/README no longer imply a runtime plugin platform
- topic recovery files no longer claim `archive-ready`, branch-rename closeout, or `recovery source: none` when that conflicts with the active loop
- Batch 3 leaves a bounded deferral record for exported/path-like rename classes instead of silently normalizing them downstream
- Batch 4 confirms the remaining in-scope drift is either authority-preserving historical naming or a deferred new-topic-only rename class

## Validation Record

- required baseline:
  - `git diff --check`
- required backend completion chain when server authority files change:
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go test ./internal/plugin/... ./internal/pluginregistry/...`
  - `cd server && go build ./cmd/graft`
- required frontend completion entrypoint when the slice remains `cross-boundary`:
  - `cd web && bun run check`
- review aid:
  - consistency search across touched governance and design docs confirmed the new module-oriented narrative and historical plugin naming notes are present

## Remaining Loop Plan

- completed:
  - `Batch 1: inventory and classify remaining backend plugin semantics`
  - `Batch 2: complete safe server/docs wording migration`
  - `Batch 3: define bounded symbol migration plan`
  - `Batch 4: archive-readiness check for this topic slice`
- current:
  - none
- pending:
  - none

## Final Topic State

- Outcome: `archive-ready`
- Deferred follow-up: `module-symbol-and-path-authority-migration`
- Follow-up trigger:
  - only if the repository later decides to repair exported symbol names, package/path names, import paths, generator output naming, or runtime-facing historical plugin identifiers at the correct authority owner
- Continue policy:
  - do not reopen this loop for more wording migration
  - do not broaden into package/path renames or web implementation work from this recovery line
