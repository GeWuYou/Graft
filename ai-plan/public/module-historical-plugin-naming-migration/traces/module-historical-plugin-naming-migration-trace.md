# Module Historical Plugin Naming Migration Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md`.
- Kept the task classified as `cross-boundary`.
- Read root `AGENTS.md`, `.ai/environment/tools.ai.yaml`, `server/AGENTS.md`, `web/AGENTS.md`, `ai-plan/public/README.md`, and the parent migration topics before writing recovery materials.
- Treated `module-physical-path-migration` as archive-ready parent evidence rather than resuming that closed loop in place.
- Opened a new active bounded topic at `ai-plan/public/module-historical-plugin-naming-migration/**`.
- Froze the accepted remaining rename classes, exclusions, validation chain, and Batch 1-6 plan.
- Kept the round docs/recovery only; no Batch 2 code rename started.
- Batch 6 re-scanned active authority surfaces, classified residual hits, and closed the topic as `archive-ready`.
- That closeout is now bounded historical evidence only; later remaining active-doc drift and live migration reset work is tracked by `server-module-semantics-and-live-migration-reset`.

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/pluginapi/**` is the true authority owner for the retained cross-module package naming that will later migrate to `moduleapi`
  - `server/internal/module/**`, `server/internal/moduleregistry/**`, `server/internal/app/runtime.go`, `server/internal/httpx/**`, `server/internal/cli/**`, and `server/modules/**` hold the remaining historical `Plugin` lifecycle and constructor/type naming that must be renamed at source
  - root `AGENTS.md`, `server/AGENTS.md`, active `ai-plan/design/**`, and active `ai-plan/public/**` hold the current governance truth that must stay aligned with the accepted rename map
  - `web/**` is excluded unless current active authority wording or visible product copy must move in the same accepted batch

## Why The Parent Topic Was Not Resumed

- `module-physical-path-migration` is already `archive-ready`.
- Its closeout explicitly limits accepted work to physical path authority and leaves retained historical symbol/package naming for a new bounded topic.
- Resuming it in place would hide the fact that the remaining work is a different rename class with a different authority split and batch plan.

## Batch 1 Decisions Frozen

- package rename target:
  - `server/internal/pluginapi` -> `server/internal/moduleapi`
- exported lifecycle target:
  - `Plugin` -> `Module`
  - `NewPlugin` -> `NewModule`
  - `RegisterPlugin` -> `RegisterModule`
- constructor/type cleanup target:
  - retained `Plugin` family naming under `server/modules/**` follows the same `Module` family target once upstream lifecycle and package authority are repaired
- exclusions:
  - no Batch 2 code rename
  - no archive/history edits
  - no compatibility alias/adapter/fallback

## Batch Plan Frozen

- Batch 2: core exported lifecycle rename
- Batch 3: `pluginapi` -> `moduleapi` package rename
- Batch 4: `server/modules` constructor/type rename
- Batch 5: current authority docs and visible copy cleanup
- Batch 6: archive-readiness scan and closeout

## Validation Record

- executed:
  - `git diff --check`
- result:
  - passed

## Scope Guard

- no code authority changed in this round
- no archive topic was edited
- no `web` file changed

## Batch 6 Archive-Readiness Scan

- scanned active recovery/docs surfaces for:
  - `pluginapi`
  - `server/plugins`
  - exported `Plugin` lifecycle wording
  - lower-case `plugin` wording that might still present retired naming as current canonical authority
- accepted residual-hit classes:
  - active recovery summaries that describe prior rename history or archived plugin-named topics
  - governance wording that explicitly marks `plugin` as historical backend naming for compile-time modules
  - domain and wire semantics where `plugin` remains the intended stable value or user-facing concept
- non-blocking examples from the scan:
  - OpenAPI/generated scope-kind enum values using `plugin`
  - cron/menu/permission/audit metadata fields named `Plugin`
  - monitor/audit product copy referring to plugin dependency or plugin startup investigation
- blocking drift not found:
  - no active current-authority recovery/index file still presented `internal/pluginapi/**` as the live canonical cross-module boundary
  - no active current-authority recovery/index file still presented `server/plugins/*` as the live physical module path
  - no active current-authority recovery/index file still instructed future work to keep exported lifecycle `Plugin` naming as canonical
