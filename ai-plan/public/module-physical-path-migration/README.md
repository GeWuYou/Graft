# Module Physical Path Migration

## Status

- Topic: `module-physical-path-migration`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `module-symbol-and-path-authority-migration`
- Loop mode: `topic-completion-loop`
- Current batch: `Batch 5: docs and archive-readiness`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - root `AGENTS.md` owns startup governance, authority-first escalation, anti-compatibility, and cross-boundary validation truth
  - `server/AGENTS.md` owns current backend execution truth for historical `plugin/plugins` naming under compile-time module semantics
  - `web/AGENTS.md` confirms frontend remains a downstream consumer of backend/shared authority and must not define alias-based compensation for backend path drift
  - `ai-plan/design/项目设计.md` and `ai-plan/design/模块与依赖注入设计.md` own the architecture narrative that backend business capability units are canonical `module`s, while current `plugin/plugins` paths are historical naming
  - `ai-plan/public/module-physical-path-migration/**` owns truthful recovery for this bounded physical rename topic

## Why This Is A New Topic

- `module-oriented-modular-monolith` closed as wording-only migration and explicitly deferred symbol/path work to a new bounded topic
- `module-symbol-and-path-authority-migration` completed authority inventory and explicitly deferred physical directories, import paths, generator-coupled path truth, and migration path strings as new-topic-only follow-up
- physical path migration changes canonical filesystem and package authority, not just wording or symbol commentary, so resuming either archived topic in place would violate truthful recovery and bounded-scope governance

## Goal

Open a new bounded topic that owns physical directory/package path migration planning and recovery truth before any code rename batch proceeds.

This topic must:

1. freeze the canonical physical rename map
2. inventory the minimum dependent authority classes that must move with the rename
3. define bounded follow-up batches with explicit validation expectations
4. keep Batch 1 docs-only and recovery-truthful

This topic must not:

- physically rename directories or rewrite imports in Batch 1
- introduce compatibility aliases, dual paths, adapters, or fallback logic
- broaden into `web` implementation, OpenAPI implementation, or unrelated runtime changes

## Canonical Rename Map

- `server/internal/plugin` -> `server/internal/module`
- `server/internal/pluginregistry` -> `server/internal/moduleregistry`
- `server/plugins/<name>` -> `server/modules/<name>`

## Dependent Authority Inventory

The following authority classes must move with the physical rename; none may be “fixed later” by compatibility layers:

1. package and import paths
   - direct Go package declarations and import sites under `server/internal/app/**`, `server/internal/**`, and moved business modules
2. generator constants and generated output references
   - generator directory constants, generated import aliases, and generated registry output currently coupled to `plugin/plugins` paths
3. migration path strings
   - module-owned migration directory strings such as `modules/<name>/migrations`
4. runtime consumer imports
   - runtime assembly consumers such as `server/internal/app/runtime.go`
5. docs and recovery authority paths
   - root/server/web governance docs, design docs, and active recovery materials that still point to old physical paths as canonical authority

## Batch Plan

- Batch 1: open physical path migration topic and freeze rename graph
  - scope: `ai-plan/public/module-physical-path-migration/**`, `ai-plan/public/README.md` only if needed
  - validation: `git diff --check`
- Batch 2: core package/path migration
  - scope: `server/internal/module/**`, `server/internal/moduleregistry/**`, `server/internal/app/runtime.go`
  - validation: `cd server && go run ./cmd/graft validate backend --stage lint`, minimum direct `go test`, `cd server && go build ./cmd/graft`
- Batch 3: business module directory migration
  - scope: `server/plugins/**` to `server/modules/**` plus affected backend imports inside owned authority
  - validation: `cd server && go run ./cmd/graft validate backend --stage lint`, touched-package `go test`, `cd server && go build ./cmd/graft`
- Batch 4: migration authority and path-string migration
  - scope: moved module migration directory strings, registry migration aggregation, package docs, and governance/recovery authority wording
  - validation: `cd server && go run ./cmd/graft validate backend --stage lint`, minimum direct `go test`, `cd server && go build ./cmd/graft`
- Batch 5: docs and archive-readiness
  - scope: governance/design/recovery truth after code rename lands
  - validation: `git diff --check`; if any `web` or shared-consumer file changes, add `cd web && bun run check`

## Loop State

- completed batches:
  - `Batch 1: open physical path migration topic and freeze rename graph`
  - `Batch 2: core package/path migration`
  - `Batch 3: business module directory migration`
  - `Batch 4: migration authority and path-string migration`
  - `Batch 5: docs and archive-readiness`
- current batch:
  - `none`
- pending batches:
  - `none`

## Validation Plan

- Batch 1 baseline:
  - `git diff --check`
- future code-moving batches:
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - minimum justified `go test`
  - `cd server && go build ./cmd/graft`
  - `cd web && bun run check` only if the accepted batch changes `web` or shared frontend-consumer authority

## Batch 4 Record

- status:
  - package docs, governance wording, and active recovery truth now align with `server/internal/module/**`, `server/internal/moduleregistry/**`, and `server/modules/**` as current canonical physical authority
- touched authority:
  - `server/internal/module/README.md`
  - `server/internal/moduleregistry/README.md`
  - `server/AGENTS.md`
  - `ai-plan/design/模块与依赖注入设计.md`
  - `ai-plan/public/module-physical-path-migration/**`
  - `ai-plan/public/README.md` active-topic summary only
- validation:
  - `git diff --check`
  - `cd server && go test ./internal/module ./internal/moduleregistry ./internal/moduleregistry/cmd/pluginregistrygen ./internal/app ./modules/...`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd server && go build ./cmd/graft`
- validation result:
  - all required Batch 4 validation commands passed

## Batch 5 Archive Decision

- archive-readiness result:
  - accepted as `archive-ready`
- rationale:
  - the physical rename is complete across `server/internal/module/**`, `server/internal/moduleregistry/**`, and `server/modules/**`
  - active governance, design, package-doc, and recovery materials in owned scope now treat those paths as the current canonical physical authority
  - remaining `internal/pluginapi/**` references are intentional retained stable symbol/path authority and are not unresolved drift from this topic
  - remaining `server/internal/plugin*` and `server/plugins/**` mentions inside archived-topic prompts or rename-history records are historical evidence, not current normative authority
- docs touched for closeout:
  - `ai-plan/public/module-physical-path-migration/README.md`
  - `ai-plan/public/module-physical-path-migration/traces/module-physical-path-migration-trace.md`
  - `ai-plan/public/README.md`
- validation:
  - `git diff --check`
  - docs-only closeout; Batch 4 already recorded the last required backend validation for accepted authority changes

## Follow-Up Boundary

- no further continuation is required inside this topic
- if the repository later decides to rename retained historical stable surfaces such as `internal/pluginapi/**` or exported `Plugin` symbols, open a new bounded topic at the true authority owner instead of resuming this archive in place
