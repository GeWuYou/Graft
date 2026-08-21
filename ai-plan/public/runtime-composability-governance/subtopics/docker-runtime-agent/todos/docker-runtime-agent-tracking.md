# Docker Runtime Agent Tracking

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: subtopic `runtime-composability-governance/docker-runtime-agent`
- authority summary: Task Runtime owns external execution lifecycle; Runtime Target owns Agent identity and capability
  binding; Docker Runtime Agent executes Provider side effects; Update Controller owns self-update state across replacement.

## Work Intake Result

- classification: long-running refactor
- existing topic reused: `runtime-composability-governance`
- artifacts required: repository design updates, ADR-026, roadmap extension, bounded subtopic recovery and phased implementation
- parallel top-level topic: rejected
- implementation engine: direct bounded batches with validation and scoped commit after each accepted batch

## Semantic Review Selection

- platform architecture: authority, runtime privilege and Task/Agent/Controller boundaries
- module architecture: narrow Provider/Gateway contracts and dependency direction
- domain model: Stage attempt, lease, receipt, cancellation and recovery invariants
- event contract: durable facts versus realtime notifications
- table design and SQL migration: Task-owned lease persistence and constraints
- test seam: repository/Runtime behavior, restart and expiry coverage
- consistency and delete review: stable vocabulary and removal of final-stage/CLI assumptions
- cross-boundary/OpenAPI/API DX: deferred until a batch changes published Agent or Web contracts

## Locked Decisions

1. One always-on `docker-runtime-agent`; direct rename from the experimental Builder Agent, no dual Agent or alias.
2. Agent pull is the sole work feed. Agent owns no queue, retry policy, Task state or business persistence.
3. Server ultimately has no Docker socket or Docker/Compose/buildx executable dependency.
4. Task Runtime owns external execution lease, bounded logs, cancellation observation, receipt and expiry recovery.
5. Update Controller remains separate and replaces Runtime Agent last.
6. Docker adapters converge on SDKs; any CLI adapter is a temporary bridge with owner and deletion trigger.

## Batch State

```json
{
  "completed_batches": [
    "batch-1-architecture-authority-and-recovery",
    "batch-2-task-runtime-external-execution-foundation",
    "batch-3-docker-runtime-agent-promotion",
    "batch-4-application-and-container-migration",
    "batch-5-build-sdk-migration",
    "batch-6-update-controller-launch-boundary",
    "batch-7-deployment-and-cli-deletion"
  ],
  "current_batch": "batch-8-ui-and-cross-boundary-convergence",
  "pending_batches": [
    "batch-8-ui-and-cross-boundary-convergence"
  ],
  "next_batch": "batch-8-ui-and-cross-boundary-convergence",
  "closeout_status": "active"
}
```

## Batch 1 Acceptance

- ADR-026 fixes the single Agent, Task-owned lease, server no-socket and independent Update Controller decisions.
- Compose, Build Provider SPI, Agent protocol, self-update and project-layout authority no longer prescribe the old
  server-local CLI or split Builder/Runtime Agent model.
- Parent and subtopic recovery materials identify the same current batch and authority.
- `git diff --check` and `python3 scripts/validate_ai_plan_structure.py` pass.

## Batch 2 Acceptance

- Task Runtime persists one fenced external execution lease per Stage attempt with database-enforced identity and state.
- Provider-neutral APIs support claim, renew, cancellation observation, bounded Stage logs and idempotent receipt.
- External receipt success can advance a non-final Stage without prematurely finalizing the Task.
- Expired running leases and interrupted external work enter `unknown`/`needs_attention`; no unsafe automatic replay.
- Migration has complete comments, preflight metadata and upgrade validation; focused Task tests and backend validation pass.

## Batch 3 Acceptance

- The experimental package, binary, image, configuration, development topology and conformance fixture are promoted
  directly to the single `docker-runtime-agent`; no old runtime entry, alias, process or optional Compose profile remains.
