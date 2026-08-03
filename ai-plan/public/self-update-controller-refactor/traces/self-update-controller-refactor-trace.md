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

## 2026-08-01 Production Beta Repair

- Beta tracking reported an accepted operation permanently projected as `READY/0%` with `runner_starting`, while a
  new tab could enter the application and terminal-only history remained empty. The likely upstream failure is the
  runner state-volume initialization path: the runner is intentionally root but drops all Linux capabilities while
  the state store changes ownership to its unprivileged state user.
- The repair grants only the required ownership-changing capability while retaining the existing drop-all baseline,
  and treats a request without validated runner state as an explicit unavailable source rather than synthetic
  progress.
- The same bounded repair adds operation-scoped, allowlisted action events. Server validates and replays events via
  the existing update realtime topic and read-only API; Web recovers active state and missed revisions on a new tab or
  reconnect. Raw Compose/Docker output, paths, backup locations, credentials, and arbitrary runner diagnostics stay
  outside this contract.
- Validation passed: `go run ./cmd/graft validate backend`, `just openapi-check`, Web `bun run typecheck`, and 22
  focused Update Center tests. The full Web entrypoint was also run but remains blocked by unrelated Container
  saved-view lint, i18n, and unused-export failures already present in the worktree.

## 2026-08-03 Durable Lease And Fencing Authority

- Replaced Docker container observability as active-state authority with the runner-owned state-volume durable lease.
  Every schema-v2 non-terminal snapshot carries `lease_epoch`, `lease_heartbeat_at`, and `lease_expires_at`; the
  runner renews every 30 seconds and a lease expires after five minutes. Heartbeat revises only `current.json` and
  does not create an action event or UI phase.
- Phase, heartbeat, and terminal writes share fencing: only the matching `runner_id + lease_epoch` can write while
  the lease is unexpired. Expiry prevents the old runner from reviving or overwriting state; recovery increments epoch
  and writes only the safe terminal conclusion.
- Server reconciles leases each minute and computes them on API reads. Expired v2 state, a first snapshot missing for
  five minutes after authorization, and v1 state beyond the 15-minute execution plus 15-minute grace bridge project
  `runner_lost` with last verified progress as diagnostics and `state_available=false`.
- Recovery now requires a pre-migration `runner_lost` projection. It receives the bound snapshot when present, or
  authorization identity plus frozen version/deployment input when first state is missing; server never fabricates a
  phase. Docker inventory, container existence, exit code, and logs are not recovery or liveness authority. Keep
  `runner_terminated` consumption only until all minimum-supported sources create schema-v2 lease snapshots.

## 2026-08-03 PR Review Remediation

- Added a bounded consecutive-heartbeat failure threshold that cancels the runner execution context after state-volume
  lease renewal can no longer be trusted, with a direct regression test for cancellation.
- Aligned OpenAPI recovery descriptions, generated web bindings, Update Center recovery eligibility, and active-topic
  recovery materials with missing-state `runner_lost` behavior; legacy `runner_terminated` remains diagnostic-only.
- Validation passed: `go run ./cmd/graft validate backend`, `just openapi-check`, and `bun run check`.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "work-intake-and-design-authority",
    "runner-state-controller-foundation",
    "compose-state-volume-and-lifecycle-integration",
    "server-projection-history-api-and-realtime",
    "update-center-recovery-rendering",
    "durable-lease-fencing-and-runner-lost-convergence"
  ],
  "pending_batches": [
    "cross-boundary-validation-and-archive-readiness"
  ],
  "current_batch": "cross-boundary-validation-and-archive-readiness",
  "next_batch": null,
  "closeout_status": "active"
}
```
