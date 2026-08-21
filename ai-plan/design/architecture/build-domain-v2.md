# Build Domain v2 Architecture

## 1. Purpose And Authority

Build Domain v2 makes Build a platform capability rather than a Docker command form. It supersedes the Docker-first
write model in [Docker Build Center](docker-build-center.md) while retaining completed legacy jobs as read-only history.

The execution authority for Registry credentials, Build capability matching, reservations, telemetry, Placement and
failure handling is [Build Domain v2 Credential And Telemetry Authority RFC](build-domain-v2-credential-and-telemetry-authority.md).
Provider integration seams are defined separately by the [Provider SDK And SPI RFC](build-domain-v2-provider-sdk-spi.md).
Where this architecture's historical phase wording conflicts with that RFC, the RFC wins. In particular,
`docker-runtime-store` is historical evidence only, Builder/Registry-local failure is not global availability, and
Pool/dynamic placement exposure follows the RFC's four-phase gates. References below to former numbered delivery slices
(`Phase 5` through `Phase 10`, including `9A` through `9D`) are historical implementation evidence only; they do not
create release gates. Their explicit recovery mapping is: Phase 5--7 map to RFC Phase 1 (credential and manual
reservation boundary); Phase 8 and 9A map to RFC Phase 2 (intent materialization and provider conformance); Phase 9B
maps to RFC Phase 3 (authority evidence and controlled promotion); and Phase 9C, 9D and 10 map to RFC Phase 4
(dynamic placement expansion). The RFC phase names remain normative wherever historical slices use different wording.

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

### Batch 5 execution boundary (current)

Build Docker side effects are now external Task stages. Build submission freezes only provider-neutral intent, placement,
capability expectation and opaque Snapshot/materialization identity; it never stores a Docker endpoint, credential,
host path, command or SDK value. Task Runtime owns the external execution lease, fencing, renewal, cancellation, bounded
logs, result digest, receipt, retry and recovery state. After a valid fence, the Build owner resolves one-shot
`build-execution-material/v1` in memory and the Docker Runtime Agent executes the exact `docker/v1` capability through
its Moby/OCI SDK adapters. The Agent returns `build-execution-result/v1` before terminal receipt settlement; Build alone
interprets and persists Artifact/Publication semantics, while Task retains only the result digest for exact replay.

The Batch 5 operation set is `build.image.local.v1`, `build.image.publish.v1`, `build.manifest.publish.v1` and
`build.artifact.copy.v1`. No server-local Build Docker/CLI adapter, fallback or compatibility alias remains. The shared
snapshot root is a named deployment volume mounted at `/tmp/graft-build-snapshots` in server and Agent; the path is
deployment topology, not persisted Task or Build domain data. The server Docker socket remains only for explicitly
unmigrated Update Controller, Runtime Target discovery/summary and Container read/stream/interactive boundaries until
their batches complete.

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

`Builder Pool` is an optional collection of eligible Builder Instances and Build-owned placement policy. It selects
Instances, not Runtime Targets directly. Phase 1 permits only manual single-Instance selection; Phase 3 can expose
static Pool policies; Phase 4 alone can expose dynamic policies backed by provider-conformant telemetry and a fenced
Build Reservation. A persisted round-robin cursor is an implementation asset, not proof that a policy is externally
available. Labels remain static eligibility; `least_load`, `capacity`, `affinity` and `region` are latent/disabled until the RFC
release gate.

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
构建、临时平台引用发布和最终 Manifest 结算。每个 leg 使用其自身冻结的 Builder Placement；同一 Execution Plan
可以因此包含多个已选 Builder Instance 和 Runtime Target。历史实施切片曾将 Builder Pool 的平台级 placement 提升
为可审计的多 Runtime Target 调度；在该能力未发布的路径中，计划仍只使用一个手工选择的 Builder Instance，避免
把未实现的负载或地域判断伪装成调度结果。

`Builder Placement` is Build's persisted, per-`ExecutionPlanID` and per-platform authority record. Its stable key is
the immutable Execution Plan identity plus target platform, and its canonical serialized form contributes to the
Execution Plan digest. Each record freezes Builder Instance, Runtime Target, `PolicyID`, `PolicyVersion`, deterministic
seed when applicable, policy-input fingerprint, selected output, capability profile/version, full
capability-negotiation result, telemetry source/observation identity when applicable, selected
`WorkspaceLocality`, selected `SnapshotDeliveryMode`, provider-backed Snapshot delivery proof and proof fingerprint,
and the Reservation fence. Task Runtime and Build executor read that one persisted record as the leg input; they never
maintain an independent recovery selection. A retry or replay reuses exactly that placement and its frozen delivery
evidence, while only an explicit new scheduling flow may create a new placement record and Execution Plan. Workspace Snapshot
已经由 Build 接管，因此 target 必须明确声明 `build-snapshot` locality 才能被 placement 选中；这不是 endpoint 或
任意目录的兼容通道。

