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
| Provider registration and lifecycle admission | Build Provider Admission / Conformance authority | Platform availability, Runtime Target health |
| Build queue state and eligibility reason | Build Queue Authority | Task Runtime scheduler, UI queue projection |
| dependency edges and readiness | Build Dependency Graph (future Build authority) | Task Runtime lifecycle, ad-hoc event handlers |
| reserved resource accounting | Build Resource Accounting (future Build authority) | Telemetry observation, host metrics, reservation lease alone |
| cost and usage evidence | Build Cost Evidence (future Build authority) | billing UI, Provider telemetry, Artifact identity |
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

Placement Evidence is immutable and contains the policy identifier and version, deterministic selection seed, input
fingerprint, ordered candidate/output decision, telemetry observation identity and freshness, selected provider
capability version, and the complete capability-negotiation result. Replay consumes this evidence with the frozen Plan;
it never substitutes current telemetry, current capability or a later policy implementation for an earlier decision.

### Capability negotiation

`CapabilityMatcher` may negotiate a result from one requirement against one or more provider capabilities, but it never
silently weakens a requirement. Each requested feature has an explicit mode: `required`, `preferred` or `optional`.
The result records satisfied, negotiated and unsatisfied features, the selected `ProviderCapabilityVersion`, and the
reason a preferred feature was not selected. A `required` miss is a deny; a `preferred` miss is an auditable trade-off;
an `optional` miss is omitted from the execution context. Negotiation output is frozen Placement Evidence and is replayed
from that evidence rather than recomputed from later provider state.

Every new Placement, including manual selection, static Pool selection, dynamic Pool selection and every distributed
leg, MUST call `CapabilityMatcher` before a Builder is selected. The frozen evidence records the requirement
fingerprint, ordered candidate fingerprint, selected capability profile and version, complete negotiation result,
policy ID/version and Reservation fence. A required miss denies the Placement; preferred and optional misses remain
explicit negotiated results rather than implicit provider fallbacks.

### Builder Profile separation

`BuilderProfile` is Build-owned static intent and eligibility metadata (for example `secure`, `gpu`, `large-memory` or
`fast`). It is not measured capability, health or load. `BuildExecutionCapability` remains the provider-proven fact,
and Telemetry remains the time-bounded observation of health and capacity. Profiles may filter candidates before matching,
but cannot fabricate capability or become a dynamic load signal.

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

### Phase 1 deployment secret source

Phase 1 supplies a core-owned, file-backed `CredentialProvider` for deployments that do not yet have a managed secret
backend. `GRAFT_REGISTRY_CREDENTIALS_FILE` is an optional absolute path to a deployment-mounted, read-only JSON secret
file; it is not a Registry resource, System Config value, Build input or HTTP contract. When unset, the provider and
the Runtime Target execution adapter are both absent, so new publication fails closed. When set, an unreadable,
non-regular, group/other-writable, malformed or invalid source prevents provider admission rather than enabling any
ambient Docker or environment authentication.

The file has version `1` and each entry contains only `credential_ref`, `endpoint`, `repositories`, `operations`,
`username`, `password` and `expires_at`. Entries require an HTTPS endpoint, a non-empty expiring credential, exact
operations and either an exact repository or a segment-safe `prefix/*` repository scope. The provider reloads the file
for each secret-free scoped eligibility assessment and each `Prepare`. Assessment accepts only a known opaque reference
plus endpoint/repository/operation scope and returns `eligible` or `ineligible`; it never lists entries or exposes
expiry, source, username, password or other secret-derived data. `Prepare` repeats scope validation, returns only an
opaque session ID and expiry, keeps plaintext only until `Revoke`, and writes Docker
`config.json` only to the adapter-created `0700` directory. The file path and contents are excluded from snapshots,
Task state, audit, logs, artifacts, publications and HTTP. A managed secret backend may replace this provider only by
implementing the same scoped `CredentialProvider` contract; it must not add an ambient-auth compatibility path.

