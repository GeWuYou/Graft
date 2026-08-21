# Build Domain v2 Roadmap

## Delivery Rule

This roadmap implements [Build Domain v2 Credential And Telemetry Authority RFC](../design/architecture/build-domain-v2-credential-and-telemetry-authority.md)
and its provider integration surface in the [Provider SDK And SPI RFC](../design/architecture/build-domain-v2-provider-sdk-spi.md)
without replacing the existing immutable chain:

```text
Workspace -> Workspace Snapshot -> Execution Plan -> Task execution -> Artifact -> Publication
```

Task Runtime retains execution, cancellation, retry, recovery and logs. Runtime Target retains target connection and
provider entry. Build owns logical Builder resources, capability matching, Placement Evidence and Reservations. Registry
Connection stores only `credential_ref`; Secret and short-lived credential issuance never cross into Build or Task
metadata. Registry/Builder-local failure must not alter platform-wide availability.

## Capability Matrix

| Capability | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
| --- | ---: | ---: | ---: | ---: |
| single Builder / manual selection | yes | yes | yes | yes |
| runtime capability discovery / matcher | yes | yes | yes | yes |
| secure Registry credential execution | yes | yes | yes | yes |
| fenced Reservation lifecycle | yes | yes | yes | yes |
| Driver, Template, Workspace materialization | basic | yes | yes | yes |
| Pool / RoundRobin / Random | no | no | yes | yes |
| telemetry | no | no | static diagnostics | dynamic authority |
| LeastLoad / Capacity / Affinity | no | no | no | yes |
| Builder Agent / distributed Build | no | no | no | yes |

Existing Pool, Placement and provider-conformance code is retained as future implementation material. It does not widen
the exposed feature set ahead of these gates. `least_load`, `capacity`, `affinity` and `region` remain latent/disabled until Phase 4.

## Phase 1: Single Builder Authority

- Expose one manually selected, authorized Builder Instance backed by Runtime Target capability discovery.
- Derive `BuildCapabilityRequirement` from frozen Template, Driver, platform, cache, Destination and security policy.
  Use `CapabilityMatcher` to return the eligible Instance or auditable deny reason.
- Freeze manual Placement Evidence and atomically acquire a short fenced `BuilderReservation` before Task acceptance.
  Use `Reserved`, `Accepted`, `Running`, `Released`, `Expired` and `Abandoned` semantics; each retry uses a new fence.
- Introduce `CredentialProvider` and the fenced Build material/Agent SDK boundary for every Registry Push. Registry
  Connection selects only `credential_ref`; the resolver and Agent create isolated per-operation credentials and remove
  them on every terminal or recovery path.
- Phase 1 deployment uses the optional absolute `GRAFT_REGISTRY_CREDENTIALS_FILE` only as a mounted secret-file
location. The core provider reloads scoped, expiring entries and returns opaque sessions; Build domain records, Registry
  records, Task Runtime and all external projections remain credential-plaintext-free. An unset or invalid source omits or
  blocks adapter admission and fails closed.
- Prohibit default Docker credential-store reads/writes and all environment-default authentication fallback. Preserve
  historical `docker-runtime-store` records as read-only evidence.
- Preserve existing Snapshot materialization, Artifact identity and Publication lifecycle.

**Release gate:** one compatible Builder can build and publish with a fresh, isolated credential; all missing,
expired, mismatched, unsupported or uncleared credential paths fail closed with redacted evidence; the accepted
Reservation and capability result are replayable; and local failure does not affect platform availability.

## Phase 2: Build Intent And Materialization Completion

- Complete explicit Builder Profile, Instance, Driver and Template contracts and their capability requirements.
- Complete `WorkspaceMaterializer` strategies and execution cleanup. Reuse, clone, copy-on-write and cache are keyed
  by immutable Snapshot identity and are isolated per execution.
- Extend provider conformance for capability version, credential injection, Snapshot delivery, cancellation/recovery,
  cleanup and redacted evidence. Non-conformant providers remain manual/static or unavailable.
- Add immutable Artifact-linked SBOM, provenance, SLSA, Cosign and attestation evidence as needed; do not add a second
  Artifact model or evidence database.

**Release gate:** all released Template/Driver/Workspace paths materialize an exact immutable Snapshot, prove delivery,
clean up execution bytes safely, and return digest-addressed Artifact evidence without secret, path or endpoint leakage.

## Phase 3: Static Pool Placement

- Open Build-owned Pool membership with `Manual`, persisted `RoundRobin`, and deterministic seeded `Random` selection.
- Treat labels as static eligibility only. Select from authorized Instances after CapabilityMatcher succeeds and retain
  matcher, policy seed/cursor and Reservation fence as Placement Evidence.
- Allow static diagnostic telemetry for operators, but Placement policies in this phase must not read it.
- Keep Pool reads compatible with existing records while limiting new Phase 1/2 writes to a manual single Builder where
  applicable.

**Release gate:** every Pool decision is deterministic and replayable from frozen static evidence; no metric, UI summary,
host value, Task JSON or unproven provider observation changes selection.

## Phase 4: Dynamic Placement And Distributed Build

- Restore Phase 4 as incomplete. A generic signed ingress is not a dynamic telemetry source; historical dynamic policy
  rows remain readable but are non-executable until Provider admission succeeds.
- Implement `BuilderTelemetryProvider` beneath `RuntimeTargetBuilderTelemetryReader` using a provisioned Docker Builder
  Agent bound to its Runtime Target, Provider, Builder scope and capability profile/version. Require monotonic sequence,
  bounded clock skew, replay rejection, report lifecycle, source/provenance/integrity checks and explicit unsupported
  dimensions. Running, queued and slots must come from the Agent's controlled execution ledger or Driver controller,
  never Docker Engine metrics, host metrics, Task JSON, UI data or whole-cluster capacity.
