# Self Update Controller Refactor Trace

## 2026-08-01 Work Intake And Design Authority

- Classified the self-update control-plane correction as a long-running cross-boundary refactor. The prior
  `platform-self-update` and `compose-update-policy-reset` topics are archive-ready and do not own this new work.
- Created the minimum Work Contract bootstrap: one active topic, ADR-009, release design convergence, and roadmap
  convergence.
- ADR-009 makes `graft-compose-runner` the active lifecycle controller. A named state volume stores a versioned,
  atomic snapshot and append-only events; runner is the only writer and server reads it only after validation.
- Preserved ADR-006/ADR-008 Deployment Runtime and Compose-root trust boundaries. Replaced only the server-owned
  lifecycle, runner log receipt, and Task Runtime settlement premise.

## Locked Decisions

- Server persists user request/audit initiation and only projects verified terminal runner state into PostgreSQL
  business history; it does not own active phase, progress, or recovery transitions.
- Runner has no PostgreSQL credentials and no public API. Existing server realtime transport relays sanitized verified
  snapshots and reconnecting clients read current state before subscribing.
- A stale non-terminal operation is recovered by an authorized manual recovery runner, not by a server write or an
  automatic database rollback.

## 2026-08-01 Implementation And Validation Recovery Point

- Completed the runner state-store/controller, Compose state-volume integration, server projection/history/API/realtime
  work, and Update Center recovery rendering described by the first four implementation batches.
- Moved the active batch to `cross-boundary-validation-and-archive-readiness`; the implementation batches must not be
  restarted solely because older recovery material still listed them as pending.
- PR #237 completed web, contract-governance, migration-governance, and static security checks. Backend runner-state
  tests currently fail where the test state-root setup invokes `chown`, which the execution environment rejects;
  resolve that validation issue and rerun the required cross-boundary checks before archive readiness.

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