- Runtime Target persists an explicit capability binding and execution claim admits only the authenticated Agent's
  provider, capability and protocol version. The old single profile is copied once and is not an execution fallback.
- The Agent performs mTLS enrollment/reconnect, bounded long-poll claim, lease renewal, cancellation observation,
  fixed redacted logs, fenced receipt settlement, certificate rotation and local readiness without opening an inbound port.
- Application, Container, Build and Update Docker operations remain at their pre-Batch-3 owners for later migration.
- Focused Agent/transport/Runtime Target/Task behavior tests, Agent race tests, migration gates and full backend validation pass.

## Batch 4 Acceptance

- Application Compose lifecycle and finite Container mutations now submit provider-neutral intent as ordered external
  Stages and reach the Docker Runtime Agent only through Task-owned fenced leases. Application and Container retain
  domain intent and result interpretation; Task Runtime remains the sole owner of Task, Stage, lease, renewal,
  cancellation, logs, receipt, retry and recovery state.
- Runtime Target capability bindings are generation-scoped. Claim and every post-claim operation re-authorize the
  frozen provider, capability and version against the authenticated active generation, so old credentials, missing
  capabilities and version mismatches fail closed.
- The Agent strictly dispatches the Application and Container operation allowlists through the Moby and Compose SDKs,
  persists only a mode-0600 recovery journal, emits fixed redacted diagnostics and resolves workspace material through
  a lease-fenced transient endpoint that never writes paths, endpoints, credentials or commands to Task persistence.
- Server-local Application and Container mutation execution, CLI command contracts, adapters, fallbacks and
  compatibility aliases are removed. Container snapshot reads and interactive streams remain outside the Task lease
  protocol as required by ADR-026; Build SDK and Update Controller execution remain unmigrated, so the server Docker
  socket mount is intentionally retained.
- Focused Application, Container, Task, Runtime Target, HTTP transport and Agent behavior tests, related race tests,
  OpenAPI/generated-Web checks, migration gates, full Web validation and the complete backend validation entrypoint pass.

## Batch 6 Acceptance

### Launch-boundary inventory

| Entry | Batch 6 disposition | Task Stage / capability | Intent and result owner |
| --- | --- | --- | --- |
| Normal digest-pinned Update Controller start (`ComposeRunnerLauncher.Launch`) | migrated; server method deleted | `controller_launch`; `docker/update_controller@docker/v1`; `platform.update.controller.launch.v1` | Update freezes version/image/operation intent and maps Task failure or controller receipt; Task Runtime owns launch lease/receipt state; Agent owns Docker start |
| Controller terminal handshake | retained as the only final Stage, not a launch fallback | `controller_result`; `compose-runner/v2` final receipt | Controller state volume is durable evidence; Update interprets it; Task Runtime owns final receipt settlement |
| Pre-migration recovery controller (`LaunchRecovery`) | explicitly not migrated in this batch | no new Stage or capability added | Update recovery claim/state invariants remain authoritative; Batch 7 owns removal or migration of the retained Docker recovery boundary |
| Controller-internal Docker/Compose CLI sequence | not a controller startup entry and unchanged | none | Update Controller still owns backup/pull/migrate/recreate/health behavior; later CLI-deletion work must not reopen normal launch |
| Bootstrap `graft update cutover-v1` command | not a controller startup entry and unchanged | none | Deployment bootstrap/cutover compatibility authority remains outside Batch 6 |

- [x] Update submits a two-stage Task plan: `controller_launch` is a fenced Runtime Agent execution stage bound to the
  selected Runtime Target generation, while `controller_result` remains the sole final receipt handshake for the
  durable Update Controller state volume.
- [x] Update owns update intent, version/image semantics, transient material resolution and receipt interpretation;
  Task Runtime owns Task/Stage/lease renewal, cancellation, bounded logs, receipt settlement, retry and recovery.
