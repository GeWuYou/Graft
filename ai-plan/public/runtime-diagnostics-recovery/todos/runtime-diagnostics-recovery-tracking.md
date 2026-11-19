# Runtime Diagnostics Recovery Tracking

## Topic

Runtime Diagnostics Recovery

## Scope

Bounded cross-boundary optimize surface: execution stage attribution, failure classification, sanitized diagnostic
summary, and claim-admission tri-state readiness projected end-to-end with additive-only contract changes.

## Repository Truth

- `AGENTS.md`
- `server/internal/moduleapi/**`, `server/modules/task/**`, `server/modules/runtime-target/**`
- `server/internal/dockerexecution/**`, `server/agents/docker-runtime-agent/**`
- `openapi/**`, `web/src/modules/task/**`, runtime-target web display, module locale keys

## Work Contract

```yaml
version: 1
kind: optimize
scope: runtime-diagnostics-recovery PR surface - task receipt settlement diagnostics, agent claim admission telemetry,
  provider error boundary redaction, runtime-target readiness projection, and web presentation
authority_summary: Task Runtime owns settlement persistence and task/stage projection; Runtime Target owns agent
  readiness projection and telemetry evidence; Docker Runtime Agent owns claim admission observation; canonical
  OpenAPI owns additive schema closure; web modules own presentation with zh+en locales.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: lead-managed workers
bootstrap:
  targets:
    - topic
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Governance topic created before implementation commits; implementation slices pending in the
  `feature/runtime-diagnostics-projection` worktree.

## Task Checklist

- [x] Contract/backend: bounded `ExecutionStage` enum, receipt settlement extension, task/stage persistence, TaskView
      projection, and HTTP responses.
- [x] Provider: sanitized compose_load failure summary extraction with redaction unit tests.
- [x] Agent: claim rejection tri-state tracking, bounded telemetry dedup/rate limiting, healthcheck surfacing.
- [x] Runtime Target: ledger diagnostic acceptance, latest-receipt read, readiness projection of
      no_work / identity_rejected / transport_failed through the additive `claim_admission` property.
- [x] Web: render failure_stage / failure_class / claim admission with zh+en i18n.
- [x] OpenAPI closure: regenerate bundle, server bindings, runtime embedded bundle, and web schema; additive gate passes.

## Acceptance Conditions

- `just openapi-check` passes without new approved breaking changes.
- `cd server && go run ./cmd/graft validate backend` passes on the touched scope.
- `cd web && bun run check` passes for the touched web scope.
- No paths, credentials, or raw provider stderr appear in receipts, logs, or task logs.
- Additive-only contract changes; `ExecutionAdmission` semantics unchanged.

## Loop Batch State

```json
{
  "loop_mode": "single-worker-bounded-topic",
  "completed_batches": [],
  "active_batch": "contract-backend-settlement",
  "worker": "track1-diagnostics"
}
```