Missing references, resolution failures, scope/audience mismatch, expiry, unsupported injection, unprovable isolation or
unverifiable cleanup prevent Push. No path may fall back to environment-default authentication. Cleanup on success,
failure, cancellation, timeout and recovery destroys credentials, temporary directories, mounts and sessions. Cleanup
failure is a security failure: retain only redacted evidence, block reuse and put the Build into `Needs Attention` for
manual action. Harbor, Docker Hub, GHCR, ECR, GCR and ACR join through credential-provider and adapter conformance,
not Build-side special cases. Historical `docker-runtime-store` evidence stays readable; new execution may not use it.

`credential_cleanup_unverified` is a stable `Internal` failure code with the `Needs Attention` recovery disposition.
Task Runtime remains the sole state-transition authority: it maps this constrained outcome without learning credential
or provider details, never auto-retries it, and prevents reservation or credential-session reuse while retaining only
redacted cleanup evidence. Credential resolution, scope, expiry and injection failures, plus Provider and
Infrastructure failures, retain their respective RFC taxonomy instead of being collapsed into `stage_executor_failed`.

## Provider Lifecycle And Compatibility

Provider conformance is an admission gate, while `ProviderLifecycle` controls whether an admitted provider can receive
new work. Its conceptual states are `Registered -> Validated -> Active -> Degraded -> Unavailable -> Retired`.
`Registered` is discovered but not trusted, `Validated` has passed required conformance checks, `Active` may serve the
policies allowed by its phase, `Degraded` may serve only explicitly safe static/manual paths, `Unavailable` cannot
receive work, and `Retired` is retained for historical evidence but cannot be selected. Lifecycle transitions are owned
by the Build provider authority and are local to Build; they do not write `CapabilityCoordinator` or global availability.
Runtime Target/Provider registry reports binding and conformance facts to the Build provider authority; it does not
admit providers or transition Build lifecycle state itself. Provider health observations may justify a transition, but
do not themselves redefine lifecycle authority.

Every Provider exposes a `ProviderCompatibilityContract` alongside its capability version. It declares the versions of
Capability, Execution Context, Snapshot delivery, Credential Adapter, Telemetry and Evidence it can consume or produce,
and labels each relation `forward-compatible`, `backward-compatible` or `breaking`. A breaking change cannot replay old
execution evidence or consume an incompatible frozen context without an explicit migration. Historical evidence remains
readable through its recorded versions; compatibility checks fail closed before new execution.

The conformance surface described here is the Build Domain boundary only. The separate
[Provider SDK And SPI RFC](build-domain-v2-provider-sdk-spi.md) packages the provider lifecycle, capability, telemetry,
credential, workspace, execution and evidence adapter seams plus a conformance suite. It must not become a second
runtime or authority.

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
- `Running`: Task Runtime started the execution stage. Only a fence-matched renewal preserves its lease; ordinary TTL
  expiry cannot reclaim it. A missing renewal is recovery evidence, not permission to reassign capacity.
- `Released`: success, failure, cancellation or a provably stopped attempt returned capacity with reason and evidence.
- `Expired`: a pre-running lease timed out.
- `Abandoned`: crash, restart or external uncertainty prevents confirmation. It enters Task Runtime recovery and is not
  silently reassigned.

Task Runtime triggers, settles and recovers the association; Provider or Agent contributes only constrained execution
facts. Every retry receives a new reservation and fencing token. Old leases are never revived.

Phase 4 Reservation is slot-aware: every Build leg reserves one explicit Builder capacity unit. `allocatable_slots` is
the Provider's instantaneous budget for Graft at the accepted observation. Under the Builder's serialized reservation
boundary, Build compares that budget only with live reservations created after that observation, so Provider-reported
running work is not charged twice and concurrent requests cannot over-allocate. Capacity ranking chooses among trusted
slots, but the atomic Reservation is the final verdict. A failed verdict denies that Placement; it never silently
chooses another target. `least_load` ranks only Provider-owned running/queued facts, and `affinity` filters only proven
affinity claims.

## Queue, Dependencies And Resource Accounting (Future)

