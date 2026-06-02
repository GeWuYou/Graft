# Server Module Semantics And Live Migration Reset

## Status

- Topic: `server-module-semantics-and-live-migration-reset`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `module-historical-plugin-naming-migration`
- Loop mode: `topic-completion-loop`
- Current batch: `none`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/module/**`, `server/internal/moduleregistry/**`, `server/internal/app/**`, and `server/modules/**` own the remaining internal historical `plugin` semantics and module runtime wording
  - active `ai-plan/design/**` and active `ai-plan/public/**` own the still-live governance and recovery wording that must stop claiming the current cleanup is already complete
  - module-owned live migrations under `server/modules/*/migrations/**` are the authority for the destructive reset because the user approved delete-and-rerun from an empty database
  - shared wire/domain values such as `plugin_disabled`, `scopeKindPlugin`, and `PluginDependencyMissing` remain excluded stable semantics in this topic

## Goal

Open a new bounded active topic for the remaining server module semantics correction after the earlier historical naming topics closed with narrower accepted scope.

This topic must:

1. correct active recovery truth so earlier topics are not treated as proof that current active semantics cleanup is finished
2. freeze the exact owned scope, exclusions, validation chain, destructive migration assumption, and batch plan for the remaining work
3. keep Batch 1 limited to recovery materials and authority reset
4. close truthfully after Batch 6 once active docs, server internal wording, live module migrations, and active recovery routing are reconciled

This topic must not:

- start broad active design cleanup yet
- start internal Go renames yet
- rewrite live migrations in Batch 1
- edit archive topics
- broaden into OpenAPI or shared contract renames

## Scope

- included:
  - `server/internal/module/**`
  - `server/internal/moduleregistry/**`
  - `server/internal/app/**`
  - `server/modules/**`
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md` only for terminology/reference repair
  - active `ai-plan/design/**`
  - active `ai-plan/public/**`
- excluded:
  - `ai-plan/public/archive/**`
  - `openapi/**`
  - `server/internal/contract/openapi/**`
  - generated files
  - shared wire/domain values such as `plugin_disabled`, `scopeKindPlugin`, and `PluginDependencyMissing`

## Destructive Migration Assumption

- the user explicitly approved destructive live migration rewrite for module-owned migration chains
- local database continuity, old migration checksum compatibility, and bridge/no-op checkpoint preservation are not required in this topic
- the accepted validation model for migration rewrite is delete database, rerun migrations from empty state, then re-run the bounded backend validation chain

## Validation Chain

- Batch 1:
  - `git diff --check`
- Batch 2:
  - `git diff --check`
- Batch 3:
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - minimum justified `go test`
  - `cd server && go build ./cmd/graft`
- Batch 4:
  - `cd server && go run ./cmd/graft migrate up`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - minimum justified `go test`
  - `cd server && go build ./cmd/graft`
  - `cd server && go run ./cmd/graft validate smoke`
- Batch 5:
  - `git diff --check`
- Batch 6:
  - `git diff --check`
  - rerun the still-applicable server validation required by the accepted delta before archive readiness is claimed

## Batch Plan

- Batch 1: authority reset and inventory freeze
  - scope: new active topic recovery materials, `ai-plan/public/README.md`, and the minimum older active recovery files needed to correct stale closeout claims
  - focus: truthful active recovery entry, frozen scope, exclusions, validation, and destructive migration assumption
- Batch 2: active design and governance authority cleanup
  - focus: active authority docs stop treating `plugin` as the current canonical architecture term
- Batch 3: server internal non-wire naming cleanup
  - focus: internal helper, test, README, and comment cleanup under `server/internal/**` and `server/modules/**`
- Batch 4: destructive live module migration reset
  - focus: rewrite module-owned live migration chains for empty-database replay
- Batch 5: active recovery topic reconciliation
  - focus: active non-archive recovery surfaces stop echoing outdated plugin-first conclusions or retired paths
- Batch 6: archive-readiness check and closeout
  - focus: confirm only intentional/history/shared-contract `plugin` residues remain

## Final Outcome

- completed batches:
  - Batch 1: authority reset and inventory freeze
  - Batch 2: active design and governance authority cleanup
  - Batch 3: server internal non-wire naming cleanup
  - Batch 4: destructive live module migration reset
  - Batch 5: active recovery topic reconciliation
  - Batch 6: archive-readiness check and closeout
- acceptance result:
  - active design and governance docs now treat `module` as the canonical current architecture term, with `plugin` retained only where explicitly documented as historical naming or stable contract vocabulary
  - `server/internal/module/**`, `server/internal/moduleregistry/**`, `server/internal/app/**`, and the accepted `server/modules/**` internal wording surfaces no longer present the cleaned-up non-wire `plugin` naming as current authority
  - live module migrations for `user`, `rbac`, and `audit` were destructively collapsed to module-owned baseline chains suitable for empty-database replay
  - active non-archive recovery routing now points current continuation to this topic during the loop and no longer presents the earlier closed module/plugin topics as proof that the remaining cleanup was already complete
  - Batch 6 residual searches confirmed that remaining `plugin` hits relevant to this topic are now limited to allowed residual classes: archive/history/traces, intentionally retained shared wire/domain contract values, or explicit historical wording not presented as current authority
- final status:
  - this topic is `archive-ready`
  - future work must open a new bounded topic if the repository later decides to rename retained shared/public `plugin` authority such as `internal/pluginapi/**`, generated OpenAPI values, or other stable wire/domain semantics
