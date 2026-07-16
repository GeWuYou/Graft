# High Frequency Log Governance

## Current Status Summary

- Topic objective: establish Graft's long-lived `TRACE + Category + Registry` governance for high-frequency runtime logs.
- Current status: `archived` on 2026-07-16
- Task class: `cross-boundary`
- Intake summary: long-running refactor with a core server authority and downstream App Log/OpenAPI/web consumers.
- Canonical authority:
  - `server/internal/logger/**`
  - `server/internal/config/**`
  - `ai-plan/design/domains/audit/日志治理开发规范.md`
- Completed so far: Work Intake, architecture decision, Batch 1 logger foundation, Batch 2 high-frequency migration/static governance, Batch 3 App Log category contract, and Batch 4 category-authority repair.
- Completed: all four implementation batches, final archive-readiness evaluation, and scoped commit review.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `archive`
- authority summary: `server/internal/logger/**` owns runtime logging; `server/internal/config/**` owns startup configuration; App Log/OpenAPI/web are downstream consumers.

## Owned Scope

- `server/internal/logger/**`, `server/internal/config/**`, and directly affected high-frequency server call sites
- logging/configuration governance documents, topic recovery materials, and bounded validation automation

Out of scope:

- replacing `*zap.Logger` dependency injection across the server
- per-category environment variables, a new generic logger framework, or a general metrics/tracing platform

## Locked Decisions

1. Use `TRACE + typed Category + Category Registry + thin CategoryLogger + lazy fields`.
2. Keep `*zap.Logger` injection and use `logger.Category(base, logger.CategoryDockerStats)` at governed call sites.
3. Keep normal polling failures at `WARN` or `ERROR`; only normal high-frequency diagnostics move to `TRACE`.
4. Use one aggregate `GRAFT_LOG_CATEGORIES` configuration instead of per-log boolean switches.

## Phase Plan

- Batch 1: bootstrap Category constants/registry, TRACE policy, configuration parsing, thin facade, tests, and normative docs.
- Batch 2: migrated Docker CPU TRACE diagnostics, periodic/watcher/cache warning categories, and Ent TRACE output; added bounded magic-string/static governance coverage.
- Batch 3: add App Log category persistence/query contract and downstream OpenAPI/web consumer updates when required by the completed runtime baseline.
- Batch 4: repair the AppLogger default/gate, migrate the incorrect Batch 3 legacy category, remove downstream category inventories, and harden the bounded literal scanner.

## Archive Receipt

- Loop mode: `topic-completion-loop`.
- Completed batches: logger foundation, high-frequency migration/static governance, App Log category contract, and category-authority repair.
- Archive decision: all topic acceptance conditions are met within owned scope.
- Validation gap: `cd web && bun run check` reached the test suite and failed only in the unrelated `src/modules/project/pages/configuration-workspace/index.test.ts` assertion at line 1457. App Log tests, format, typecheck, generated OpenAPI governance, i18n, lint, stylelint, and hygiene checks passed before that failure.
- Future work: the project workspace test failure remains owned by the project configuration workspace topic, not this archived logger topic.

## Work Intake

- This topic was created through `Work Intake`.
- The full Work Contract is in `todos/high-frequency-log-governance-tracking.md`.

## Validation Targets

```bash
git diff --check
cd server && go run ./cmd/graft validate backend
```

## Historical Loop Entry

- Historical entry: `startup-prompt.md`
- Execution mode used: `$graft-multi-agent-loop`
