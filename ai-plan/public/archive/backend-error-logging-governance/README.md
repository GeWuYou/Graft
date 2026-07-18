# Backend Error Logging Governance

## Current Status Summary

- Topic objective: establish one backend error-reporting authority and migrate server failure paths so operators see the real cause once.
- Current status: `archived`
- Task class: `server`
- Intake summary: long-running refactor requiring design alignment, active-topic recovery, and loop execution.
- Canonical authority:
  - `ai-plan/design/domains/audit/日志治理开发规范.md`
  - `server/internal/logger/**`
  - `server/internal/httpx/**`
- Completed: shared foundation, Project/Task priority chain, administrative and operational routes, asynchronous recovery/cause integrity, bounded static governance guard, and final full audit.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `subtopic`
- authority summary: logger owns application error reporting, httpx owns request outcomes and recovery, and typed internal errors own safe HTTP presentation metadata.

## Owned Scope

- `server/internal/{apperror,logger,httpx,requestctx,module,app}/**`
- affected `server/modules/**` compile adaptations and later error-path migrations
- backend logging/error governance and this topic's recovery materials

Out of scope:

- `web/**`, OpenAPI wire-shape changes, database schemas, and migrations
- new observability dependencies, alternate loggers, or alternate response envelopes

## Locked Decisions

1. One error is recorded once by the layer that best understands its operational meaning; HTTP fallback only records an otherwise unreported internal error.
2. Access Log always records request facts at `INFO`; only panic recovery records a full stack.
3. Internal causes remain available to logs and control flow but never enter public response data.
4. Request correlation is established before business execution and enriched, not rebuilt, by authentication.

## Phase Plan

1. Batch 1 - authority, topic bootstrap, and core error/logging foundation
2. Batch 2 - Project/Task priority request failure chain
3. Batch 3 - remaining synchronous administrative HTTP modules
4. Batch 4 - operational synchronous HTTP modules (container, audit, monitor, realtime, dashboard, httpx explorer)
5. Batch 5 - asynchronous recovery, cause integrity, and logging governance guard
6. Batch 6 - full audit, validation, and archive readiness

## Completion Evidence

- Batch 6 found and repaired remaining unreported internal HTTP failures in shared authorization, App Log Explorer, realtime subscription issuance, and container/project authorization boundaries.
- Full handwritten production Go review found no remaining cause-breaking `%v` wrappers, access-log ERROR/WARN promotion, raw `err.Error()` public data exposure, or unowned panic boundary.
- No authority escalation or compatibility exception is active; no further batch remains in scope.

## Work Intake

- This topic was created through `Work Intake`.
- The full `Work Contract` is persisted in the tracking file.

## Archive Status

- The topic was archived after Batch 6 validation. Its retained historical evidence lives at `ai-plan/public/archive/backend-error-logging-governance/` and is not an active recovery entry.

## Validation Targets

```bash
python3 scripts/validate_ai_plan_structure.py
cd server && go run ./cmd/graft validate backend
git diff --check
```

## Historical Entry

- Historical receipt: `ai-plan/public/archive/backend-error-logging-governance/startup-prompt.md`
- The topic has no pending batch and must not resume `$graft-multi-agent-loop` execution.
