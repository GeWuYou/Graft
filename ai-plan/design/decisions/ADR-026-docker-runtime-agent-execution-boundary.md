# ADR-026: Docker Runtime Agent Execution Boundary

- Status: accepted
- Date: 2026-08-21
- Scope: Docker-backed Application, Container, Build and platform-update execution; Task Runtime external execution; Runtime Target Agent transport
- Supersedes in part: ADR-006 and ADR-009 only where they assume a Compose-CLI runner or server-local Docker execution handoff; their Compose-root, durable update-state, fencing and self-replacement survival decisions remain authoritative
- Extends: ADR-023, ADR-024 and ADR-025

## Context

Graft currently proves Docker target availability with the Moby client while several Application and Build paths invoke
the `docker` executable. Mounting the Docker socket therefore does not prove that the server image contains Docker CLI,
Compose or buildx. This split authority can report a target as online and later fail an operation with an executable-not-
found error. Installing all three CLI packages in the server image would couple business modules to one vendor runtime,
increase the privileged server surface and preserve two incompatible execution paths.

The repository already owns Task/Stage persistence, Runtime Target Agent enrollment and mTLS identity, Provider contracts,
and a short-lived self-update controller. The missing primitive is a Task-owned external execution lease that lets one
provisioned Agent perform a frozen Stage without becoming a second scheduler or task state machine.

## Decision

1. `docker-builder-agent` is migrated directly into one always-on `docker-runtime-agent`. There is no dual-Agent or
   compatibility alias period. The Agent is the only always-on process that mounts `docker.sock`.
2. The server depends on provider-neutral Application, Container, Build and Update contracts. It neither installs Docker
   CLI nor mounts `docker.sock` in the target topology.
3. Task Runtime owns external execution leases, Stage state, bounded logs, cancellation observation, receipt settlement,
   expiry and recovery. The Agent pulls leases through the existing certificate-authenticated listener and never accepts
   server push or owns a queue, retry policy, Task status or business database.
4. A lease is bound to task, stage, attempt, Runtime Target, provider, capability, operation, payload digest, fencing
   token and expiry. The frozen Stage plan contains no argv, Docker endpoint, certificate path, host credential or SDK
   type. Secrets are delivered through a separate operation-scoped credential boundary.
5. A successful receipt for a non-final Stage atomically settles that Stage and makes the next Stage eligible. A failed
   receipt settles the Task as failed. An uncertain receipt or an expired running lease settles the Stage as `unknown`
   and the Task as `needs_attention`. Task Runtime never automatically replays an external Docker side effect whose
   idempotence is not proven.
6. Docker adapters inside the Agent use the Moby SDK for Engine resources, the official Compose SDK for Application
   lifecycle, and an approved Engine/OCI SDK path for Build and Manifest operations. CLI-backed adapters may exist only
   as an explicitly owned migration bridge with a deletion trigger and conformance coverage.
7. Platform self-update remains a separate, digest-pinned short-lived Update Controller. The Agent launches it from a
   Task-owned lease. During that operation only, the Controller may also mount `docker.sock`; it updates server/web,
   verifies the new server, replaces the Agent last, verifies the new Agent mTLS/capability readiness and writes the
   terminal durable update fact.
8. Agent pull remains the sole work feed. A later transport-only, Agent-initiated interactive channel for Container
   logs/exec may be designed separately; it cannot claim leases, settle Tasks or become a streaming work queue.

## Authority Boundaries

| Concern | Canonical owner |
| --- | --- |
| Task, Stage, attempt, lease, log, receipt, cancellation and recovery state | Task Runtime |
| Agent identity, generation, target binding, mTLS authentication and capability projection | Runtime Target |
| Application lifecycle intent and immutable operation snapshot | Application/Project module |
| Container resource semantics and authorization | Container module |
| Build plan, placement, reservation, artifact and publication | Build module |
| Provider SDK translation and Docker side effects | Docker Runtime Agent provider adapters |
| Update durable state, fencing and survival across server/Agent replacement | short-lived Update Controller |

## Consequences

- Server target health can no longer be confused with local CLI availability; execution readiness also requires an
  active Agent generation and the requested capability.
- Task Runtime gains a reusable external-execution primitive without gaining Docker knowledge.
- The Runtime Agent binary may be larger than the current server binary, but the deployment removes the unpacked Docker
  CLI, Compose and buildx packages and keeps privileged dependencies out of server.
- Existing CLI-backed non-terminal work must be drained or explicitly cancelled before the final cutover. There is no
  silent fallback to server-local Docker execution.
- ADR-006 and ADR-009 continue to govern Compose-root trust, durable update state, fencing and the need for an executor
  that survives server recreation. ADR-026 changes the launcher and SDK boundary, not those recovery invariants.

## Rejected Alternatives

- Install Docker CLI, Compose and buildx in server: fixes one image symptom while retaining vendor coupling and socket
  privilege in the main process.
- Embed Compose SDK directly in server: reduces packages but keeps the server privileged and makes self-replacement and
  host-path semantics unsafe.
- Add a second Agent queue or Agent-owned task table: duplicates Task Runtime authority and creates irreconcilable retry
  and recovery state.
- Merge Update Controller into the always-on Agent: the controller would terminate while replacing itself and could not
  durably prove the terminal update result.
