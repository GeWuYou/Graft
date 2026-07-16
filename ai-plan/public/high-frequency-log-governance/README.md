# High Frequency Log Governance

## Current Status Summary

- Topic objective: establish Graft's long-lived `TRACE + Category + Registry` governance for high-frequency runtime logs.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running refactor with a core server authority and downstream App Log/OpenAPI/web consumers.
- Canonical authority:
  - `server/internal/logger/**`
  - `server/internal/config/**`
  - `ai-plan/design/domains/audit/日志治理开发规范.md`
- Completed so far: Work Intake, architecture decision, and Batch 1 logger foundation are complete.
- Not started yet: Batch 2 high-frequency migration and static governance; Batch 3 App Log/OpenAPI/web contract evaluation.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
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
- Batch 2: migrate identified high-frequency call sites and add bounded magic-string/static governance coverage.
- Batch 3: add App Log category persistence/query contract and downstream OpenAPI/web consumer updates when required by the completed runtime baseline.

## Current Recovery Point

- Loop mode: `topic-completion-loop`.
- Current batch: Batch 1 completed.
- Next step: dispatch Batch 2 high-frequency migration and static-governance worker.

## Work Intake

- This topic was created through `Work Intake`.
- The full Work Contract is in `todos/high-frequency-log-governance-tracking.md`.

## Validation Targets

```bash
git diff --check
cd server && go run ./cmd/graft validate backend
```

## Loop Entry

- Preferred entry: `ai-plan/public/high-frequency-log-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
