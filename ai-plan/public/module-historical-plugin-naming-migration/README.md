# Module Historical Plugin Naming Migration

## Status

- Topic: `module-historical-plugin-naming-migration`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `module-physical-path-migration`
- Loop mode: `topic-completion-loop`
- Current batch: `Batch 6: archive-readiness scan and closeout`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/pluginapi/**` remains the current stable cross-module capability authority, but the repository has now locked the next rename target to `server/internal/moduleapi/**`
  - `server/internal/module/**`, `server/internal/moduleregistry/**`, `server/internal/app/runtime.go`, `server/internal/httpx/**`, `server/internal/cli/**`, and `server/modules/**` own the remaining backend historical `Plugin` lifecycle and constructor naming that must be renamed at their true authority owners
  - root `AGENTS.md`, `server/AGENTS.md`, and active `ai-plan/design/**` files own the current governance and architecture wording that must stay aligned with the accepted rename map
  - `web/**` remains downstream-only for this topic unless current authority wording or visible product copy must be updated in the same accepted batch
  - `ai-plan/public/module-historical-plugin-naming-migration/**` owns truthful recovery for this bounded follow-up topic

## Why This Is A New Topic

- `module-physical-path-migration` is already `archive-ready` and explicitly limits its accepted result to physical directory and package-path authority.
- The remaining work is no longer physical-path migration. It is the retained historical naming inside:
  - exported lifecycle symbols
  - `pluginapi` package/path authority
  - constructor/type names under `server/modules/**`
  - current governance wording and visible copy
- Continuing the archived topic in place would falsify recovery state by pretending a new rename class is still part of the closed physical-path loop.

## Goal

This archive-ready topic no longer owns the current remaining work; that active continuation now lives in `server-module-semantics-and-live-migration-reset`.

This topic must:

1. freeze the accepted rename map for the remaining historical naming surfaces
2. define the exact in-scope authority owners, exclusions, and validation chain
3. split the remaining work into bounded batches that can be accepted independently
4. keep Batch 1 docs/recovery only

This topic must not:

- start Batch 2 code renames
- reopen physical-path migration
- introduce compatibility aliases, adapters, fallback layers, or dual naming
- broaden into unrelated `web` implementation work

## Locked Decisions

- package rename target:
  - `server/internal/pluginapi` -> `server/internal/moduleapi`
- exported lifecycle naming target:
  - `Module`
  - `NewModule`
  - `RegisterModule`
  - related `Module` family names at the same authority owner
- retained history policy:
  - archive/history files stay untouched unless active index summarization is required
- compatibility policy:
  - no alias, adapter, fallback, or compatibility bridge is allowed for this migration

## Accepted Rename Map

### Batch 2 Authority

- exported lifecycle naming under current backend runtime/module authority:
  - `Plugin` -> `Module`
  - `NewPlugin` -> `NewModule`
  - `RegisterPlugin` -> `RegisterModule`
  - related exported `Plugin` lifecycle family names -> matching `Module` family names at the same owner

### Batch 3 Authority

- package and import-path authority:
  - `server/internal/pluginapi` -> `server/internal/moduleapi`
  - import sites must follow the package rename in the same accepted batch

### Batch 4 Authority

- constructor/type naming under `server/modules/**` and coupled runtime consumers:
  - retained constructor/type names using historical `Plugin` wording -> accepted `Module` family naming

### Batch 5 Authority

- current governance wording and visible copy:
  - active docs and user-visible copy must stop presenting retained `plugin` naming as current canonical wording after the code rename batches land

## Scope

- included:
  - `server/internal/pluginapi/**`
  - `server/internal/module/**`
  - `server/internal/moduleregistry/**`
  - `server/internal/app/runtime.go`
  - `server/internal/httpx/**`
  - `server/internal/cli/**`
  - `server/modules/**`
  - `AGENTS.md`
  - `server/AGENTS.md`
  - active `ai-plan/design/**`
  - active `ai-plan/public/**` excluding `ai-plan/public/archive/**`
- conditionally included:
  - `web/**` only if current active authority wording or visible product copy must be updated in the same accepted batch
- excluded:
  - `ai-plan/public/archive/**`
  - physical directory/package-path renames already accepted by `module-physical-path-migration`
  - OpenAPI, generated artifact, schema, migration, or unrelated UI expansion unless later authority discovery proves they are directly required by one accepted batch

## Validation Chain

- Batch 1:
  - `git diff --check`
- Batch 2:
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - minimum direct `go test`
  - `cd server && go build ./cmd/graft`
- Batch 3:
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - minimum direct `go test`
  - `cd server && go build ./cmd/graft`
- Batch 4:
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - minimum direct `go test`
  - `cd server && go build ./cmd/graft`
- Batch 5:
  - `git diff --check`
  - add `cd web && bun run check` only if the accepted batch changes `web` files
- Batch 6:
  - `git diff --check`
  - rerun any still-applicable server/web validation required by the accepted code/doc delta before archive-readiness is claimed

## Batch Plan

- Batch 1: open a new active topic for remaining historical plugin naming migration and freeze the accepted rename map and scope
  - scope: `ai-plan/public/module-historical-plugin-naming-migration/**`, `ai-plan/public/README.md` only if needed
  - validation: `git diff --check`
- Batch 2: core exported lifecycle rename
  - scope: `server/internal/module/**`, `server/internal/moduleregistry/**`, `server/internal/app/runtime.go`, and the minimum coupled docs in active scope
  - focus: exported lifecycle `Plugin` -> `Module` family authority rename
- Batch 3: `pluginapi` -> `moduleapi` package rename
  - scope: `server/internal/pluginapi/**`, import consumers in owned scope, and the minimum coupled docs in active scope
  - focus: package path and import truth without compatibility aliasing
- Batch 4: `server/modules` constructor/type rename
  - scope: `server/modules/**`, `server/internal/httpx/**`, `server/internal/cli/**`, `server/internal/app/runtime.go`, and other minimum authority files required by the accepted rename
  - focus: retained module constructor/type naming still using historical `Plugin` wording
- Batch 5: current authority docs and visible copy cleanup
  - scope: `AGENTS.md`, `server/AGENTS.md`, active `ai-plan/design/**`, active `ai-plan/public/**`, and `web/**` only if current visible copy must be updated
  - focus: active authority wording and visible product copy only; archives stay untouched
- Batch 6: archive-readiness scan and closeout
  - scope: active topic recovery docs plus any minimum active index/design updates needed for truthful final status
  - focus: verify no active current-authority surface still presents retired naming as canonical unless intentionally retained

## Batch 1 Record

- result:
  - opened `module-historical-plugin-naming-migration` as a new active bounded topic after the parent physical-path topic closed as `archive-ready`
- decisions frozen:
  - `server/internal/pluginapi` is the accepted next package rename target and must migrate to `server/internal/moduleapi`
  - exported lifecycle naming target is the `Module / NewModule / RegisterModule` family
  - Batch 1 remains docs/recovery only
- exclusions frozen:
  - no code rename started
  - no archive topic edited
  - no compatibility bridge allowed

## Batch 6 Closeout

- result:
  - re-scanned the active current-authority surfaces that were inside this topic's accepted bounded scope
  - confirmed no additional repair was taken inside this closed topic beyond that accepted scope
  - left any later-discovered active-doc drift, remaining internal non-wire naming drift, and destructive live migration reset work for a new bounded topic
  - this topic now stands only as parent evidence for `server-module-semantics-and-live-migration-reset`
- acceptable residual hit classes:
  - active recovery summaries describing prior rename stages or archived plugin-named topics
  - governance wording that explicitly marks `plugin` as historical backend naming for compile-time modules
  - intentional domain or wire semantics where the stable value or user-facing concept is literally `plugin`
  - registry or metadata fields named `Plugin` that still describe source ownership rather than live path authority
- blocking drift not found in this round:
  - no active current-authority recovery doc still presents `internal/pluginapi/**` as the live canonical cross-module boundary
  - no active current-authority recovery doc still presents `server/plugins/*` as the live physical module path
  - no active current-authority recovery doc still presents exported lifecycle `Plugin` naming as the canonical target for new work

## Loop State

- completed batches:
  - `Batch 1: open a new active topic for remaining historical plugin naming migration and freeze the accepted rename map and scope`
  - `Batch 6: archive-readiness scan and closeout`
- current batch:
  - none
- pending batches:
  - none

## Closeout

- terminal status:
  - `archive-ready`
- stop reason:
  - Batch 6 found no remaining bounded follow-up required inside this topic's accepted scope; later remaining work must open a new bounded topic
