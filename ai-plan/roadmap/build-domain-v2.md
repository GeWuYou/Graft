# Build Domain v2 Roadmap

## Delivery Principle

Each phase is independently releasable and preserves the v2 authority chain: immutable Workspace Snapshot and
Execution Plan, Runtime Target-owned build capability, Task Runtime-owned execution, first-class Artifact, and
provider-neutral Artifact Destination.

## Phase 1: Authority And Single Builder

- Publish Build Domain v2 authority and retire Docker-first new-write semantics.
- Add Runtime Target `build` capability discovery and a controlled Application Workspace Snapshot adapter.
- Add immutable Execution Plan submission, one Docker-compatible Driver/Profile/Instance, and one eligible Runtime
  Target build capability.
- Add Registry Connection, Artifact Repository, OCI Registry Destination, Artifact evidence and Publication records.
- Deliver the platform create flow and direct v2 API write-contract replacement. Legacy completed jobs remain readable.

**Release gate:** a user can freeze an authorized Application Workspace Snapshot, submit a plan to one compatible Builder
Instance, publish an immutable Artifact to an authorized OCI Repository, and retry from the original frozen Plan.

## Phase 1.5: Registry Credential Execution

- Establish the Registry-owned non-plaintext credential execution contract for OCI publication.
- Bind a Registry Connection to a supported Runtime credential mechanism without placing credential values, Docker
  configuration paths or login commands in Execution Plans, Task metadata, HTTP responses or logs.
- Start with the selected Docker Runtime's credential store and fail closed for unsupported credential execution modes.

**Release gate:** every OCI publication records a Registry-selected credential execution mode and the target adapter
rejects any mode it cannot execute; secret material remains owned by the Runtime environment.

## Phase 1.75: Snapshot Materialization And Retention Foundation

- Make Build the lifecycle authority for frozen snapshot materialization; Project remains only an authorized Application
  Workspace source adapter.
- Record materialization ownership, availability state and retention policy separately from immutable Snapshot identity.
- Remove content-digest global uniqueness so equal bytes from different source provenance remain distinct Snapshots.
- Keep the materialization reference execution-private and opaque; cleanup may purge materialized bytes without deleting
  the auditable Snapshot or any immutable Execution Plan that references it.

**Release gate:** a submitted plan retains one auditable Snapshot identity independent of materialization cleanup, and
materialization state/retention ownership is queryable without exposing host paths or source credentials.

Foundation delivered: Build can lease and purge expired private materializations with interrupted-lease recovery. It
retains every Snapshot and Execution Plan identity, and refuses any materialization reference outside Build-owned
storage.

## Phase 2: Workspace And Build Intent

- Add the Build-owned reusable Workspace aggregate and immutable Snapshot materialization, beginning with the authorized
  Application Workspace source adapter; Git, uploaded archive, generated Workspace and approved target-local sources
  require their own provider-bound sub-batches.
- Add managed Template versions and Driver compatibility contracts, beginning with additional OCI/Dockerfile Drivers
  only where the Runtime Target capability proves support.
- Add source-transfer and retention contracts plus additional Artifact Destinations where delivery requirements justify
  them.

**Release gate:** the released source types create immutable, auditable Snapshots through Build-owned Workspaces; a
mutable Application Workspace cannot alter an already-submitted plan, and unsupported source kinds fail closed.

## Phase 3: Builder Pools And Multi-platform

- Add Build-owned Builder Profiles, Instances, Pools and Pool membership with capacity, labels, affinity and region
  eligibility evidence.
- Add the first transactional pool policy, round robin, with a persisted selection cursor; other policies remain
  disabled until truthful load, affinity and region telemetry exists.
- Validate every selected Instance against Runtime Target build capability and caller assignment before an Execution
  Plan can freeze it. The current public write contract continues to accept an explicit Runtime Target while the pool
  authority is introduced behind the Build service boundary.
