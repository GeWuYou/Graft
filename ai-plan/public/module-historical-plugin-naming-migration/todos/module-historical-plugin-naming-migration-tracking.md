# Module Historical Plugin Naming Migration Tracking

## Topic

- Topic: `module-historical-plugin-naming-migration`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
- Parent topic: `module-physical-path-migration`
- Loop mode: `topic-completion-loop`
- Current batch: `Batch 6: archive-readiness scan and closeout`

## Owned Scope

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
- `web/**` only if current active authority wording or visible product copy must be updated in the same accepted batch

## Frozen Decisions

- package rename target:
  - `server/internal/pluginapi` -> `server/internal/moduleapi`
- exported lifecycle target:
  - `Module`
  - `NewModule`
  - `RegisterModule`
- no compatibility aliases, adapters, or fallback layers
- no archive/history edits in this topic except active index summarization if required

## Batch State

- completed:
  - `Batch 1: open a new active topic for remaining historical plugin naming migration and freeze the accepted rename map and scope`
  - `Batch 6: archive-readiness scan and closeout`
- current:
  - none
- pending:
  - none

## Batch 6 Outcome

- residual active hits reviewed in this round are acceptable as:
  - historical evidence in active recovery summaries
  - explicit governance wording about historical plugin naming
  - intentional domain or wire semantics whose stable value is literally `plugin`
  - visible product copy or test fixtures describing plugin-oriented runtime concepts
- no still-live canonical naming drift remained in owned current-authority recovery docs after the accepted earlier batches
- this round did not reopen code migration or archive edits

## Validation Record

- Batch 1:
  - `git diff --check`
- Batch 6:
  - `git diff --check`
  - no additional validation was justified because this round only updated recovery docs/index files

## Notes

- `module-physical-path-migration` remains archive-ready parent evidence only.
- This topic exists to keep the remaining historical naming migration bounded by rename class instead of resuming the closed physical-path loop.
- This topic is now closed as `archive-ready`; later active-doc drift, remaining internal non-wire naming drift, and destructive live migration reset work moved to `server-module-semantics-and-live-migration-reset`.
