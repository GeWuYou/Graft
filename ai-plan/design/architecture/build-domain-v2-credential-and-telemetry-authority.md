# Build Domain v2 Credential And Telemetry Authority RFC

## Status

Accepted design authority for the Build Domain v2 execution path. It complements
[Build Domain v2 Architecture](build-domain-v2.md), which remains authoritative for the immutable Workspace Snapshot,
Execution Plan, Artifact and Publication model.

## Purpose

Registry publication is still coupled to the historical `docker-runtime-store` implementation and Builder Telemetry
currently has only a read contract, not a source Provider. This RFC defines the missing execution authorities without
creating another Task Runtime, scheduler, event bus, global health registry, Registry resource model or evidence store.

It also preserves the platform availability decision: `PlatformAvailabilityStore` and `CapabilityCoordinator` own
platform-wide availability and capability health. A Registry or Builder failure is local to the affected Build
capability and must fail closed there; it must not send the browser to a global service-unavailable page. A future
Agent may project a coarse `feature` diagnostic, but that projection is never Build's execution authority.

## Authority Map

| Fact | Canonical authority | Not an authority |
| --- | --- | --- |
| execution state, cancellation, retry, recovery, logs | Task Runtime | Build Job, Builder Agent |
| target connection, target executability, provider entry | Runtime Target | Builder Instance, Build |
| endpoint, repository policy, `credential_ref` | Registry Connection / Artifact Repository | Build Plan, Docker credential store |
| secret material and credential issuance | Secret Backend / Credential Provider | Registry Connection, Build, Task Runtime |
| Profile, Instance, Pool, capability requirement, placement, reservation | Build Domain | Runtime Target |
| measured Builder capability, health, load | Provider through the narrow Runtime Target reader | UI, Monitor, Docker host metrics |
| Artifact digest and lifecycle | Build Artifact | tag, Build Job |
| platform availability | `PlatformAvailabilityStore` / `CapabilityCoordinator` | Builder Telemetry |

`BuilderIdentity` is immutable associated evidence, not a mutable connection registry: `BuilderInstanceID`,
`RuntimeTargetID`, `ProviderID`, optional `AgentID`, and `ProviderCapabilityVersion`. An Instance is logical, a Target
is a physical connection, and an Agent is a mutable execution participant.

## Build Capability Contract

The following are logical contracts, not persisted domain entities:

- `BuildCapabilityRequirement` is derived from the frozen Template, Driver, platforms, cache, Destination and security
  policy.
- `BuildExecutionCapability` is the versioned, provable profile a Runtime Provider reports for one Builder.
- `CapabilityMatcher` is a pure Build decision boundary. It returns eligible Builder Instances or an auditable deny
  reason.

The vocabulary includes Driver, supported platforms and multi-platform support, cache import/export, provenance, SBOM,
registry login, Snapshot delivery and manifest publication. `ProviderCapabilityVersion` versions the capability
semantics; `DriverVersion` is diagnostic only and cannot alone drive placement.

These contracts do not belong to Runtime Target, because a Target does not understand Build Templates, Destinations or
Build policy. They also do not alter Execution Plan structure. Requirements are derived from its existing frozen
fields, while accepted matcher results are retained as Placement Evidence. This keeps API, Execution Plan, Workspace
Snapshot and Artifact identity stable.

## Registry Credential Execution

The required execution chain is:

```text
Registry Connection -> Credential Reference -> Credential Provider
-> Ephemeral Execution Credential -> Runtime Execution Adapter -> Builder Driver -> Push
```

- Registry Connection chooses `credential_ref` and owns Registry/repository access policy. It stores a reference only.
- `CredentialProvider` is a required runtime contract, not a resource. It resolves a reference into a short-lived,
  endpoint-, repository-, push-operation- and time-window-scoped credential.
- `RuntimeExecutionAdapter` is a required provider boundary. It injects that credential into an isolated Docker,
  BuildKit, Kaniko or Kubernetes execution context and destroys it afterwards.
- The Driver receives only an ephemeral session, restricted mount or handle. Its public input, logs and evidence never
  contain a username, password, token, Docker config, credential path or secret-derived content.