- Do not implement platform fan-out in a single Build executor. It is gated by the distributed-leg coordinator release gate in Phase 6.

**Release gate:** a multi-platform plan selects compatible Instances deterministically, records every platform Artifact,
and publishes a manifest only when every required leg succeeds.

## Phase 4: Artifact Supply Chain And Automation

- Add Artifact promotion/copy, remote builders, distributed cache and additional OCI Artifact kinds.
- Add SBOM, provenance, signatures, attestations and supply-chain policy gates as Artifact-linked evidence.
- Add pipeline and deployment handoff that reference Artifacts and Publications rather than Build Jobs.

**Release gate:** promoted, scanned and signed Artifacts can be deployed independently of the Build Job that produced
them, with immutable provenance and publication history.

## Cross-phase Constraints

- Do not introduce a Graft-built Registry service; external registry providers remain Infrastructure integrations.
- Do not turn Builder Pool into a second Task Runtime or scheduler.
- Do not expose Docker-specific context paths, endpoints, credentials or implementation flags in the generic Build
  contract or review UI.
- Do not make a mutable tag the identity of an Artifact.

## Phase 5: Task Runtime Coordination Contract

- Define a Task Runtime-owned coordinated execution contract for multiple Build legs, including leg identity,
  platform assignment, shared cancellation, retry/recovery semantics and aggregate terminal state.
- Define the Build-owned manifest aggregation and publication handoff that consumes successful immutable platform
  Artifacts without creating a second Build queue or worker runtime.
- Freeze the contract before enabling multi-platform fan-out, remote builder execution or distributed cache workers.

Phase 5 establishes the public coordination vocabulary and keeps unsupported fan-out fail-closed. It does not claim
that a coordinated task can yet be executed.

## Phase 6: Distributed Leg Coordinator (new gap closure)

Phase 5 exposed a concrete implementation gap: Task Runtime has no persisted leg aggregate, parallel claim protocol,
shared cancellation, or recovery-aware terminal aggregation. This phase closes that gap before Build enables more than
one platform.

Scope:

- Task Runtime-owned coordinated-task and leg persistence, with stable leg identity, platform, Builder Instance,
  Runtime Target, attempt, status and recovery facts.
- Transactional leg claiming that allows independent legs of one coordinated task to run concurrently while preventing
  duplicate claims.
- Aggregate terminal semantics: all required legs succeed before success; a required failure cancels or drains the
  remaining legs; cancellation is propagated to every running leg; uncertain external outcomes remain recoverable.
- Shared retry/fencing and restart recovery for individual legs without rebuilding the frozen Execution Plan.
- Build adapter for platform Artifact results and Build-owned OCI manifest aggregation/publication.
- Release gate tests and observability for parallelism, cancellation, retry, recovery and manifest completeness.

Out of scope until this phase is complete:

- accepting multi-platform Build submissions;
- emulating fan-out with a loop inside the Build executor;
- claiming that Builder Pool selection alone provides distributed execution.

**Release gate:** Task Runtime can observe, cancel, retry and recover coordinated platform legs as one task, while Build
retains plan, Artifact and manifest authority and no executor-local loop acts as a scheduler.

## Phase 7: Driver-owned OCI Manifest Publication (new provider gap)

Phase 6 establishes execution and Artifact authority, but it must not invent a provider implementation for OCI Index
creation. The selected Builder Driver/Runtime adapter owns the only safe merge-and-publish action.

Scope:

- A versioned Driver capability that accepts only selected immutable platform Artifact digests and an authorized
  destination binding, then publishes an OCI Image Index/Manifest List on the selected Runtime Target.
- Build-owned validation that every frozen platform has exactly one successful Artifact before requesting publication.
- Immutable manifest Artifact recording followed by one mutable Publication, with idempotent replay and no tag-based
  identity.
- Driver implementations for Docker Buildx, BuildKit, Kaniko or remote builders only when each provider can prove
  digest-preserving manifest publication.

