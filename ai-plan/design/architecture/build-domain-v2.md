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
content; retention only affects availability, never its identity.

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
`Execution Plans`, `Build Jobs`, `Artifacts` and `Publications`. Build Job endpoints submit, list, detail, cancel and
retry; scheduler policy is administered through Builder Pools rather than a detached Scheduler API. New write contracts
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
