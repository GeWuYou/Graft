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
    "batch-2-task-runtime-external-execution-foundation"
  ],
  "current_batch": "batch-3-docker-runtime-agent-promotion",
  "pending_batches": [
    "batch-3-docker-runtime-agent-promotion",
    "batch-4-application-and-container-migration",
    "batch-5-build-sdk-migration",
    "batch-6-update-controller-launch-boundary",
    "batch-7-deployment-and-cli-deletion",
    "batch-8-ui-and-cross-boundary-convergence"
  ],
  "next_batch": "batch-3-docker-runtime-agent-promotion",
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

## Next Recovery Point

Batch 2 added the provider-neutral Runtime Agent gateway, Task-owned Stage-attempt lease/receipt persistence, atomic
Agent claim, bounded logs, cancellation observation, restart survival, expiry recovery and retry fencing. Begin Batch 3
by directly promoting the experimental Docker Builder Agent into the single Docker Runtime Agent; do not retain two
Agent binaries or add aliases, and do not migrate Application/Container operations ahead of that promotion batch.