Out of scope:

- shelling out from Build with Docker commands;
- synthesizing a manifest from mutable tags;
- publishing a partial platform set.

**Release gate:** a provider-owned adapter publishes a digest-addressed OCI Manifest only after Build verifies the full
Artifact set; only then may the v2 API accept multi-platform submission.

## Phase 8: Platform-aware Builder Pool Scheduling (additional gap)

Phase 7 closes the single-pool-instance execution path: each coordinated leg is scheduled by Task Runtime and a
Buildx-capable target can publish the final manifest. The remaining gap is truthful placement across different
Runtime Targets. The current pool selection freezes one instance for the plan, so it cannot yet express an AMD64 leg
on one target and an ARM64 leg on another, nor can it prove load, region or affinity decisions.

Scope:

- A scheduler-owned placement plan that assigns each platform leg to an authorized Builder Instance/Runtime Target.
- Audited policies for least load, labels, affinity and region, with explicit capability and locality checks. A
  labels placement freezes the validated selector evidence alongside the selected Instance.
- Per-leg target identity in the coordinated Task contract and execution evidence; Build remains free of scheduler loops.
- Manifest finalization only after every assigned leg settles an immutable digest.

Release gate: a multi-platform plan can use more than one Runtime Target without changing Workspace Snapshot,
Execution Plan, Artifact or Publication authorities, and placement decisions are replayable from persisted evidence.

Foundation delivered: `Builder Placement` is persisted as immutable plan data and is consumed by coordinated legs and
their Build executor. It requires a target-declared `build-snapshot` locality. The remaining Phase 8 work is policy
truth beyond round robin. Provider-backed remote materialization and execution evidence is a separate Phase 9 so it
does not block the independently releasable placement model.

### Phase 8A: Builder Telemetry Authority (additional gap closure)

The scheduler cannot safely enable `least_load`, `region` or `affinity` from the current Runtime Target/UI metrics.
Those projections lack builder queue/capacity semantics, freshness expiry, source provenance and canonical region or
affinity ownership. This phase defines and implements the Runtime/Infrastructure-owned telemetry authority before
enabling those policies.

Scope:

- expose Runtime Target-scoped capacity, running, queued, availability, `ObservedAt` and `ExpiresAt` through
  `RuntimeTargetBuilderTelemetryReader`; Build binds that fact to its own Builder Instance when freezing placement;
- establish canonical region and affinity claims with authorization and provider provenance;
- let Build Scheduler consume only fresh, self-consistent snapshots and persist their identity as replayable placement
  evidence;
- add provider/integration fixtures for stale, missing, conflicting and unauthorized telemetry.

The provider-neutral contract is now defined, but no source implementation is claimed. Until its release gate passes,
`round_robin` and static `labels` remain the only executable policies; unsupported policies fail closed.

**Release gate:** a registered authority can provide fresh, authorized telemetry for every selected Builder Instance,
and a replayed placement can be explained from frozen evidence without consulting mutable UI summaries.

### Phase 8B: Builder Capacity And Telemetry Source Authority (additional prerequisite)

Phase 8A cannot source truthful telemetry from Docker UI summaries, container metrics, Monitor host data or Task JSON.
Add the missing authority as a separately releasable prerequisite:

- Runtime/Infrastructure registers a provider-conformant target telemetry source that supplies durable or
  provider-queried capacity, running, queued, availability, provenance, freshness, region and affinity facts.
- Build owns an atomic Instance reservation ledger keyed by Execution Plan/Task leg. It prevents concurrent Graft
  submissions from oversubscribing a target without claiming to account for external work.
- Task Runtime completion, cancellation and recovery reconcile fenced reservations; every accepted source fact and
  reservation reference freezes into Builder Placement scheduling evidence.