`BuildQueueAuthority` is a future Build-owned contract for work waiting to become eligible. It records queue class and
priority, enqueue order, and a precise wait reason such as `WaitingCapability`, `WaitingReservation`,
`WaitingDependency`, `WaitingRetry` or `WaitingManualApproval`. `Queued` remains a derived Build event over Task Runtime
facts. The Queue Authority may order Build candidates and request a reservation, but it cannot execute tasks, cancel them,
or introduce a scheduler/event bus. Phase 1 and Phase 2 use Task Runtime's existing queueing path; this contract is not
publicly enabled until its ownership and recovery semantics are implemented.

`BuildDependencyGraph` is a future Build-owned immutable graph of Build-to-Build or leg-to-leg prerequisites. It answers
readiness and records the dependency snapshot used for a placement decision; it does not own Task lifecycle or duplicate
Task events. Cycles, missing references and stale dependency snapshots are configuration failures and must fail closed.
Until a later phase defines graph persistence and recovery, no implicit DAG or event-handler dependency semantics are
supported.

`BuildResourceAccounting` is a future Build-owned ledger for resources reserved by a Builder Reservation, such as CPU,
memory, disk, cache namespace and network budget. It records requested, reserved, consumed and released quantities with
the reservation fencing token. Telemetry remains observation only, and the ledger does not claim capacity for unmanaged
external work. Quota, fair-share and burst policies are later consumers of this ledger, not alternative authorities.

`BuildCostEvidence` is a future immutable association to an execution attempt that records provider-reported usage,
pricing/cost model version, currency or credits, and the evidence interval. It does not change Artifact identity or act as
a billing ledger. Unknown or unverifiable cost is recorded as unknown, never guessed, and cost evidence cannot authorize
placement or publication.

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

The Phase 4 Docker admission proof is a provisioned Builder Agent bound to one Runtime Target, Provider, Builder scope
and capability profile/version. Its reports require a monotonic sequence, bounded clock skew, replay rejection and an
explicit lifecycle; running, queued and slots come from the Agent's controlled execution ledger or Driver controller.
The Docker provider's sole CLI execution boundary updates that durable Driver-controller ledger around each real build;
one enabled Agent scope is allowed per Runtime Target until frozen Placement carries a scope identity. This proves the
source of the execution counts, but does not itself define an out-of-process Agent transport, private-key bootstrap or
operator lifecycle. Those deployment facts require their own authority-owned protocol before this phase can be released.
A generic signed ingress, Docker stats, container counts, host metrics, Task projections or UI data is not that proof.
Profile/version, scope, provenance, integrity and unsupported dimensions must all validate before telemetry can enter a
dynamic decision. Until this gate is proven, dynamic policy remains disabled and historical dynamic Pool rows remain
readable but non-executable.

### Docker Agent deployment control plane

Runtime Target owns the operator-facing Docker Builder Agent deployment control plane. It is a narrow telemetry
identity and lifecycle authority, not a Build scheduler, Task Runtime, Registry model or platform health registry.
The control plane owns enrollment, installation metadata, target/provider/scope/profile binding, one-active-scope
uniqueness, report transport authentication, agent retirement and the redacted audit trail. Build continues to own
requirements, matching, placement and reservations; Task Runtime continues to own execution, cancellation, retry and
recovery.

The first deployable protocol follows the accepted [Runtime Target Agent Trust Model ADR](../decisions/ADR-023-runtime-target-agent-trust-model.md)
and [Credential Vault And Runtime Target Agent Protocol RFC](credential-vault-and-runtime-target-agent-protocol.md),
which define Vault-backed issuance, exact URI SAN identity, vault-managed enrollment/private-key delivery, mTLS
snapshot acknowledgement, OCI packaging and revocation propagation together with a Runtime Target-bound enrollment
record and mutually authenticated Agent transport. Runtime Target stores only opaque references, digests, expiry and
binding facts. Bootstrap credentials and private keys never enter Build, Task, browser, telemetry reads, logs or
execution evidence. Rotation creates a new enrollment generation and revokes the old generation before accepting a
new sequence; disable, retirement and target rebinding reject all later reports from that generation.

