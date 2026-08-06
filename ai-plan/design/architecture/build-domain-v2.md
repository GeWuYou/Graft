# Build Domain v2 Architecture

## 1. Purpose And Authority

Build Domain v2 makes Build a platform capability rather than a Docker command form. It supersedes the Docker-first
write model in [Docker Build Center](docker-build-center.md) while retaining completed legacy jobs as read-only history.

The central immutable chain is:

```text
Workspace -> Workspace Snapshot -> Execution Plan -> Task execution -> Artifact -> Publication -> Deployment
```

`Workspace` is reusable source configuration. A `Workspace Snapshot` is the exact immutable source material consumed by
a build. An `Execution Plan` freezes all execution inputs before Task submission. A `Build Job` records user intent,
authorization and references to plans, Task executions, Artifacts, and Publications; it does not own a second execution
state machine.

Retry creates a new Task execution from the same frozen Execution Plan and Workspace Snapshot. A rebuild of current
source creates a new Snapshot and a new Execution Plan; it is never an implicit retry.

## 2. Domain Boundaries

| Owner | Resources and responsibility | Explicit non-responsibility |
| --- | --- | --- |
| Runtime Domain | `Runtime Target`: provider connection, credential reference, authorization scope, health, labels and discovered capabilities, including `build`. | Build recipes, Build scheduling policy, Artifact history. |
| Build Domain | Workspaces, Snapshots, Templates, Drivers, Builder Profiles/Instances/Pools, Execution Plans, Build Jobs, Artifacts and Publications. | Runtime connections, plaintext credentials, Task state/log stores. |
| Infrastructure Domain | Registry Connection, Artifact Repository and destination-provider integration. Connection owns endpoint, credential reference, health and provider behavior; Repository owns namespace and repository policy. | Build execution state, Artifacts' identity, copied credentials. |
| Task Runtime | Task/Stage state, execution, logs, cancellation, retry, recovery and realtime. | Build-specific source, driver, registry or OCI semantics. |
| Application | Mutable Application Workspace and its authorization/lifecycle. It may expose a controlled Snapshot source adapter. | A universal Workspace directory or Build execution state. |

Runtime Target is the single authority for whether a physical target can build. A target reports provider-neutral build
capability details such as supported Drivers, platform support, transfer/locality features, availability and capacity
signals. Neither Builder resources nor Build Jobs repeat endpoint or connection configuration.

## 3. Core Aggregates And Relationships

### Workspace And Workspace Snapshot

`Workspace` owns a source definition and reusable metadata. Supported source kinds are Application Workspace, Git,
uploaded archive, generated Workspace and approved target-local materialization. It never means an arbitrary host path.

`Workspace Snapshot` resolves a Workspace at a point in time to immutable input evidence: a Git commit, frozen
Application content, archive digest, generated-content digest, or approved target-local capture. It records source
provenance and materialization constraints without exposing protected paths or credentials. A Snapshot has no mutable
content; retention only affects availability, never its identity. Source adapters may create a temporary capture, but
Build adopts the retained execution materialization and is its sole lifecycle owner; the persisted execution reference
is private and opaque outside the executor boundary.

### Template And Driver

`Build Template` is a versioned provider-neutral build-intent schema: declared inputs, expected outputs, compatible
Drivers, policy constraints and source requirements. It does not equal an executor.

`Build Driver` is an implementation contract and requirement set. Docker Engine, BuildKit, Kaniko, Buildah, Cloud Native
Buildpacks, Bazel and Nix are Drivers. A Dockerfile may be an input convention of an OCI-oriented Template, while the
actual selected Driver remains explicit. This separation permits one Template to run through more than one compatible
Driver without changing the user-level Build model.

### Builder Profile, Instance And Pool

`Builder Profile` is Build-owned reusable configuration and policy for a Driver: version, cache mode, network limits,
resource limits, platform limits and required labels. A `Builder Instance` binds one Profile to one eligible Runtime
Target `build` capability. It represents a runnable builder realization, not a second physical connection record.

`Builder Pool` is an optional collection of eligible Builder Instances and its scheduling policy. It selects Instances,
not Runtime Targets directly. Phase 1 allows exactly one eligible Instance. Later pool policies may use round robin,
least load, labels, affinity and region only after the associated capability signals are truthful and auditable.
The first Pool implementation uses a persisted, transactionally locked round-robin cursor; other policy values remain
disabled until their telemetry and placement authorities are available.

### Execution Plan, Build Job, Artifact And Publication