- At least one provider must demonstrate restart-safe source freshness and concurrent capacity behaviour before a
  dynamic scheduling policy becomes selectable. Docker requires a Build agent/control-plane or equivalent authority;
  Docker Engine itself is not a Graft-scoped queue source.

**Release gate:** a real provider source and Build reservation ledger pass stale/replay, concurrent allocation,
cancellation/recovery and multi-platform leg evidence tests. Only then may `least_load`, `region` or `affinity` be
enabled; `round_robin` and `labels` remain independently releasable.

## Phase 9: Snapshot Delivery And Remote Builder Execution (new gap closure)

Phase 8 can freeze and audit a placement, but a placement is not proof that a different Runtime Target can consume a
Build-owned Snapshot. The Phase 9 Docker provider now supports local Unix-socket and validated remote TCP/SSH targets.
Kubernetes and other providers remain deliberately disabled. Reusing a host-local Docker process or a host path for a
different Runtime Target would violate Runtime Target and Workspace authority; this phase supplies provider execution
proof without changing the immutable Build model.

Scope:

- Define a provider-owned Snapshot delivery contract that transfers or materializes one immutable, Build-owned
  Snapshot to an explicitly selected Runtime Target without exposing arbitrary paths, source credentials or endpoint
  details to Build plans or Task metadata.
- Implement one proven remote-provider adapter at a time, such as a remote Docker transfer path, a Kubernetes
  BuildKit workspace volume/transfer, or a Kaniko-compatible upload mechanism. Each adapter must attest the received
  Snapshot identity before Driver execution.
- Require Runtime Target capability discovery to report both the supported Driver and the specific Snapshot delivery
  mode; `build-snapshot` locality alone is not an execution adapter. The Docker executor accepts `target-local` for
  Unix sockets and `provider-transfer` only for a proven TCP/SSH adapter.
- Bind Registry credential execution, platform capability, materialization cleanup and cancellation/recovery to the
  selected provider adapter. Build continues to freeze the Placement; Task Runtime continues to run and recover the
  leg.
- Add provider-bound integration tests that prove a selected non-local Target builds the frozen Snapshot and publishes
  the expected immutable platform digest. Unsupported Targets remain unavailable rather than falling back locally.

Out of scope:

- copying host directories to a target through generic Build code;
- treating a Runtime Target capability declaration as evidence that a provider adapter exists;
- implementing a second source store, queue, worker, scheduler or registry.

**Release gate:** at least one non-local Runtime Target provider can receive a specific immutable Snapshot through its
declared delivery mode, execute its declared Driver, and settle the same plan/Artifact/Publication facts as Local
Docker. Other providers remain fail-closed until they meet the same proof.

### Phase 9A: Delivery Contract Foundation (completed)

The provider-owned delivery contract is now part of the stable Build/Runtime boundary. Docker proves `target-local`
consumption from the Build-managed Snapshot root for Unix sockets and `provider-transfer` for validated TCP/SSH targets.
Snapshot identity and digest mismatches fail closed and are covered by provider and executor tests.

### Phase 9B: Remote Docker Provider Adapter Enablement (completed)

This sub-phase is intentionally separate from the foundation. Runtime Target now owns a private target-scoped Docker
connection lookup and validates availability, build capability and endpoint/connection-kind consistency before the
provider executes. The provider accepts `target-local` for Unix sockets and `provider-transfer` for validated TCP/SSH
targets; the Docker client transfers the immutable Snapshot context and executes Build, image publication and OCI
manifest publication against the selected target. Registry credential execution remains the existing provider-owned
Docker credential-store binding. No endpoint, credential or host path enters Build plans, Task metadata or HTTP, and
unsupported targets remain fail-closed without local fallback.

The first authority repair for 9B is now in place: Runtime Target's store can retrieve a private Docker connection fact
for an existing target. The fact is not part of `BuildRuntimeTargetSummary`; the target-bound provider is registered by
Runtime Target and is the only production owner of remote Docker execution.

