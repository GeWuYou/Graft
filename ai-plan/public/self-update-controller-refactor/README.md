# Self Update Controller Refactor

## Current Status Summary

- Topic objective: move active self-update lifecycle authority from `server` to `graft-compose-runner` so progress
  and recovery survive server replacement.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: a long-running control-plane refactor requires a new ADR, release design/roadmap convergence, and
  bounded runner, server/OpenAPI, Web, Compose, and validation batches.
- Canonical authority: ADR-009, the platform self-update design, official Compose deployment contract, and frozen
  Deployment Runtime snapshots.
- Completed so far: Work Intake bootstrap, ADR-009, design/roadmap authority convergence, runner state-store and
  controller lifecycle, Compose state-volume integration, server projection/API/realtime convergence, and Update
  Center recovery rendering, and durable lease/fencing convergence.
- In progress: cross-boundary validation and archive-readiness review.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: schema-v2 runner state-volume lease is active/recovery authority; server validates and projects
  it read-only, and PostgreSQL owns verified terminal business history.

## Owned Scope

- self-update runner controller, official Compose state-volume contract, and runner/server projection protocol
- update server/OpenAPI/realtime contracts, Update Center recovery behavior, migrations, and focused validation
- `ai-plan/design/release/**`, `ai-plan/design/decisions/**`, `ai-plan/roadmap/**`, and this topic's recovery records

Out of scope:

- a persistent update agent, runner database credentials, runner public HTTP/realtime endpoints, or direct browser
  access to runner state
- automatic database rollback/restore after migration, multi-node orchestration, Kubernetes execution, or binary
  self-replacement

## Locked Decisions

1. `graft-compose-runner` is the only self-update lifecycle owner; it writes versioned atomic snapshots and sparse
   append events to a named state volume. Schema v2 fences writes with `lease_epoch` and renews its lease every 30
   seconds with five-minute expiry, while `server` mounts the volume read-only.
2. `server` accepts authorized requests and exposes verified state through API/realtime, but active phase/progress and
   recovery transitions never originate from `server` or PostgreSQL.
3. PostgreSQL receives only idempotent, verified terminal history/audit/backup projections; runner never receives DB
   credentials.

## Current Recovery Point

- ADR-009 replaces only ADR-006's server-owned lifecycle/log-receipt premise and preserves Compose trust boundaries.
- Docker container existence, exit state and inventory no longer decide liveness: an expired v2 lease, missing first
  state after five minutes, or v1's 30-minute bridge projects `runner_lost`; recovery is only pre-migration and must
  conclude terminally.
- Durable lease/fencing convergence is complete across runner, server/OpenAPI, and Update Center.
- Next step: rerun the required cross-boundary validation and archive-readiness review. Do not restart an already-completed batch.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is in `todos/self-update-controller-refactor-tracking.md`.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/self-update-controller-refactor/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