- [x] `docker-runtime-agent` declares `update_controller` at `docker/v1`, claims only the exact frozen operation and
  protocol, and starts the digest-pinned controller with fixed mounts, no network, a read-only rootfs and stable
  failure codes. The Agent has no inbound listener and does not persist material, credentials, endpoints, host paths or
  commands.
- [x] Runtime Target local Docker capability declarations include `update_controller`; generation and post-claim
  re-authorization remain enforced by the shared Agent transport for version mismatch, capability removal, certificate
  rotation and reconnect recovery.
- [x] The server-side normal launch method was removed from the old Docker launcher. Container read/stream/interactive
  paths, Runtime Target discovery/summary and the bounded recovery observer remain on the retained server Docker socket
  boundary for later batches; no fallback, dual execution or compatibility alias was added.
- [x] Focused Update/Agent/Runtime Target tests, full Go tests, race coverage and backend validation evidence are
  recorded by the main agent.

Evidence: focused Update/Agent/Runtime Target/Task tests, the same packages under `go test -race`, production
`golangci-lint`, `python3 scripts/validate_ai_plan_structure.py`, `git diff --check`, and the complete
`go run ./cmd/graft validate backend` entrypoint all passed.

## Next Recovery Point

Batch 6 Update Controller launch-boundary migration is accepted. The next bounded slice is Batch 7, which handles the
remaining deployment/CLI cleanup after the launch and recovery contracts are proven. Build remains provider-neutral:
Task Runtime owns its fenced lease, renewal, cancellation, logs, transient result digest, receipt, retry and recovery;
Runtime Target binds `docker/v1`; and `docker-runtime-agent` performs Moby/OCI SDK side effects. Neither material nor
result JSON is durable Task/Agent state.

The server Docker socket remains only for the retained Update recovery/observation boundary, Runtime Target
discovery/summary and explicitly unmigrated Container read/stream/interactive capabilities. Batch 7 must not introduce
a second launch path while removing the remaining server-local recovery/CLI dependency.

## Batch 5 Acceptance

- [x] Every Build Docker operation reaches the Agent through a Task-owned external execution lease and the exact
  provider/capability/version binding.
- [x] Material and normalized result are transient, fence-bound and redacted; Task retains only `result_sha256`, and
  Build alone owns Artifact/Publication/result interpretation.
- [x] Agent SDK tests cover capability admission, Moby/OCI build/push/manifest/copy behavior, cancellation, retry and
  recovery; duplicate result replay is idempotent and conflicting replay is rejected.
- [x] No Build server-local Docker/CLI execution, fallback, old adapter or compatibility alias remains.
- [x] Deployment/conformance docs describe the shared snapshot volume and no Agent inbound port; server socket retention
  is explicitly limited to unmigrated Update/Container read-stream boundaries.
- [x] Focused, race, migration, AI-plan and complete backend validation evidence is recorded by the main agent.

Evidence: focused and race Go tests, `go run ./cmd/graft validate backend` (including lint), SQL migration/version
checks, generated module registry freshness, AI-plan structure validation and `git diff --check` all passed.

## Batch 7 Deployment and CLI Deletion Evidence

- [x] Official Compose bootstrap and smoke overlay retain `graft update cutover-v1` as the one-time legacy update-state
  authority before `graft migrate up`; cutover is not a runtime launch or compatibility execution path.
- [x] Deployment and conformance documentation state the only retained server Docker socket consumers: Update
  observation/recovery, Runtime Target discovery/summary, and Container snapshot/stream/interactive transport. Build,
  Update Controller launch, and finite mutation side effects remain Agent-owned.
- [x] Official migration guidance fixes the ordered Update rollout: replace `server`/`web`, verify server health,
  replace the Runtime Agent last, then verify mTLS identity, active generation, and capability readiness.
- [x] Conformance fixture documentation and Compose comments preserve the outbound-only Agent/no inbound port boundary
  and the retained server socket list.

Validation evidence for this documentation/configuration slice: `git diff --check` and
`python3 scripts/validate_ai_plan_structure.py`.