- The Docker Runtime Agent's real SDK build boundary updates the durable Driver-controller ledger before and after execution;
  the current target-only Placement contract permits one enabled Agent scope per target. ADR-023 and ADR-024 govern
  the out-of-process Agent protocol: Runtime Target creates a provider-neutral pending Agent enrollment and peppered
  token verifier, and the active Docker deployment automation creates the bound Docker secret and returns a verified
  delivery receipt. The Agent presents the token plus CSR to Graft's bootstrap listener, and Vault PKI issues only
  after Graft authorizes the exact target/Agent Identity/generation binding. The receipt is delivery evidence only and
  cannot bind or activate the Agent. Docker is the only MVP provider; no unsupported-provider implementation, menu,
  API or placeholder is in scope. Operator and Agent APIs remain unpublished until verified delivery, bootstrap,
  issuance, durable activation, mTLS reconnect, revocation and ledger conformance pass together.
- Require `CapabilityMatcher` before every manual, static, dynamic and distributed-leg Placement. Freeze requirement,
  candidate, profile/version, complete negotiation, policy/version, telemetry observation and Reservation-fence evidence.
- Make Reservation slot-aware: each Build leg claims one explicit capacity unit; atomically compare live reservations
  created after the telemetry observation with Provider `allocatable_slots`. A failed capacity verdict denies the current
  Placement and requires a new placement flow, never an implicit target swap.
- Enable `LeastLoad`, `Capacity` and `Affinity` only after all Provider admission gates pass. `least_load` uses only
  Provider-owned running/queued facts; affinity uses only proven affinity claims. Recheck the same frozen Placement
  before an infrastructure retry; do not silently choose another target.
- Deliver Task Runtime-owned distributed legs, cancellation, retries and recovery. Build maps derived `BuildEvent`
  vocabulary to Task facts and settles immutable platform Artifacts and final manifest Publication.
- Treat `Abandoned` Reservation, `credential_cleanup_unverified` and unknown external outcomes as `Needs Attention`
  until recovery proves a terminal result. Cleanup is `Internal`, never auto-retried, and its reservation/credential
  session cannot be reused.

**Release gate:** Docker Agent conformance proves that a verified automation delivery receipt, one-time bootstrap,
Vault-issued exact SPIFFE identity, durable active generation, mTLS reconnect, revocation and its controlled execution
ledger all fail closed on mismatch or replay. Stale, duplicate, scope-invalid, version-mismatched or unsupported reports
fail closed. Dynamic decisions replay from frozen capability/profile, telemetry observation, policy/version inputs,
negotiation result and Reservation fence; fresh telemetry is only an input to a new decision. Provider conformance and
recovery cover cancellation, timeout, restart and cleanup; and distributed execution does not create a second Build
queue, Task state machine or event store. BuildKit, Kaniko and Kubernetes remain future extensions until each passes
equivalent conformance.

### Batch 5 implementation overlay

The current Docker Build cutover uses Task Runtime external leases and the bound `docker/oci-build` capability at
`docker/v1`. The exact operation allowlist is `build.image.local.v1`, `build.image.publish.v1`,
`build.manifest.publish.v1` and `build.artifact.copy.v1`. Build material and normalized artifact results are transient
fence-bound payloads; Task persists only the result digest and Build owns artifact/publication interpretation. The
server-local Build Docker/CLI path, fallback and compatibility aliases are removed. The shared snapshot volume is
`/tmp/graft-build-snapshots`; Update Controller, Runtime Target discovery/summary and Container read/stream/interactive
boundaries remain the only explicit reasons for the server socket until their later batches.

## Cross-Phase Constraints

- Do not create a Registry server, default Docker credential-store dependency, generic plugin/OPA scheduler, second
  Runtime Target connection model, global health registry, Build worker/queue/log runtime or evidence database.
- Build Events are derived Task Runtime facts; `task_events` must not duplicate Task/Stage lifecycle records.
- Artifact identity is an immutable digest. Tags, manifests, promotions and copies are Publication facts.
- Provider Capability Version is schedule-relevant; Driver Version is diagnostic only.
- Dynamic policy failure maps to Task Runtime's existing taxonomy: transient uses bounded retry, permanent/configuration/
  authorization do not retry, infrastructure re-conforms the same placement, Provider disables locally, Internal is an
  incident, and Unknown requires attention without guessed success or rescheduling.
- New Provider integrations must implement the Provider SDK/SPI conformance surface before entering Matcher or Placement;
  concrete language bindings and registration locations remain implementation follow-ups.

## Migration Order

1. Add Matcher and manual single-Builder evidence over existing Runtime Target capability.
2. Move new Registry publication through Credential Provider and Runtime Execution Adapter; make legacy
   `docker-runtime-store` read-only evidence.
3. Make Reservation lifecycle and recovery association explicit without changing Task lifecycle ownership.
4. Complete Builder Instance, Driver, Template and materialization contracts.
5. Open static Pools in Phase 3; retain older Pools as readable data.
6. Open dynamic policies only after a real Telemetry Provider, Reservation recovery and Task Runtime distributed-leg
   proof. Retries always retain the frozen Plan and Placement.

## Validation Expectations

Implementation batches validate their affected server, OpenAPI and web surfaces using repository entrypoints. The
authority RFC itself is accepted only when credential isolation, cleanup failure, Reservation fencing, stale telemetry,
provider conformance, task recovery and global-vs-local availability behavior each have focused evidence.
