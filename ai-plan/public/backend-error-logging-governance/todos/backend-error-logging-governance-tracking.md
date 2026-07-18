# Backend Error Logging Governance Tracking

## Topic

Backend Error Logging Governance

## Scope

Establish the backend error/logging foundation, migrate owned server failure paths in bounded batches, and close with a full governance audit.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/design/domains/audit/日志治理开发规范.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: server/internal/logger owns application error reporting; server/internal/httpx owns HTTP outcomes and recovery; platform typed errors own safe presentation metadata
requires:
  design: true
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - topic
    - design
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Batches 1 through 4 are complete: synchronous administrative and operational routes preserve their existing expected 4xx mapping while unexpected failures pass through the shared safe AppError boundary.
- No compatibility bridge or authority escalation is approved.
- Next step: start Batch 5 - asynchronous recovery, cause integrity, and logging governance guard.

## Task Checklist

- [x] Batch 1 - authority, topic bootstrap, and core error/logging foundation
- [x] Batch 2 - Project/Task priority request failure chain
- [x] Batch 3 - remaining synchronous administrative HTTP modules
- [x] Batch 4 - operational synchronous HTTP modules (container, audit, monitor, realtime, dashboard, httpx explorer)
- [ ] Batch 5 - asynchronous recovery, cause integrity, and logging governance guard
- [ ] Batch 6 - full audit, validation, and archive readiness

## Acceptance Conditions

- One synchronous system error produces at most one cause-bearing ERROR record and a factual INFO access record.
- Expected 4xx errors do not emit ERROR; unknown internal errors receive one HTTP fallback record.
- Panic recovery emits request correlation and real stack frames without exposing internals to clients.
- Backend completion validation and governance structure checks pass.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "Batch 1 - authority, topic bootstrap, and core error/logging foundation",
    "Batch 2 - Project/Task priority request failure chain",
    "Batch 3 - remaining synchronous administrative HTTP modules",
    "Batch 4 - operational synchronous HTTP modules (container, audit, monitor, realtime, dashboard, httpx explorer)"
  ],
  "pending_batches": [
    "Batch 5 - asynchronous recovery, cause integrity, and logging governance guard",
    "Batch 6 - full audit, validation, and archive readiness"
  ],
  "current_batch": "Batch 4 - operational synchronous HTTP modules (container, audit, monitor, realtime, dashboard, httpx explorer)",
  "next_batch": "Batch 5 - asynchronous recovery, cause integrity, and logging governance guard",
  "closeout_status": "batch-4-complete"
}
```

## Batch 1 Evidence

- Focused foundation tests: `cd server && go test ./internal/requestctx ./internal/apperror ./internal/logger ./internal/httpx ./internal/module ./internal/app`
- Full server tests: `cd server && go test ./...`
- Backend completion entrypoint: `cd server && go run ./cmd/graft validate backend`
- Governance checks: `python3 scripts/validate_ai_plan_structure.py`, `python3 scripts/validate_shared_asset_registries.py`, and `git diff --check`

## Shared Asset Preflight

- status: used
- registries_checked: `.ai/registries/server-shared-assets.yaml`, `.ai/registries/cross-boundary-assets.yaml`
- assets_reused: `httpx-runtime`, `app-logger`, and `request-correlation-contract`
- assets_considered_but_rejected: no existing HTTP-neutral request-context helper; the new `server/internal/requestctx` boundary prevents logger-to-httpx coupling
- new_registry_entries: `application-error-contract`, `request-context`
- registry_entries_removed_or_replaced: none

## Batch 2 Evidence

- Project lifecycle Task submission failures are recorded through `logger.ReportError` with application, action, task type, and request correlation fields; the reported error preserves the original cause for the HTTP boundary.
- Project routes now delegate unhandled errors to `httpx.AbortAppError`, so the HTTP fallback records only unreported failures.
- Task routes use typed safe descriptors and no longer serialize `err.Error()` into public response data.
- Validation: `cd server && go test ./modules/project ./modules/task ./internal/logger ./internal/httpx`, `cd server && go test ./...`, `cd server && go run ./cmd/graft validate backend`, and `git diff --check`.

## Batch 3 Evidence

- Auth and User route owners now use `ReportError` for unexpected failures and pass the reported cause to `AbortAppError`; existing safe auth, validation, and not-found mappings remain silent 4xx responses.
- RBAC read paths report unexpected failures with operation and resource context; scheduler, announcement, notification, system-config, and runtime-target unknown request failures use `AbortAppError`, so the shared HTTP fallback records their cause exactly once when no semantic AppLogger owner exists yet.
- Security overview reports each failed aggregate read with stable operation and preset context; Project lifecycle uses `lifecycle_action` rather than the audit-owned `action` field.
- Validation: focused affected-module tests, `cd server && go test ./...`, `cd server && go run ./cmd/graft validate backend`, `python3 scripts/validate_ai_plan_structure.py`, and `git diff --check`.

## Batch 4 Evidence

- Container unknown route errors now record one semantic AppLogger entry before the shared HTTP boundary; existing status mappings keep expected client and authorization errors silent.
- Audit route read/write and response-mapping failures now use `ReportError` with stable module, operation, and record context; existing audit/security ownership and expected 4xx behavior remain unchanged.
- Monitor server-status and request-performance owners now report semantic 5xx causes without hand-built request identifiers. Access-log explorer delegates unknown repository failures to the correlated HTTP fallback; a regression test proves one log and no public cause leak.
- Realtime ticket/auth and dashboard widget paths were reviewed but intentionally unchanged because their known failures are expected client states or already have a semantic owner.
- Validation: focused affected-package tests, `cd server && go test ./...`, `python3 scripts/validate_ai_plan_structure.py`, and `git diff --check`.
