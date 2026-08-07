# Build Domain v2 Roadmap

## Delivery Rule

This roadmap implements [Build Domain v2 Credential And Telemetry Authority RFC](../design/architecture/build-domain-v2-credential-and-telemetry-authority.md)
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
the exposed feature set ahead of these gates. `least_load`, `affinity` and `region` remain latent/disabled until Phase 4.

## Phase 1: Single Builder Authority

- Expose one manually selected, authorized Builder Instance backed by Runtime Target capability discovery.
- Derive `BuildCapabilityRequirement` from frozen Template, Driver, platform, cache, Destination and security policy.
  Use `CapabilityMatcher` to return the eligible Instance or auditable deny reason.
- Freeze manual Placement Evidence and atomically acquire a short fenced `BuilderReservation` before Task acceptance.
  Use `Reserved`, `Accepted`, `Running`, `Released`, `Expired` and `Abandoned` semantics; each retry uses a new fence.
- Introduce `CredentialProvider` and `RuntimeExecutionAdapter` for every Registry Push. Registry Connection selects only
  `credential_ref`; adapters create isolated per-operation credentials and remove them on every terminal or recovery
  path.
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

- Implement `BuilderTelemetryProvider` beneath `RuntimeTargetBuilderTelemetryReader`. Require Builder-scoped running
  builds, queue length, allocatable slots, health, capability profile, source identity, `ObservedAt`, `ExpiresAt`,
  integrity and unsupported dimensions.
- Add Builder Agent/control-plane or equivalent telemetry authority for Docker; use bounded BuildKit/Kaniko controller
  or workload facts for Kubernetes. Never use Docker Engine metrics, host metrics or whole-cluster capacity as a Builder
  source.
- Enable `LeastLoad`, `Capacity` and `Affinity` only for fresh, conformant Provider observations paired with a valid
  Reservation. Recheck the same frozen Placement before an infrastructure retry; do not silently choose another target.
- Deliver Task Runtime-owned distributed legs, cancellation, retries and recovery. Build maps derived `BuildEvent`
  vocabulary to Task facts and settles immutable platform Artifacts and final manifest Publication.
- Treat `Abandoned` Reservation, credential cleanup failure and unknown external outcomes as `Needs Attention` until
  recovery proves a terminal result.

**Release gate:** dynamic decisions replay from capability profile, fresh telemetry and Reservation fence; Provider
conformance and recovery cover cancellation, timeout, restart and cleanup; and distributed execution does not create a
second Build queue, Task state machine or event store.

## Cross-Phase Constraints

- Do not create a Registry server, default Docker credential-store dependency, generic plugin/OPA scheduler, second
  Runtime Target connection model, global health registry, Build worker/queue/log runtime or evidence database.
- Build Events are derived Task Runtime facts; `task_events` must not duplicate Task/Stage lifecycle records.
- Artifact identity is an immutable digest. Tags, manifests, promotions and copies are Publication facts.
- Provider Capability Version is schedule-relevant; Driver Version is diagnostic only.
- Dynamic policy failure maps to Task Runtime's existing taxonomy: transient uses bounded retry, permanent/configuration/
  authorization do not retry, infrastructure re-conforms the same placement, Provider disables locally, Internal is an
  incident, and Unknown requires attention without guessed success or rescheduling.

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
