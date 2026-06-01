# Module-Oriented Modular Monolith

## Status

- Topic: `module-oriented-modular-monolith`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
- Parent topic: `module-oriented-modular-monolith`
- Loop mode: `topic-completion-loop`
- Current batch: `Batch 4: archive-readiness check for this topic slice`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - root `AGENTS.md`, `server/AGENTS.md`, and `web/AGENTS.md` own the repository execution truth for this correction slice
  - `ai-plan/design/项目设计.md` and `ai-plan/design/插件与依赖注入设计.md` own the architecture narrative being corrected
  - `server/internal/plugin/**` and `server/internal/pluginregistry/**` own the historical backend naming surfaces being re-described as compile-time modules
  - `ai-plan/public/**` owns branch/topic recovery mapping for this bounded correction slice

## Goal

This bounded loop kept symbols and paths stable while correcting wording-only drift back to `Module-Oriented Modular Monolith` semantics.

Bounded outcomes:

1. complete wording-only migration in owned governance, design, comment, and README surfaces
2. classify remaining symbol renames by safety boundary before any identifier cleanup
3. document that backend `plugin` naming is historical and semantically means compile-time business modules
4. keep recovery materials truthful to the active loop, accepted Batch 1 inventory, and current batch state

Final topic result:

- safe wording migration is complete inside the owned governance, design, README, and comment surfaces
- remaining `plugin`-first identifiers inside scope are either intentionally historical authority-preserving names or deferred because renaming them would require exported symbol, package/path, import, generator, or runtime churn beyond this bounded slice
- no compatibility adapter, runtime feature switch, or physical rename was introduced

## Scope

- included:
  - root/server/web governance documents
  - relevant architecture design docs
  - `server/internal/plugin/**`
  - `server/internal/pluginregistry/**`
  - backend module `descriptor.go` comments
  - `ai-plan/public/module-oriented-modular-monolith/**`
  - `ai-plan/public/README.md`
- excluded:
  - package, import, and exported-name renames unless a later batch reclassifies one as safe
  - runtime feature switch implementation
  - physical directory renaming from `plugin/plugins` to `module/modules`
  - new compatibility layers or dual public APIs
  - `web/src/**` implementation changes

## Loop State

- completed batches:
  - `Batch 1: inventory and classify remaining backend plugin semantics`
  - `Batch 2: complete safe server/docs wording migration`
  - `Batch 3: define bounded symbol migration plan`
  - `Batch 4: archive-readiness check for this topic slice`
- current batch:
  - none
- pending batches:
  - none

## Follow-Up Status

- Follow-up status: `new-topic-only`
- Deferred future topic:
  - `module-symbol-and-path-authority-migration`
- Deferred topic intent:
  - only open this if the repository explicitly wants exported symbol, package/path, import, generator output, or runtime-facing historical `plugin` names re-evaluated at the true authority owner
- Deferred topic non-goals:
  - do not treat this archive-ready topic as permission to continue renaming symbols in place
  - do not add compatibility aliases, runtime switches, or physical path renames as a continuation of this slice

## Next-Session Prompt

`Re-run startup preflight from root AGENTS.md. Governance source: root AGENTS.md. Task class: cross-boundary. Recovery source: parent topic module-oriented-modular-monolith. Owned scope: server/internal/plugin/**, server/internal/pluginregistry/**, server/plugins/*/descriptor.go, ai-plan/design/项目设计.md, ai-plan/design/插件与依赖注入设计.md, ai-plan/public/module-oriented-modular-monolith/**, ai-plan/public/README.md, and only the minimum additional authority files required by exported symbol or path migration analysis. Treat module-oriented-modular-monolith as archive-ready evidence and open a new bounded topic only as module-symbol-and-path-authority-migration; do not continue this wording-migration loop in place.`

## Validation Plan

- `git diff --check`
- `cd server && go run ./cmd/graft validate backend --stage lint`
- `cd server && go test ./internal/plugin/... ./internal/pluginregistry/...`
- `cd server && go build ./cmd/graft`
- `cd web && bun run check` when the slice remains `cross-boundary`
- consistency search across touched governance and design docs