`Execution Plan` freezes a resolved Snapshot, Template version, selected Driver, Builder selector/Instance or Pool
policy, requested platforms, cache policy, labels, secret references, policy decisions and Artifact Destinations. It is
the review subject and the audit/retry authority.

`Artifact` is a first-class immutable, digest-addressed output resource. It stores OCI digest, platform set, size,
labels, creation metadata and, when produced, SBOM, provenance, signature and attestation references. Deployment,
promotion, scan and signing reference Artifact rather than Build Job.

`Publication` binds an immutable Artifact to a mutable delivery reference, such as a registry repository/tag, manifest
publication, promotion or copy. It preserves the historical relationship between a tag and its digest without making a
tag the Artifact identity.

`Artifact Destination` is the provider-neutral output interface. An OCI Registry destination is required in Phase 1 for
deployment-grade builds. Future destination providers can include OCI layout, tarball, object/artifact storage and SBOM
export without changing Artifact identity.

## 4. Execution And Lifecycle

The Build service validates authorization, compatibility and policy, freezes the Snapshot and Execution Plan atomically,
then submits a Task Runtime plan. Task Runtime projects Build stages but remains authoritative for all states and logs:

```text
prepare_workspace -> schedule_builder -> run_build -> export_artifact -> publish_destination
                                                                    -> publish_manifest (when required)
```

多平台执行的当前边界是：Task Runtime 负责 coordinated leg 的并行领取、取消、重试与恢复，Build executor 负责单 leg
构建、临时平台引用发布和最终 Manifest 结算。Phase 8 才会把 Builder Pool 的平台级 placement 提升为可审计的
多 Runtime Target 调度；在此之前，一个冻结计划只使用一个已选 Builder Instance，避免把未实现的负载或地域判断
伪装成调度结果。

Phase 8 的第一项实现将 `Builder Placement` 冻结到 Execution Plan：每个目标平台记录 Builder Instance、
Runtime Target、已采用的调度策略及其校验后的调度证据，并成为 Task Runtime leg 与 Build executor 的唯一目标依据。Workspace Snapshot
已经由 Build 接管，因此 target 必须明确声明 `build-snapshot` locality 才能被 placement 选中；这不是 endpoint 或
任意目录的兼容通道。

Placement 只表达并冻结调度决定，不能证明一个 Provider 能消费该 Snapshot。Runtime Target 还必须声明
`SnapshotDeliveryModes`；`WorkspaceLocalities` 只表示物化位置，不等于传输适配器。Phase 9 owns provider-backed Snapshot
delivery and remote execution: a provider receives or materializes the exact immutable Snapshot through a declared
delivery mode, verifies its identity, then executes its declared Driver. The Docker provider now proves the same
contract for local Unix-socket and validated remote TCP/SSH targets by invoking the selected target explicitly and
transferring the immutable Snapshot context through the Docker client. Build must never receive endpoint or
credential facts or compensate with a local target fallback. The remaining non-Docker work is split into a provider
execution foundation (Phase 9C) and concrete Kubernetes/BuildKit/Kaniko adapters (Phase 9D); capability declarations
alone are never treated as execution proof.

The Phase 9 foundation uses `TargetBoundWorkspaceSnapshotDeliveryCapability` as the provider boundary. Build passes only
the frozen Snapshot identity, content digest, selected target and declared delivery mode inside the execution-private
call. The Docker adapter accepts `target-local` for Unix-socket targets and `provider-transfer` for validated TCP/SSH
targets, verifies the managed Build snapshot root and returns a matching delivery proof before Docker execution.
Unsupported targets and connection kinds remain fail-closed; they must never fall back to the local Docker process.

Build-owned Snapshot retention 使用显式清理 lease：到期物化从 `available` 或 `expired` 进入 `purging`，删除私有
物化字节后才进入 `purged`。中断的 `purging` lease 可被重新领取；任何不在 Build 私有快照根目录下的引用都会被拒绝。
清理不删除 Snapshot、Execution Plan、Artifact 或 Publication，因此历史重试会明确看到物化不可用，而不是隐式刷新来源。

Build exposes a derived lifecycle suitable for Build users: `Draft`, `Queued`, `Preparing Workspace`, `Scheduling
Builder`, `Running`, `Exporting Artifact`, `Publishing`, `Succeeded`, `Failed`, `Cancelled`, or `Needs Attention`.
These are projections of a frozen Plan plus Task/Stage facts, not an independent mutable Build state machine. Unknown
external outcomes remain `Needs Attention` until Task Runtime recovery is reconciled; Build must never infer success.

