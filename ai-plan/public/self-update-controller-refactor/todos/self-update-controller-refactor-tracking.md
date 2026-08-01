# Self Update Controller Refactor Tracking

## Topic

Self Update Controller Refactor

## Scope

Correct self-update control-plane ownership by making the Compose runner the durable active-state controller and
making server a verified read-only projection, API/realtime delivery, and terminal-history boundary.

## Repository Truth

- `AGENTS.md`
- `ai-plan/design/decisions/ADR-009-self-update-controller-state-authority.md`
- `ai-plan/design/release/platform-self-update.md`
- `ai-plan/design/decisions/ADR-008-deployment-runtime-context.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: official Compose deployment and graft-compose-runner own active self-update execution state; server owns validated projection and PostgreSQL owns verified terminal business history
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/public/self-update-controller-refactor
    - ai-plan/design/release/platform-self-update.md
    - ai-plan/roadmap/platform-self-update.md
    - ai-plan/design/decisions/ADR-009-self-update-controller-state-authority.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Work Intake classified the request as a long-running cross-boundary refactor and created the minimum active-topic
  materials required by the Work Contract.
- ADR-009 is the lifecycle authority: the runner writes active state to a named volume, server only verifies and
  projects it, and database history is terminal-only.
- Runner state durability, recovery behavior, server/UI consumers, and Compose wiring are implemented. The active
  recovery point is cross-boundary validation and archive-readiness review; do not reopen an implementation batch
  unless validation identifies a concrete authority repair.

## Task Checklist

- [x] Work Intake contract, active-topic bootstrap, ADR-009, release-design and roadmap convergence
- [x] runner state store, atomic snapshot/events, operation mutual exclusion, phase controller, and manual recovery
  runner
- [x] official Compose state-volume contract and runner lifecycle integration
- [x] server request admission, read-only projection, terminal-history migration, API, and realtime convergence
- [x] Update Center active-state recovery rendering and localization
- [ ] cross-boundary validation, Compose interruption/restart evidence, and archive-readiness review

## Acceptance Conditions

- During server/web recreation, runner continues durable phase/progress/terminal-state writes without server or
  database access.
- A restarted server reconstructs and validates the latest runner state, then serves the same active or terminal
  result through API and realtime without browser-local lifecycle authority.
- Only verified terminal results enter append-only PostgreSQL history/audit/backup projections, exactly once.
- A failed pre-migration operation can record controlled rollback; post-migration failure never claims automatic
  database rollback and stale work requires the manual recovery controller path.
- Public contracts expose no old server-owned lifecycle aliases or raw runner diagnostics.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "work-intake-and-design-authority",
    "runner-state-controller-foundation",
    "compose-state-volume-and-lifecycle-integration",
    "server-projection-history-api-and-realtime",
    "update-center-recovery-rendering"
  ],
  "pending_batches": [
    "cross-boundary-validation-and-archive-readiness"
  ],
  "current_batch": "cross-boundary-validation-and-archive-readiness",
  "next_batch": null,
  "closeout_status": "active"
}
```

## Validation Milestone

- PR #237 has completed the web, contract-governance, migration-governance, and static security checks for this
  refactor.
- The current backend validation gap is limited to runner-state tests that attempt to change temporary state-root
  ownership with `chown`; the execution environment rejects that operation. Resolve and revalidate this issue before
  archive readiness is assessed.