Each accepted report is bound to the enrollment generation, Runtime Target, Docker Provider ID, Builder scope and
capability profile/version. The transport authenticates the enrolled Agent identity, while the signed report preserves
integrity across queued delivery. Sequence, clock skew, freshness, scope/profile/version, unsupported dimensions and
ledger provenance are validated together. The Agent service owns restart reconciliation for its durable Driver ledger;
unknown external work is reported as unavailable rather than inferred from host metrics or Task state. Installation,
upgrade, rotation, revocation, disable and recovery require Runtime Target operator authorization and produce redacted
audit facts.

No Build HTTP route, Docker daemon API, ambient configuration, global agent registry or ad-hoc CLI is an enrollment or
telemetry authority. The ADR's concrete wire schema, operator API and service packaging must be introduced together under
this Runtime Target control-plane contract, with end-to-end conformance proving bootstrap, rotation/revocation,
restart/reconnect and real Docker CLI ledger reporting before Phase 4 can be released.

Provider conformance extends the existing `TargetBoundProviderExecutionConformanceCapability`; it does not create a
competing specification. Providers MUST prove capability version, credential injection, Snapshot delivery,
cancellation/recovery, cleanup, redacted evidence, telemetry freshness and unsupported dimensions. They SHOULD prove
cache state, health diagnostics, idempotent receipts and Builder-scoped capacity explanation. Agent projection,
provider-specific optimization and read-only diagnostics are MAY capabilities. Unknown or non-conformant Providers may
participate only in static/manual paths and never in dynamic policy selection.

## Placement, Events And Failures

`PlacementPolicy` is Build-owned. It consumes authorized Instances, static eligibility, `CapabilityMatcher` results and,
when allowed, fresh telemetry and Reservation. It returns a selected Builder with frozen Evidence or a deny reason.
Every policy has a stable `PolicyID` and `PolicyVersion`; the version, deterministic seed (when applicable), input
fingerprint and output are retained in Placement Evidence. A policy implementation change is therefore a new version,
not an in-place reinterpretation of old placements. Replay uses the recorded version and evidence and does not require
the current policy to produce the same result.

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

The taxonomy maps to existing Task Runtime failure codes and recovery policies. In particular,
`credential_cleanup_unverified` is `Internal` with `Needs Attention`, no automatic retry and no reuse of its
Reservation or credential session.

## Execution Context

`ExecutionContext` is the single provider-facing, per-attempt input assembled by Build from already-authorized and
frozen facts. It may contain environment contract, static labels, timeout/deadline, proxy and mirror policy, registry
aliases, cache namespace, Snapshot identity and the ephemeral credential session handle. It never contains secret
plaintext, ambient host configuration or mutable telemetry. Provider adapters must reject unknown or unsupported fields,
record the context schema version in Evidence, and keep provider-specific translation inside the adapter. Execution
Context is not a new Task metadata store and does not alter the immutable Execution Plan; it is an execution projection
that is frozen for the attempt and tied to its reservation fence.

## Four-Phase Delivery Gates

| Capability | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
| --- | ---: | ---: | ---: | ---: |
| one Builder / manual selection | yes | yes | yes | yes |
| capability discovery / matcher | yes | yes | yes | yes |
| secure Registry credential execution | yes | yes | yes | yes |
| Reservation lifecycle | yes | yes | yes | yes |
| Driver, Template, Workspace materialization | basic | yes | yes | yes |
| Pool / RoundRobin / Random | no | no | yes | yes |
| Provider lifecycle/conformance admission | basic validation | active/manual gating | degraded/static gating | recovery-aware gating |
| Queue, dependency, resource and cost contracts | latent only | latent only | bounded contracts as implemented | policy and accounting consumers |
| Capability negotiation / Builder Profile / versioned evidence | manual requirement match | full static negotiation | pool selection with frozen versions | telemetry-informed negotiation |
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
replayed from its frozen telemetry observation, capability profile, policy/version inputs, negotiation result and
reservation fence; fresh telemetry is used only for a new Placement decision. Secret and cleanup failures are redacted and
fail closed; and Registry/Builder-local failure leaves platform availability unchanged.
