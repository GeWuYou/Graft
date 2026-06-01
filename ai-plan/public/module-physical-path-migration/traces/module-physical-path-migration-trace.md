# Module Physical Path Migration Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md`.
- Kept the task classified as `cross-boundary`.
- Read root `AGENTS.md`, `.ai/environment/tools.ai.yaml`, `server/AGENTS.md`, `web/AGENTS.md`, `ai-plan/public/README.md`, parent topic recovery docs, and the relevant design authorities before writing recovery materials.
- Opened a new bounded topic at `ai-plan/public/module-physical-path-migration/**`.
- Froze the canonical physical rename map for backend module path migration.
- Completed the accepted physical move to `server/internal/module/**`, `server/internal/moduleregistry/**`, and `server/modules/**`.
- Recorded the remaining Batch 4 authority cleanup focus for package docs, governance truth, and path-string wording after the move.

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/module/**` and `server/internal/moduleregistry/**` are the current backend path authority after the accepted physical move
  - `server/internal/app/runtime.go` is the minimum runtime-consumer authority file already identified by the parent topic for registry/module path consumers
  - `server/modules/**` is the current business-module physical directory authority after the accepted move
  - `ai-plan/design/项目设计.md` and `ai-plan/design/插件与依赖注入设计.md` remain the architecture authority for why `module` is canonical and `plugin/plugins` is historical naming
  - `ai-plan/public/module-physical-path-migration/**` now owns truthful recovery for this specific physical rename topic

## Why The Parent Topic Was Not Resumed

- `module-symbol-and-path-authority-migration` is already `archive-ready`.
- Its final topic state explicitly lists physical directories, package paths, import paths, generator-coupled path truth, and migration strings as deferred follow-up requiring a new bounded topic.
- Resuming the parent in place would falsify recovery state by pretending a new rename class is still part of the closed authority-inventory loop.

## Frozen Rename Graph

- `server/internal/plugin` -> `server/internal/module`
- `server/internal/pluginregistry` -> `server/internal/moduleregistry`
- `server/plugins/<name>` -> `server/modules/<name>`

## Dependent Authority Classes

1. package/import path authority
   - every package declaration and import site coupled to the moved directories
2. generator authority
   - generator constants, generated import aliases, and generated registry output references tied to current paths
3. migration path authority
   - strings and defaults that still encode `modules/<name>/migrations`
4. runtime consumer authority
   - runtime assembly imports and references, especially `server/internal/app/runtime.go`
5. docs/recovery authority
   - governance/design/recovery files that still describe current physical paths as canonical

## Planned Follow-Up Batches

- Batch 2: core package/path migration
  - moved `server/internal/plugin/**` and `server/internal/pluginregistry/**` first, then repaired direct runtime consumers
  - required validation: `cd server && go run ./cmd/graft validate backend --stage lint`, minimum direct `go test`, `cd server && go build ./cmd/graft`
- Batch 3: business module directory migration
  - moved `server/plugins/**` to `server/modules/**` and repaired affected backend imports in bounded scope
  - required validation: `cd server && go run ./cmd/graft validate backend --stage lint`, touched-package `go test`, `cd server && go build ./cmd/graft`
- Batch 4: migration authority and path-string migration
  - update migration directory strings, module/moduleregistry docs, and governance truth after directories moved
  - required validation: `cd server && go run ./cmd/graft validate backend --stage lint`, minimum direct `go test`, `cd server && go build ./cmd/graft`
- Batch 5: docs and archive-readiness
  - reconcile governance/design/recovery truth after code rename completes
  - required validation: `git diff --check`; add `cd web && bun run check` if frontend/shared consumer authority changes

## Batch 4 Execution Update

- aligned package docs so `server/internal/module/**` and `server/internal/moduleregistry/**` no longer describe retired `plugin/pluginregistry` paths as the current canonical physical authority
- aligned active governance and design wording so `server/modules/**`, `server/internal/module/**`, and `server/internal/moduleregistry/**` are the current physical-path truth, while historical `Plugin` naming remains explicitly marked as symbol-level legacy only
- kept archived-topic prompts untouched; only the active topic summary in `ai-plan/public/README.md` was updated to reflect the accepted current state

## Validation Record

- executed:
  - `git diff --check`
  - `cd server && go test ./internal/module ./internal/moduleregistry ./internal/moduleregistry/cmd/pluginregistrygen ./internal/app ./modules/...`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go build ./cmd/graft`
- result:
  - all Batch 4 validation commands passed

## Scope Guard

- no compatibility alias or adapter was introduced
- no `web` implementation file was changed

## Batch 5 Closeout

- performed the final archive-readiness review against active governance, design, package-doc, and recovery authority
- confirmed no current-authority file in owned scope still presents `server/internal/plugin/**`, `server/internal/pluginregistry/**`, or `server/plugins/**` as the live canonical physical path
- confirmed remaining old-path hits are limited to:
  - archived-topic historical prompts and archive summaries
  - explicit rename-map / trace history inside this topic
  - intentionally retained stable authority under `internal/pluginapi/**`
- accepted the topic as `archive-ready`
- recorded a new-topic-only follow-up rule for any future `pluginapi` or exported-symbol rename work