Submission is rejected when the Snapshot cannot be materialized, the selected Builder is unauthorized/unhealthy or
incompatible, the Driver cannot execute the Template, the requested platforms cannot be met, or every Destination is
invalid. Secret values, Runtime endpoint details and arbitrary host paths never enter the plan, API response or log.

## 5. Registry And Artifact Delivery

Graft does not implement a registry server. The supported platform model is external Registry Connection integration:

- Harbor, Distribution Registry, Docker Hub, GHCR, ECR, GCR, ACR and comparable providers are adapters behind a
  Registry Connection.
- Artifact Repository is separate from Connection so namespace, authorization and repository policy do not collapse
  into credential configuration.
- A future system-app installer may provision Harbor or Distribution, then create an ordinary Registry Connection; that
  is not a Graft Registry implementation.

This avoids registry storage, replication, garbage collection, token service, OCI compatibility and security-patch
ownership while preserving a coherent Build-to-Deployment user experience.

## 6. Scheduling, Parallelism And Platforms

Builder Pools are a later Build scheduling layer, not a replacement Task Runtime. They select an eligible Builder
Instance according to the frozen Execution Plan and record the selection as execution evidence. Round robin, least load,
label, affinity and region policies must be deterministic enough to audit and must fail closed when capacity telemetry is
stale or incompatible.

Multi-platform execution fans one Execution Plan out into platform-specific build legs, for example `linux/amd64` and
`linux/arm64`, then performs a final manifest publication after every required platform Artifact is available. This
requires explicit distributed-build support in Task Runtime before implementation; Build must not emulate a second
orchestrator.

This distributed-leg contract is an explicit follow-up Phase 6 authority gap. Task Runtime must own coordinated leg
identity, cancellation, retry/recovery and aggregate terminal state; Build owns the resulting platform Artifacts and
manifest Publication. Until that contract is released, a Build executor must not loop over platforms or create an
implicit fan-out scheduler.

### Phase 8A: Builder Telemetry Authority

The remaining scheduler gap is not a selection algorithm. Runtime Target currently exposes only capability and UI
summary projections; it does not expose a Build-facing, freshness-bounded fact for builder capacity, running builds,
queued builds, region or affinity. Build therefore must not derive `least_load`, `region` or `affinity` from CPU charts,
host load, endpoint names or static Builder labels.

Phase 8A defines `BuilderTelemetrySnapshot` and `RuntimeTargetBuilderTelemetryReader` as the narrow authority boundary.
Each snapshot is Runtime Target-scoped, carries capacity/load dimensions, `ObservedAt`, `ExpiresAt`, a source reference
and explicit region/affinity claims. Consumers must reject stale, unavailable or internally inconsistent snapshots. A
future Runtime/Infrastructure provider must publish and authorize these facts, while Build Scheduler binds the accepted
target fact to its Build-owned Builder Instance in placement evidence and replays the same decision without re-querying
mutable telemetry.

Until Phase 8A has a registered source implementation and integration evidence, only `round_robin` and static `labels`
selection remain executable; `least_load`, `region` and `affinity` stay fail-closed.

### Phase 8B: Builder Capacity And Telemetry Source Authority

Phase 8A's contract cannot be implemented from Runtime UI summaries, Monitor data, Docker container counts or Task JSON.
Phase 8B therefore introduces the missing durable authority rather than treating any of those projections as scheduler
input. Runtime/Infrastructure owns a provider-conformant, target-scoped telemetry source with provenance, freshness and
attested topology claims. Build separately owns atomic Builder Instance reservations, so physical target availability
and Graft's own concurrent allocations are never conflated.

Before freezing a placement, Build obtains a fresh target fact, selects an eligible Instance and atomically records a
fenced reservation. The accepted source/version, capacity facts and reservation reference become frozen scheduling
evidence. Task lifecycle completion, cancellation and restart recovery reconcile the reservation; provider execution
may reject an unavailable target but must not silently re-schedule it. A provider source must be real for its Runtime:
Docker requires a bounded Build agent/control plane or equivalent provider authority, Kubernetes requires its
BuildKit/Kaniko controller or API evidence, and remote builders require their provider API. Until one source meets this
release gate, `least_load`, `region` and `affinity` remain unavailable.

## 7. UI And API Information Architecture

The create flow is a platform workflow, not a Docker form:

```text
Workspace -> Snapshot -> Builder -> Template / Driver -> Destinations -> Execution Plan Review -> Submit
```

- **Workspace:** select a source and its authorized source-specific inputs.
- **Snapshot:** display the resolved commit/content digest and materialization availability before submission.
- **Builder:** show only authorized, healthy Runtime Target build capabilities with compatible Driver, locality and
  platform support; a Pool may be selected only when policy permits it.
