# ADR-009: Self Update Controller State Authority

- Status: accepted
- Date: 2026-08-01
- Scope: `graft-compose-runner`, self-update state ownership, server projection, and Update Center realtime recovery
- Supersedes: the server-owned lifecycle, runner log receipt, and Task Runtime settlement portions of ADR-006

## Context

An official Compose self-update recreates `server` and may temporarily make its API and realtime gateway unavailable.
The former model made `server` persist running stage state and later settle a marker-bounded runner log receipt. That
makes the upgraded process the owner of work it cannot observe while it is stopped: the browser can remain on an old
phase even after the runner completes.

This is a control-plane ownership error, not a missing operation field. The executor that survives the controlled
service replacement must own execution state. Existing Deployment Runtime and Compose trust boundaries remain valid:
only a frozen Docker-daemon-host Compose root is executable, and the runner receives only server-validated,
operation-scoped input.

## Decision

`graft-compose-runner` is the short-lived Self Update Controller and the sole lifecycle owner for one self-update
operation. It starts after `server` accepts and persists an authorization/auditable user request, but that request is
not an execution phase. The runner writes the first authoritative execution state and continues independently while
`server` and `web` are recreated.

The official Compose installation provides one named update-state volume. The runner is its only writer; `server`
mounts it read-only. The volume stores a versioned atomic `current.json` snapshot and operation-scoped append-only
event records. Snapshot writes use a temporary file, durable flush, and atomic rename; each update has a monotonically
increasing revision. Event records contain the operation binding, sequence, timestamp, phase transition, and snapshot
integrity digest so a restarted server can validate and reconstruct the latest durable state. The state store never
contains database credentials, backup locations or contents, host paths, arbitrary command output, Docker stderr, or
secrets.

The authoritative state includes `operation`, `phase`, `progress`, safe `message`, `started_at`, `finished_at`,
safe structured `error`, and `runner_id`, plus version/schema/revision and the frozen target/deployment snapshot
identity needed to bind it to the accepted request. Phases are `READY`, `PREFLIGHT`, `BACKUP`, `PULL_IMAGES`,
`STOP_SERVICES`, `APPLY_UPDATE`, `MIGRATION`, `START_SERVICES`, `HEALTH_CHECK`, `SUCCESS`, `FAILED`, and
`ROLLBACK`. The controller records every transition before taking the associated irreversible step; `SUCCESS`,
`FAILED`, and `ROLLBACK` are terminal.

`server` is a read-only projection and delivery boundary:

- It authorizes the request, freezes trusted runner input, starts the runner, and records the user request/audit fact.
- It validates state schema, operation binding, monotonic revision, and event/snapshot integrity before serving it as
  the active operation API or publishing a sanitized snapshot on the existing realtime topic.
- On startup and while running it reconciles the read-only state store. SSE/WebSocket is transport only; a reconnect
  first reads the latest API snapshot and then subscribes, so unavailable server time is not mistaken for lost runner
  progress.
- It projects only verified terminal state into PostgreSQL as append-only update history, backup/audit references, and
  safe diagnostics. PostgreSQL is the business-history authority after verified terminal projection, never the
  authority for active runner phase or progress.

The runner has no PostgreSQL credentials and no runner HTTP, SSE, or WebSocket service. It acquires a state-volume
lease and heartbeat before execution so only one non-terminal operation exists. A stale lease, invalid state, or
integrity mismatch fails closed. A stale non-terminal operation is recovered only by a newly authorized manual
recovery runner, which reads the prior state and writes its own bound recovery transition; `server` must not mutate a
runner phase to simulate recovery.

The controlled order is `PREFLIGHT -> BACKUP -> PULL_IMAGES -> STOP_SERVICES -> APPLY_UPDATE -> MIGRATION ->
START_SERVICES -> HEALTH_CHECK -> terminal`. Before migration begins, a failure may restore the configuration/image
snapshot and conclude `ROLLBACK`; after migration starts, no automatic database rollback or restore is permitted and
failure concludes `FAILED` with operator recovery evidence.

## Consequences

The Update Center can lose its server connection during a recreate without losing the execution fact. Once the new
server starts, it observes the current state or terminal result and resumes API/SSE delivery. Retained runner logs may
remain diagnostic evidence, but they are not status authority and cannot settle an update operation.

The prior `update_operations` running-state and Task Runtime receipt-settlement model must be replaced by a server
request record plus verified terminal history projection. The public update contract must expose the runner snapshot
for an active operation and terminal history for completed operations without preserving old lifecycle aliases.

The additional named volume and recovery controller increase Compose deployment requirements, but avoid a persistent
agent and keep high-privilege Docker execution inside the existing short-lived runner boundary. ADR-006 continues to
govern manifest-pinned runner publication, immutable target verification, backup/migration ordering, and Compose-root
trust limits except where this ADR explicitly supersedes lifecycle state ownership.

## Rejected Alternatives

- Adding more server-owned operation fields or polling runner logs: server remains unavailable during its own
  replacement, so neither is an execution authority.
- Letting the runner write PostgreSQL directly: it expands credentials and business-data authority across the Docker
  control boundary without solving active-state durability better than the state volume.
- Making the browser consume the runner directly: it exposes a privileged executor and bypasses server authorization,
  state validation, and existing realtime controls.
- A permanent update agent: this refactor requires independent execution, not a second long-lived runtime version.