Secret plaintext may exist only in Secret Backend/Credential Provider and the Adapter's constrained transient injection
boundary. It must never enter Build, Execution Plan, Task metadata, HTTP, audit, logs, Artifact, Publication, Pool or
Agent control-plane state. Docker must create a per-operation isolated `DOCKER_CONFIG` equivalent and may not read or
write its default credential store. Other providers use an equivalent short-lived token, restricted secret mount or
workload identity.

Missing references, resolution failures, scope/audience mismatch, expiry, unsupported injection, unprovable isolation or
unverifiable cleanup prevent Push. No path may fall back to environment-default authentication. Cleanup on success,
failure, cancellation, timeout and recovery destroys credentials, temporary directories, mounts and sessions. Cleanup
failure is a security failure: retain only redacted evidence, block reuse and put the Build into `Needs Attention` for
manual action. Harbor, Docker Hub, GHCR, ECR, GCR and ACR join through credential-provider and adapter conformance,
not Build-side special cases. Historical `docker-runtime-store` evidence stays readable; new execution may not use it.

## Builder Reservation

`BuilderReservation` is Build-owned capacity accounting with a short lease and fencing token. It is not a Task state
machine and does not claim external unmanaged work.

```text
Reserved -> Accepted -> Running -> Released
    |           |           |
    v           v           v
 Expired     Expired     Abandoned
```

- `Reserved`: Matcher selected a Builder and Build atomically acquired its fenced lease.
- `Accepted`: Task Runtime accepted the frozen Task or leg binding; it may still expire safely before execution.
- `Running`: Task Runtime started the execution stage. Only renewal and fencing preserve it; ordinary TTL cannot reclaim it.
- `Released`: success, failure, cancellation or a provably stopped attempt returned capacity with reason and evidence.
- `Expired`: a pre-running lease timed out.
- `Abandoned`: crash, restart or external uncertainty prevents confirmation. It enters Task Runtime recovery and is not
  silently reassigned.

Task Runtime triggers, settles and recovers the association; Provider or Agent contributes only constrained execution
facts. Every retry receives a new reservation and fencing token. Old leases are never revived.

## Workspace, Artifact And Evidence

The workspace chain remains `Workspace Snapshot -> WorkspaceMaterializer -> Execution Workspace -> Cleanup`.
`WorkspaceMaterializer` is a Build service contract, not an Aggregate: Workspace is mutable source configuration and
Task Runtime remains generic, so neither can own execution materialization. Reuse, clone, copy-on-write and cache are
Materializer strategies keyed by immutable Snapshot identity/digest, isolated per execution, and cannot become a new
Workspace authority. Build owns materialization lease and byte cleanup without changing Snapshot, Plan, Artifact or
Publication identity.

Artifact remains the first-class immutable digest identity. Metadata, provenance, SBOM, SLSA, Cosign and attestations
are immutable associated evidence, not another Artifact resource.

`Execution Evidence` is one semantic vocabulary, not an evidence database. Existing authorities write their own
evidence, which must link Plan, Task/Stage/leg/attempt, Builder Identity, reservation fence, Provider capability
version, time and redacted integrity summary. Replay uses frozen evidence and never re-queries mutable telemetry or
credentials.

## Telemetry And Provider Conformance

`RuntimeTargetBuilderTelemetryReader` remains Build's only telemetry read facade. Beneath it,
`BuilderTelemetryProvider` is the provider-specific observation adapter. It supplies Builder-scoped running builds,
queue length, allocatable Build slots, Builder health, proven platforms and labels, Driver/Provider capability profile,
cache state, source identity, observation/expiry time, integrity and unsupported dimensions.

Capability meaning is defined by the Build capability contract. Telemetry proves the source and freshness of an
observation. `docker stats`, gopsutil, CPU or memory percentages, host load, UI summaries, Monitor charts, endpoint
names, Task JSON, static labels and container counts are diagnostics only. They cannot be Placement input because they
do not prove Builder scope, Graft/external work coverage, time window, provenance or capacity occupancy.

- Docker requires a Builder Agent/control plane or equivalent scoped queue authority. Docker Engine and host metrics are
  insufficient.
- BuildKit reads its bound controller or worker.
- Kaniko may expose static capability only when it cannot prove queue/slot authority.
- Kubernetes reads a specifically bound BuildKit/Kaniko controller, workload or controlled API, never whole-cluster
  resources as Builder capacity.
