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
    "batch-4-application-and-container-migration"
  ],
  "current_batch": "batch-5-build-sdk-migration",
  "pending_batches": [
    "batch-5-build-sdk-migration",
    "batch-6-update-controller-launch-boundary",
    "batch-7-deployment-and-cli-deletion",
    "batch-8-ui-and-cross-boundary-convergence"
  ],
  "next_batch": "batch-5-build-sdk-migration",
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

## Next Recovery Point

Batch 4 migrated Application Compose lifecycle and finite Container mutations to the single Docker Runtime Agent through
Task-owned external execution leases, generation-scoped capability binding and transient material resolution, while
removing their server-local mutation paths. Begin Batch 5 by migrating Build SDK execution only; do not begin Update
Controller migration or remove Docker socket access still required by explicitly unmigrated/runtime-stream boundaries.