Container keeps the legacy local Docker capability for container management and standalone module tests. In production,
Runtime Target registers the target-bound build, publication, manifest and Snapshot delivery capabilities explicitly, so
Build resolves the provider boundary without a type assertion and without a second execution runtime.

The private connection lookup also rejects unavailable targets, targets without `image_build`, malformed endpoints and
unsupported connection-kind/scheme combinations before any provider can consume the fact. Kubernetes, Kaniko, BuildKit
pods and other non-Docker providers remain future Phase 9D work and are not advertised by the current target summary.

### Phase 9C: Provider Execution Foundation (additional gap closure)

Phase 9B proves Docker execution, but the non-Docker provider gap includes more than an adapter implementation: the
provider-neutral execution contract must define connection and credential ownership, Snapshot import proof, digest
settlement, cancellation/recovery fencing and capability registration. This phase releases that foundation without
shipping or advertising a Kubernetes, BuildKit or Kaniko provider.

Scope:

- define the private provider execution and conformance contract for connection acquisition, Snapshot delivery and
  identity verification, Driver invocation, credential mode, cancellation, timeout, recovery and Artifact/Publication
  evidence;
- introduce `TargetBoundProviderDriverExecutionCapability` as the provider-neutral Driver execution boundary; its
  request carries frozen Snapshot identity and delivery proof, and its result carries only digest-addressed execution
  evidence;
- require Runtime Target capability declarations to resolve to a registered provider implementation and health check;
- add fail-closed conformance tests for unsupported providers, unavailable connections, Snapshot digest mismatch,
  credential rejection and uncertain external outcomes;
- preserve the existing Docker adapters and Artifact/Publication authorities as the reference implementation.

**Release gate:** unsupported or unverifiable providers are rejected before scheduling, no local fallback is possible,
and the contract proves that endpoint, credential and host-path facts remain below the Runtime Target/provider boundary.

### Phase 9D: Concrete Kubernetes/BuildKit/Kaniko Provider Adapters

Implement one non-Docker provider at a time behind the Phase 9C contract. Each adapter must receive the exact immutable
Snapshot, execute its declared Driver on the selected Runtime Target, own target-specific connection and credential
resolution, support cancellation/recovery, and return digest-preserving Artifact/Publication evidence. A provider is
selectable only after its provider-bound integration proof passes; all other providers remain fail-closed.

## Phase 10: Build Selector Read Model And Platformized Create Flow (additional gap closure)

The execution authorities were complete while the create page still exposed internal identifiers as free-form inputs.
Phase 10 closes that independent UI/API gap without weakening module boundaries: Build owns read endpoints for authorized
Workspaces, Build-capable Runtime Targets and Builder Pools, while Runtime Target continues to own target assignment and
private connection facts.

Scope:

- expose `GET /api/build/workspaces`, `GET /api/build/runtime-targets` and `GET /api/build/builder-pools` from the Build
  module using non-secret selector projections;
- filter Workspaces by caller ownership/shared visibility, Runtime Targets by Build assignment and capability, and Pools
  by at least one member Runtime Target the caller may use for Build. Pools whose policy lacks authoritative scheduler
  evidence are omitted from the selectable projection until their policy is implemented;
- replace manual Workspace, Runtime Target and Builder Pool ID inputs with controlled TDesign selectors and explicit
  loading, empty and error states;
- keep Template, Driver, Snapshot and Destination references provider-neutral and preserve the immutable Execution Plan
  submission boundary.

Out of scope: exposing Runtime endpoints or credentials, importing another module's private frontend API, or enabling
Kubernetes/BuildKit/Kaniko execution (Phase 9D remains the concrete-provider gap; Phase 9C defines its foundation).

**Release gate:** an authorized operator can open the create flow, load only selectable Build resources through Build-owned
API contracts, choose a Workspace and Builder selector without typing opaque IDs, and submit the same immutable v2 plan.
