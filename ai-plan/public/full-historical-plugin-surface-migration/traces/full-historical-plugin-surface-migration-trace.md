# Full Historical Plugin Surface Migration Trace

## Summary

- re-ran the inherited cross-boundary startup context from root `AGENTS.md`
- treated `server-module-semantics-and-live-migration-reset` as archive-ready parent evidence only
- confirmed the new decision: previously retained stable/public `plugin` surfaces are no longer exempt from migration
- opened a new active topic at `ai-plan/public/full-historical-plugin-surface-migration/**`
- froze the rename map, owned scope, exclusions, validation chain, and batch plan for the full migration
- kept Batch 1 limited to recovery materials and active index truth

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/pluginapi/**` and remaining backend runtime naming are now in-scope canonical authorities for rename
  - `openapi/**` and generated artifacts are downstream but required authority-following batches once source renames land
  - `web` remains a downstream consumer for contract fields and visible copy, but its repository-owned module/page copy is in scope for the same topic
  - active `ai-plan/design/**`, active `ai-plan/public/**`, and repository AGENTS files own current recovery and governance truth

## Why A New Topic Was Required

- `server-module-semantics-and-live-migration-reset` explicitly closed with retained shared/public `plugin` authority outside its bounded scope.
- the current user direction removes that retention policy entirely for active repository-owned surfaces.
- continuing any earlier closed topic in place would make recovery state false because the accepted scope has materially widened to backend shared boundaries, OpenAPI, generated consumers, and frontend visible copy.

## Batch 1 Decisions Frozen

- `server/internal/pluginapi` must migrate to `server/internal/moduleapi`
- monitor/runtime/domain/shared contract fields using repository-owned `plugin` wording must migrate to `module`
- visible copy using `插件` / `Plugins` for repository module semantics must migrate to `模块` / `Modules`
- no code rename, spec regeneration, or frontend consumer repair starts in Batch 1
- archive topics remain untouched

## Validation Record

- executed:
  - `git diff --check`

## Scope Guard

- no backend code changed in this round
- no OpenAPI or generated artifact changed in this round
- no `web/src/**` code changed in this round
- no archive topic changed in this round