- **Template / Driver:** collect Template-defined parameters and show the selected compatible Driver, never raw Docker
  endpoint or arbitrary filesystem fields.
- **Destinations:** select authorized Repository and immutable/rewriteable Publication references without credential
  values.
- **Review:** render the immutable Execution Plan, including Snapshot, Builder selector, platforms, cache, labels,
  secret references and policy outcome. Submit freezes this plan.

REST resources are `Workspaces`, `Workspace Snapshots`, `Build Templates`, `Build Drivers`, `Builder Profiles`,
`Builder Instances`, `Builder Pools`, `Registry Connections`, `Artifact Repositories`, `Artifact Destinations`,
`Execution Plans`, `Build Jobs`, `Artifacts` and `Publications`. Artifact endpoints expose immutable digest facts and
never resolve mutable Publication references; Build Job endpoints submit, list, detail, cancel and retry; scheduler
policy is administered through Builder Pools rather than a detached Scheduler API. New write contracts
use resource references and selected typed parameters, replacing direct `context_path`, `dockerfile_path`, free-form
endpoint and credential inputs. Legacy Jobs remain readable only.

## 8. Security, Compatibility And Non-goals

- Direct v2 write-contract replacement is required. Do not keep aliases or hidden compatibility fields for the
  Docker-first create API.
- All source and destination access is checked at submission and at execution using the owning authority. Build carries
  references and redacted evidence only.
- Managed target-local sources require explicit Target/root authorization and a capture path; remote execution requires
  an explicit transferable materialization. No API accepts an arbitrary host path.
- No registry server, independent Build worker/queue/log system, user-configured Runtime connection, or generic
  workflow engine is introduced.

## 9. Provider Execution Evolution

### Phase 9C: Provider Execution Foundation (additional gap closure)

Phase 9B proves target-bound Docker execution, but non-Docker runtimes still lack a complete provider-neutral
execution evidence contract. This phase closes that prerequisite without claiming that a Kubernetes, BuildKit or
Kaniko adapter exists.

Scope:

- Define the private provider contract for connection acquisition, Snapshot import/materialization, identity
  verification, Driver invocation, Artifact/Publication evidence, credential execution mode, cancellation, timeout
  and recovery fencing.
- `TargetBoundProviderDriverExecutionCapability` is the provider-neutral Driver boundary: it consumes only frozen
  Snapshot identity, verified delivery proof and platform/Driver selection, emits through `BuildDriverLogSink`, then
  returns digest-addressed execution evidence. It is deliberately separate from the Docker reference-provider
  interfaces.
- Keep Runtime Target as the only authority for provider connection and credential facts; Build and Task metadata
  receive opaque references and redacted evidence only.
- Require every advertised Driver and Snapshot delivery mode to be backed by a registered provider implementation and
  health check before the target is selectable.
- Provide conformance tests and a failure taxonomy for unsupported providers, unavailable connections, Snapshot digest
  mismatch, credential rejection, cancellation and uncertain external outcomes. Unsupported paths fail closed and never
  fall back to Local Docker.
- Define the provider handoff for per-platform digest settlement and final manifest publication so future adapters
  reuse the existing Artifact and Publication authorities.

**Release gate:** the contract, capability-registration rule and conformance suite reject every unsupported or
unverifiable provider path, while existing Docker adapters continue to pass and no endpoint, credential or host path
crosses the Build/Task boundary.

Foundation delivered: `TargetBoundProviderExecutionConformanceCapability` receives only a frozen target, Driver,
platform, Snapshot identity and delivery mode. The Runtime Target Docker provider is registered as the reference
implementation; v2 execution requires complete delivery, Driver, publication, cancellation and cleanup evidence before
it calls Snapshot delivery. Build persists the non-secret conformance result per Execution Plan and Task stage as an
append-only evidence fact. `TargetBoundDockerBuildProvider` aggregates the Docker reference adapter only; future
non-Docker providers must use a provider-neutral Driver contract and cannot inherit Docker-specific types.

### Phase 9D: Concrete Non-Docker Provider Adapters

After Phase 9C, implement one provider at a time, such as Kubernetes BuildKit or Kaniko, behind the foundation
contract. Each adapter must own target connection/credential resolution, receive the exact immutable Snapshot, execute
the selected Driver, support cancellation/recovery, and return digest-preserving Artifact/Publication evidence. A
provider becomes selectable only after its integration proof passes; all other providers remain fail-closed.
