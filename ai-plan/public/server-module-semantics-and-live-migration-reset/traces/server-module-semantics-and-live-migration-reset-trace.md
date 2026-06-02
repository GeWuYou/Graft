# Server Module Semantics And Live Migration Reset Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md`.
- Kept the task classified as `cross-boundary`.
- Read root `AGENTS.md`, `.ai/environment/tools.ai.yaml`, `server/AGENTS.md`, `web/AGENTS.md`, `ai-plan/public/README.md`, and the earlier active module migration topics before writing recovery materials.
- Confirmed the currently observed drift is not limited to archived `plugin` naming history; active docs and live module migrations still need a bounded new topic.
- Opened a new active topic at `ai-plan/public/server-module-semantics-and-live-migration-reset/**`.
- Froze the loop scope, exclusions, validation chain, future batch plan, and destructive migration assumption.
- Corrected active recovery wording so earlier topics remain truthful for their own accepted scope without claiming the current server semantics cleanup is already complete.

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/module/**`, `server/internal/moduleregistry/**`, `server/internal/app/**`, and `server/modules/**` are the current code authority owners for remaining internal `plugin` semantics and live migration cleanup
  - active `ai-plan/design/**` and active `ai-plan/public/**` are the current authority owners for stale governance and recovery wording
  - module-owned live migrations under `server/modules/*/migrations/**` are the authority for the approved destructive reset
  - shared wire/domain values remain excluded stable semantics in this loop

## Why A New Topic Was Required

- `module-oriented-modular-monolith` closed as a wording-and-comment migration topic with symbols and paths intentionally deferred.
- `module-symbol-and-path-authority-migration` closed as an inventory/validation topic and did not land the broader remaining cleanup.
- `module-historical-plugin-naming-migration` closed with a narrower accepted claim than the repository's current visible state now supports.
- active docs and live module migrations still expose current-authority drift, so continuing to point recovery at the closed topics would be untruthful.

## Batch 1 Decisions Frozen

- scope remains limited to recovery truth and topic docs
- destructive live migration rewrite is approved for a later batch because the user will delete the local database
- no archive topics were edited
- no OpenAPI/shared-contract rename was approved
- no Batch 2-6 implementation work started in this round

## Validation Record

- executed:
  - `git diff --check`

## Scope Guard

- no Go/runtime/module migration code changed in this round
- no archive topic was edited
- no `web/src/**` file changed

## Batch 6 Archive-Readiness Audit

- Re-read the current topic docs, active public index, and active non-archive residual search output before final closeout.
- Confirmed the accepted delta from Batches 2-5 now lands in the repository truth as intended:
  - active design docs no longer route current authority through `ai-plan/design/插件与依赖注入设计.md`
  - active recovery routing now points the remaining cleanup to this topic instead of the earlier narrower module/plugin topics
  - Batch 3 internal wording cleanup and Batch 4 destructive migration reset remain within the declared owned scope
- Classified remaining `plugin` hits relevant to this topic into allowed residual classes only:
  - archive/history/traces
  - intentionally retained shared wire/domain contract values such as `plugin_disabled`, `scopeKindPlugin`, and `PluginDependencyMissing`
  - explicit historical wording that is not presented as current authority, such as retained file names, stable exported identifiers, or generator/package names intentionally left outside this loop
- Counted the already-successful main-thread rerun of `cd server && go run ./cmd/graft validate smoke` as the final Batch 4 smoke evidence after Redis recovery.
- Closed the topic as `archive-ready` instead of leaving an active recovery entry once no further bounded batches remained.