- Future Providers pass conformance before entering Matcher or Placement.

Provider conformance extends the existing `TargetBoundProviderExecutionConformanceCapability`; it does not create a
competing specification. Providers MUST prove capability version, credential injection, Snapshot delivery,
cancellation/recovery, cleanup, redacted evidence, telemetry freshness and unsupported dimensions. They SHOULD prove
cache state, health diagnostics, idempotent receipts and Builder-scoped capacity explanation. Agent projection,
provider-specific optimization and read-only diagnostics are MAY capabilities. Unknown or non-conformant Providers may
participate only in static/manual paths and never in dynamic policy selection.

## Placement, Events And Failures

`PlacementPolicy` is Build-owned. It consumes authorized Instances, static eligibility, `CapabilityMatcher` results and,
when allowed, fresh telemetry and Reservation. It returns a selected Builder with frozen Evidence or a deny reason.

| Release phase | Policies |
| --- | --- |
| 1 | `Manual` only |
| 3 | `RoundRobin`, auditable-seeded `Random`, Pool `Manual` |
| 4 | `LeastLoad`, `Capacity`, `Affinity` |

Labels are static eligibility selectors, not a dynamic policy. `Custom` is reserved for deterministic, auditable,
compile-time registration; arbitrary plug-ins and OPA require a separate authority design. Existing `least_load`,
`affinity` and `region` literals are latent/disabled until the Phase 4 gate; they are not supported features.

`BuildEvent` is derived vocabulary over Task Runtime facts, not a second event store or bus: `Queued`, `Reserved`,
`Started`, `Progress`, `Completed`, `Failed` and `Cancelled`. Task/Stage remains lifecycle authority and `task_events`
must not duplicate Stage lifecycle records.

| Failure class | Default handling |
| --- | --- |
| Transient | bounded backoff retry; local circuit break after threshold |
| Permanent | terminate without retry |
| Configuration | no retry; manual repair |
| Authorization | no retry; security handling |
| Infrastructure | retry only after re-conformance of the same frozen placement; otherwise manual handling |
| Provider | locally disable the Provider capability and escalate |
| Internal | no default retry; record incident |
| Unknown | `Needs Attention`; do not guess success or reschedule |

The taxonomy maps to existing Task Runtime failure codes and recovery policies.

## Four-Phase Delivery Gates

| Capability | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
| --- | ---: | ---: | ---: | ---: |
| one Builder / manual selection | yes | yes | yes | yes |
| capability discovery / matcher | yes | yes | yes | yes |
| secure Registry credential execution | yes | yes | yes | yes |
| Reservation lifecycle | yes | yes | yes | yes |
| Driver, Template, Workspace materialization | basic | yes | yes | yes |
| Pool / RoundRobin / Random | no | no | yes | yes |
| Telemetry | no | no | static diagnostics | dynamic authority |
| LeastLoad / Capacity / Affinity | no | no | no | yes |
| Builder Agent / distributed Build | no | no | no | yes |

Phase 1 is a single Builder selected manually with capability discovery, secure Push and minimum Reservation. Phase 2
completes Builder Instance, Driver, Template and Workspace materialization but does not open Pool selection. Phase 3
opens static Pool policies only and never reads Telemetry. Phase 4 adds Builder Agent, real Provider Telemetry,
Reservation recovery, dynamic policies and Task Runtime-owned distributed Build.

Existing Pool and Placement foundations are retained as implementation assets, but their public exposure follows these
gates. Retries keep their frozen Plan and never implicitly select another Target.

## Migration And Acceptance

1. Runtime Target continues to own build capability. Add Matcher and manual single-Builder Evidence first.
2. Make logical Builder Instances explicit while preserving Target connection identity and existing Plan selection facts.
3. Existing Pools remain readable; Phase 1 writes allow only manual single-Builder selection.
4. Open dynamic policies only after real Telemetry Provider and Reservation recovery; no retry changes a frozen Target.
5. Retire `docker-runtime-store` to read-only historical evidence. New publications always use Credential Provider and
   Runtime Execution Adapter.

Acceptance requires that every new execution avoids the default Docker credential store; every dynamic Placement can be
replayed from fresh telemetry, capability profile and reservation fence; secret and cleanup failures are redacted and
fail closed; and Registry/Builder-local failure leaves platform availability unchanged.
