# Backend Error Logging Governance

## Current Status Summary

- Topic objective: establish one backend error-reporting authority and migrate server failure paths so operators see the real cause once.
- Current status: `active`
- Task class: `server`
- Intake summary: long-running refactor requiring design alignment, active-topic recovery, and loop execution.
- Canonical authority:
  - `ai-plan/design/domains/audit/日志治理开发规范.md`
  - `server/internal/logger/**`
  - `server/internal/httpx/**`
- Completed so far: shared foundation, Project/Task priority chain, and remaining synchronous administrative HTTP modules are committed.
- Not started yet: operational synchronous modules, asynchronous recovery/cause integrity, and final governance audit.

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

## Current Recovery Point

- Batch 3 moved auth, user, RBAC, scheduler, announcement, notification, system-config, security, and runtime-target synchronous unknown failures to the shared safe AppError path.
- No authority escalation or compatibility exception is active.
- Next step: `Batch 4 - operational synchronous HTTP modules (container, audit, monitor, realtime, dashboard, httpx explorer)`.

## Work Intake

- This topic was created through `Work Intake`.
- The full `Work Contract` is persisted in the tracking file.

## Pending Batch Direction

- Migrate operational synchronous HTTP modules, retaining the shared AppError/ReportError authority.

## Validation Targets

```bash
python3 scripts/validate_ai_plan_structure.py
cd server && go run ./cmd/graft validate backend
git diff --check
```

## Loop Entry

- Preferred entry: `ai-plan/public/backend-error-logging-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
