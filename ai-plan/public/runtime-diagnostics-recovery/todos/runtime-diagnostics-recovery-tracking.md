# Runtime Diagnostics Recovery Tracking

## Topic

Runtime Diagnostics Recovery

## Scope

Bounded cross-boundary optimize surface: execution diagnostics projection (delivered in PR #376) and
policy-driven bounded automatic recovery for interrupted external execution stages.

## Repository Truth

- `AGENTS.md`
- `server/internal/moduleapi/**`, `server/modules/task/**`, `server/modules/runtime-target/**`
- `server/internal/dockerexecution/**`, `server/agents/docker-runtime-agent/**`
- `openapi/**`, `web/src/modules/task/**`, runtime-target web display, module locale keys
- `scripts/validate_docker_isolated.py` + `scripts/test_validate_docker_isolated.py` (isolated acceptance)

## Work Contract

```yaml
version: 1
kind: optimize
scope: runtime diagnostics projection and bounded automatic recovery of interrupted external stages with
  additive-only contract changes
authority_summary: Task Runtime owns settlement, stage recovery and bounded auto-resume; Runtime Target owns
  readiness projection and telemetry evidence; Docker Runtime Agent owns claim admission observation;
  canonical OpenAPI owns additive schema closure; web modules own presentation with zh+en locales.
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

- PR #376 diagnostics projection is delivered and merged (9abe8cc1c).
- Bounded auto-recovery round (worker track1-recovery, branch feature/bounded-execution-recovery) is
  implemented and locally validated; commits prepared in the task worktree, not pushed.

## Task Checklist

- [x] Delivered PR #376: settlement diagnostics, claim admission telemetry, readiness projection, web
      presentation, OpenAPI closure (history compressed into the trace).
- [x] Bounded auto-recovery: candidates listing, atomic task-level resume with recovery_resolved exactly
      once, budget-exhaustion convergence with stable diagnostics, unclaimable-pending hang fix.
- [x] Regressions: policy-driven resume, budget exhaustion, manual_reconcile untouched, explicit
      needs_attention untouched, idempotent claim replay, multi-leg atomic resume, rejected resume rollback.
- [x] Harness scenarios: compose-material-stage-code (local gate); agent-claim-admission-rejected,
      agent-recovery-resume, agent-recovery-budget-exhausted (agent gate) with probe/helper unit tests.
- [ ] Full live harness probe on a registered agent-mode fixture (developer primary checkout after merge).

## Acceptance Conditions

- `just openapi-check` passes without new approved breaking changes.
- `cd server && go run ./cmd/graft validate backend` passes on the touched scope.
- `cd web && bun run check` passes for the touched web scope.
- No paths, credentials, or raw provider stderr appear in receipts, logs, or task logs.
- Additive-only contract changes; `ExecutionAdmission` semantics unchanged.
- manual_reconcile stages stay manual; only runtime-recorded lease interruptions auto-resume within budget.

## Loop Batch State

```json
{
  "loop_mode": "single-worker-bounded-topic",
  "completed_batches": [
    "contract-backend-settlement",
    "agent-claim-admission",
    "provider-error-boundary",
    "runtime-target-readiness",
    "web-presentation",
    "openapi-closure",
    "bounded-auto-recovery-runtime",
    "bounded-recovery-acceptance-harness"
  ],
  "active_batch": null,
  "worker": "track1-recovery"
}
```
