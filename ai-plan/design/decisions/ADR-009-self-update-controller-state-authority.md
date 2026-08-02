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

Event records are a user-visible, allowlisted action timeline rather than runner output. Each record is bound to one
operation and has a monotonic revision, timestamp, phase, stable action code, and safe localized-message input.
The server validates and may replay these records through the existing operation topic. Command lines, Compose or
Docker stderr, host paths, backup locations, credentials, and arbitrary metadata are never event fields. Event files
are operation-scoped so a revision from one operation cannot overwrite or be replayed as another operation's event.

`server` is a read-only projection and delivery boundary:

- It authorizes the request, freezes trusted runner input, starts the runner, and records the user request/audit fact.
- It validates state schema, operation binding, monotonic revision, and event/snapshot integrity before serving it as
  the active operation API or publishing a sanitized snapshot on the existing realtime topic.
- On startup and while running it reconciles the read-only state store. The active-operation and bounded event-replay
  APIs are read-only projections; a reconnect or new browser tab first reads the latest snapshot and missed events,
  then subscribes. SSE/WebSocket is transport only, so unavailable server time is not mistaken for lost runner
  progress.
- It projects only verified terminal state into PostgreSQL as append-only update history, backup/audit references, and
  safe diagnostics. PostgreSQL is the business-history authority after verified terminal projection, never the
  authority for active runner phase or progress.

The runner has no PostgreSQL credentials and no runner HTTP, SSE, or WebSocket service. The state store enforces one
non-terminal operation by rejecting a write for a different `OperationID` while the prior snapshot is non-terminal.
Invalid state or an integrity mismatch fails closed. A stale non-terminal operation is recovered only by a newly
authorized manual recovery runner, which reads the prior state and writes its own bound recovery transition; `server`
must not mutate a runner phase to simulate recovery.

When Docker proves that the runner bound to a non-terminal snapshot has exited, `server` must not continue to project
that snapshot as live execution. It preserves the last verified `phase`, `progress`, and safe `message` solely as
diagnostic context, projects `state_source=runner_terminated`, `state_available=false`, and the stable safe error
`PLATFORM_UPDATE_RUNNER_TERMINATED`, then persists one allowlisted operation diagnostic. Raw container logs, paths,
commands, exit-detail text, and secrets remain host-operator evidence and never cross the API boundary. The browser
must treat this projection as terminal for its current session rather than polling indefinitely.

Recovery is an explicit `platform-update.manage` action, exposed as `POST /api/platform/updates/operations/{operationID}/recovery`.
It can launch exactly one recovery runner only when the recorded runner identity matches an exited container and the
verified snapshot is non-terminal and pre-migration. The recovery runner concludes the interrupted operation with a
safe terminal result; it never resumes the upgrade. Running, terminal, mismatched, already-recovered, unavailable,
or post-migration operations fail closed. After that terminal result is projected, the normal new-update path may
evaluate eligibility again.

Before the potentially slow Docker image pull, server atomically records an opaque recovery-launch coordination claim
on the request record. This is authorization and duplicate-launch evidence only, not a runner phase or progress
projection. A claim is released only when Docker is proven not to have created a recovery container; after container
creation is attempted it remains durable so retries fail closed until the recovery runner publishes its terminal
result.

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