Placement 只表达并冻结调度决定，不能证明一个 Provider 能消费该 Snapshot。Runtime Target 还必须声明
`SnapshotDeliveryModes`；`WorkspaceLocalities` 只表示物化位置，不等于传输适配器。历史 Phase 9 evidence owns provider-backed Snapshot
delivery and remote execution: a provider receives or materializes the exact immutable Snapshot through a declared
delivery mode, verifies its identity, then executes its declared Driver. The Docker provider now proves the same
contract for local Unix-socket and validated remote TCP/SSH targets by invoking the selected target explicitly and
transferring the immutable Snapshot context through the Docker client. Build must never receive endpoint or
credential facts or compensate with a local target fallback. The remaining non-Docker work is split into a provider
execution foundation (Phase 9C) and concrete Kubernetes/BuildKit/Kaniko adapters (Phase 9D); capability declarations
alone are never treated as execution proof.

The Phase 9 foundation uses `TargetBoundWorkspaceSnapshotDeliveryCapability` as the provider boundary. Build passes only
the frozen Snapshot identity, content digest, selected target and declared delivery mode inside the execution-private
call. The Docker Runtime Agent SDK adapter accepts the declared snapshot delivery mode, verifies the managed Build
snapshot root and returns a matching delivery proof before SDK execution.
Unsupported targets and connection kinds remain fail-closed; they must never fall back to an unbound local Docker
process or server-side adapter.

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
- A Registry Connection owns endpoint, credential reference, availability and lifecycle management. An Artifact
  Repository owns one unique repository path and its pull/push policy within that connection. User assignments are a
  separate many-to-many use-authority relation owned by the Registry module; granting a creator use access does not
  grant Connection or Repository management permissions.
- Repository management is reached from the Registry Connection detail surface rather than a separate first-level
  navigation item. Repository creation may atomically grant the creating user use access by an explicit, default-on
  request option. Assignment management is repository-centric in the first release and supports bounded batch
  replacement; a reverse user-centric view remains a future projection.
- A future system-app installer may provision Harbor or Distribution, then create an ordinary Registry Connection; that
  is not a Graft Registry implementation.

This avoids registry storage, replication, garbage collection, token service, OCI compatibility and security-patch
ownership while preserving a coherent Build-to-Deployment user experience.

## 6. Scheduling, Parallelism And Platforms

Builder Pools are a later Build scheduling layer, not a replacement Task Runtime. They select an eligible Builder
Instance according to the frozen Execution Plan and record the selection as execution evidence. The RFC gates manual
single Builder, static Pool and dynamic Placement separately. Dynamic selection must fail closed for stale or
incompatible provider telemetry; a static label selector is not dynamic telemetry.

Multi-platform execution fans one Execution Plan out into platform-specific build legs, for example `linux/amd64` and
`linux/arm64`, then performs a final manifest publication after every required platform Artifact is available. This
requires explicit distributed-build support in Task Runtime before implementation; Build must not emulate a second
orchestrator.

Task Runtime owns coordinated leg identity, cancellation, retry/recovery and aggregate terminal state; Build owns the
resulting platform Artifacts and manifest Publication. Each platform leg is an external lease with a frozen Runtime
Target capability binding; the aggregate manifest stage is also external and consumes only Build-owned platform result
facts. Build does not create a second fan-out scheduler.

### Telemetry And Reservation

`RuntimeTargetBuilderTelemetryReader` is the sole Build-visible telemetry facade. A historical generic signed ingress
does not prove the required Docker Runtime Agent source, so it is not Provider admission evidence. Runtime UI summaries,
Monitor data, Docker container counts, Task JSON, CPU charts, host load, endpoint names and static labels cannot supply
Builder-scoped queue, slot, freshness and provenance facts. The authority RFC defines `BuilderTelemetryProvider`,
`BuildExecutionCapability`, `CapabilityMatcher` and Build-owned fenced `BuilderReservation`; dynamic policy remains
disabled until the real Provider admission, slot-aware Reservation and recovery gates pass. Historical dynamic Pool rows
remain readable but are not executable.

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
unverifiable provider path, while the Docker SDK adapters execute only in `docker-runtime-agent`; no endpoint,
credential or host path crosses the Build/Task boundary.

Foundation delivered: `TargetBoundProviderExecutionConformanceCapability` receives only a frozen target, Driver,
platform, Snapshot identity and delivery mode. The Runtime Target Docker capability is admitted only when the bound
Agent generation advertises `oci-build`/`docker/v1`; the Agent SDK path provides delivery, Driver, publication,
cancellation and cleanup evidence. Build retains only non-secret conformance and result evidence per Execution Plan and
Task stage. Future non-Docker providers must use the provider-neutral Driver contract and cannot inherit Docker-specific
types.

### Phase 9D: Concrete Non-Docker Provider Adapters

Future work may implement one provider at a time, such as Kubernetes BuildKit or Kaniko, behind the foundation contract.
Each adapter must own target connection/credential resolution, receive the exact immutable Snapshot, execute the selected
Driver, support cancellation/recovery, and return digest-preserving Artifact/Publication evidence. A provider becomes
selectable only after its integration proof passes; all other providers remain fail-closed.
