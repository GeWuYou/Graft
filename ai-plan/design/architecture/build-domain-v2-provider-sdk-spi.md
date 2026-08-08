# Build Domain v2 Provider SDK And SPI RFC

## Status And Scope

This RFC defines the compile-time Provider Software Development Kit (SDK) and Service Provider Interface (SPI) for
Build Domain v2. It is an extension contract for Runtime Target-bound build providers; it is not a new runtime,
resource model, scheduler, event bus, queue, health registry or evidence database.

The [Build Domain v2 Credential And Telemetry Authority RFC](build-domain-v2-credential-and-telemetry-authority.md)
owns Build capability, credentials, reservations, placement, telemetry semantics and failure classification. This
document defines how a provider supplies the facts and execution adapters those authorities consume. The existing
`TargetBoundProviderExecutionConformanceCapability` remains the provider-neutral execution conformance authority;
this SPI refines its implementation seams and does not replace it.

The first implementation may be Docker, BuildKit, Kaniko or another approved backend. A provider is selectable only
after its declared capabilities and conformance evidence pass the release gate for the current Build phase.

## Authority And Non-Goals

| Concern | Authority | Provider SPI responsibility |
| --- | --- | --- |
| Runtime Target connection, authorization and endpoint facts | Runtime Target | receive an opaque, already-authorized target handle |
| Build requirements, matching, placement and reservation fence | Build Domain | consume frozen inputs and return provider facts; never choose a target |
| execution state, retry, cancellation and recovery | Task Runtime | implement cancellation/recovery hooks and report constrained facts |
| secret storage and issuance | Secret Backend / Credential Provider | request a short-lived credential session through the credential adapter |
| telemetry read boundary | `RuntimeTargetBuilderTelemetryReader` | publish provider-scoped observations beneath that facade |
| Artifact digest and Publication lifecycle | Build Artifact / Publication | return digest-preserving receipts and redacted evidence |
| platform-wide availability | `PlatformAvailabilityStore` / `CapabilityCoordinator` | report local provider status only; never register Builder health globally |

Providers MUST NOT persist a second connection registry, Build Job state machine, scheduler, queue, event store, global
health state or evidence store. Queue state, resource accounting and future Build cost are Build-owned extension points;
the SPI may report bounded observations but cannot become their authority.

## Provider Identity And Lifecycle

Every provider has a stable compile-time `ProviderID`, a human-readable display name, a semantic `SDKVersion`, a
provider `ImplementationVersion`, and a set of declared Driver and delivery-mode identifiers. Provider identity is
not a Runtime Target identity and must not contain endpoint or credential data.

Registration and lifecycle are explicit and ordered:

```text
Registered -> Validated -> Active -> Degraded -> Unavailable -> Retired
                         |          |             |
                         +----------+-------------+
```

- `Registered`: the module has compile-time registered the provider descriptor; it cannot execute.
- `Validated`: descriptor, version, required adapters and conformance suite pass locally; target binding is still
  required.
- `Active`: at least one bound Runtime Target reports the provider capability as usable for an operation.
- `Degraded`: the provider remains registered but one or more declared capabilities are unavailable, stale or
  non-conformant; only unaffected static/manual operations may proceed.
- `Unavailable`: no operation may select the affected capability. Existing external executions follow Task Runtime
  recovery and are not guessed as successful.
- `Retired`: new bindings and execution are rejected; historical evidence remains readable for replay.

Runtime Target/Provider registry supplies binding and conformance facts, while the Build provider authority owns lifecycle
admission and transitions. Neither UI, Monitor, Builder Telemetry nor a Provider goroutine may transition lifecycle.
A provider MUST expose deterministic `Validate`, `Start` and
`Stop` hooks (or equivalent synchronous lifecycle methods) and MUST release all provider-owned resources on `Stop`.
Startup failure is fail-closed and local to the provider capability; it does not make the platform globally
unavailable.

## SPI Surface

The SDK exposes narrow, provider-neutral interfaces. Exact language bindings may use repository `moduleapi` types,
but the semantic inputs and outputs below are stable.

### Provider Descriptor And Lifecycle

```text
ProviderDescriptor {
  provider_id, sdk_version, implementation_version,
  driver_ids, snapshot_delivery_modes, capability_version,
  compatibility_contract
}

ProviderLifecycle {
  Validate(ctx, ValidationInput) -> ValidationResult
  Start(ctx, StartInput) -> error
  Stop(ctx, StopInput) -> error
}
```

`compatibility_contract` declares the Capability, Execution Context, Snapshot Delivery, Credential Adapter, Telemetry
and Evidence schema versions the provider consumes or produces, plus each relation's compatibility direction. `Validate`
MUST be side-effect bounded, reject an incompatible mandatory relation, and prove every advertised mandatory adapter. `Start` may prepare
provider-local clients, but may not start an independent scheduler or worker queue. `Stop` is idempotent; a failed
cleanup produces redacted evidence and leaves the capability unavailable until operator recovery.

### Capability Provider

`CapabilityProvider` evaluates one already-authorized Runtime Target and returns a versioned
`BuildExecutionCapability` profile: supported Drivers and diagnostic versions, platforms, multi-platform behavior,
cache import/export, provenance/SBOM, registry-login, Snapshot delivery, manifest publication and unsupported
dimensions. The provider reports facts; `CapabilityMatcher` owns requirement negotiation and eligibility.

Capability negotiation MUST preserve requirement intent (`required`, `preferred`, `optional`) and return a frozen
`NegotiatedCapability` with provider capability version, selected Driver/platform/delivery mode and deny reasons. A
`BuilderProfile` is static Build policy; capability is provider fact; telemetry is time-bound observation. They must not
be merged.

### Telemetry Provider

`BuilderTelemetryProvider` is consumed only through `RuntimeTargetBuilderTelemetryReader`. It may return
Builder-scoped running Builds, queue length, allocatable slots, health, cache state, source identity, observed-at,
expires-at, integrity summary and explicitly unsupported dimensions. It MUST identify the Builder scope and observation
window. Host CPU/memory, `docker stats`, whole-cluster capacity, UI summaries and container counts are diagnostics,
not Placement input.

Unknown, stale or unverifiable dynamic telemetry MUST be represented as unsupported and deny dynamic Placement. Static
manual selection can proceed only when capability and execution conformance remain valid.

For Phase 4 Docker admission, the provider MUST receive reports only from a provisioned Builder Agent bound to the
Runtime Target, Provider, Builder scope and capability profile/version. Reports require a monotonic sequence, bounded
clock skew, replay rejection and lifecycle state; running, queued and slot facts originate in the Agent's controlled
execution ledger or Driver controller. A generic signed ingress is insufficient. Scope, profile/version, provenance,
integrity and unsupported dimensions MUST validate together before dynamic Placement; otherwise the Provider stays
manual/static only and historical dynamic policy data remains readable but non-executable.

### Credential Adapter

`CredentialAdapter` bridges the existing Credential Provider to execution without exposing secret plaintext:

```text
Prepare(ctx, CredentialRequest) -> EphemeralCredentialSession
Inject(ctx, session, ExecutionInjectionTarget) -> InjectionReceipt
Revoke(ctx, session) -> CleanupReceipt
```

The request contains opaque `credential_ref`, endpoint/repository scope, push operation and expiry window. The session
is short-lived and operation-scoped. The adapter MUST use an isolated per-operation context (for example temporary
`DOCKER_CONFIG`, restricted secret mount or workload identity), MUST never use an environment default credential store,
and MUST redact all receipts. Missing scope, unsupported injection, expiry or unverifiable cleanup rejects execution.
`Revoke` is idempotent and required on success, failure, cancellation, timeout and recovery; cleanup failure is a
security failure and blocks reuse.

### Workspace Adapter

`WorkspaceAdapter` consumes only the immutable Snapshot identity/digest and a declared delivery mode:

```text
Deliver(ctx, SnapshotDeliveryRequest) -> SnapshotDeliveryReceipt
Cleanup(ctx, SnapshotLease) -> CleanupReceipt
```

It MUST verify the delivered digest before Driver invocation, isolate each execution materialization and limit cleanup
to provider-owned bytes. It MUST NOT mutate Workspace Snapshot, Execution Plan, Artifact or Publication identity, and
must reject arbitrary host paths. Reuse, clone, copy-on-write and cache are implementation strategies keyed by Snapshot
identity, not new authority.

### Execution Adapter

`ExecutionAdapter` receives a frozen Driver/platform selection, verified Snapshot delivery receipt, ephemeral credential
session/receipt, execution context and reservation fencing token:

```text
Execute(ctx, ExecutionRequest) -> ExecutionReceipt
Cancel(ctx, ExecutionHandle) -> CancelReceipt
Recover(ctx, RecoveryRequest) -> RecoveryReceipt
```

Execution Context is a provider-neutral, immutable input envelope for approved environment, labels, timeout, proxy,
mirror, registry alias and cache namespace values. It excludes endpoint details, secret values and arbitrary paths.
The adapter MUST honor fencing, cancellation and timeout, return digest-preserving receipts, and surface uncertain
external state as `Unknown` for Task Runtime recovery. It MUST not retry or reschedule independently.

### Evidence Writer

`EvidenceWriter` appends redacted provider facts through the existing owning authority; it is not a database:

```text
Write(ctx, ExecutionEvidenceFact) -> EvidenceReceipt
```

Every fact links Plan, Task/Stage/leg/attempt, Builder Identity, reservation fence, provider capability version,
timestamps and a redacted integrity summary. Evidence MUST omit credentials, endpoint secrets, Docker config, host
paths and secret-derived content. Replay consumes the frozen facts and never re-queries mutable telemetry or secrets.

## Conformance Test Suite

The SDK supplies a repository test harness, but the provider owns its fixtures and does not add a runtime test service.
Conformance is a release gate for the advertised capability set:

**MUST pass**

- descriptor identity and SDK/capability version validation;
- lifecycle ordering, idempotent Stop and no hidden worker/scheduler;
- capability negotiation, unsupported dimensions and deterministic deny reasons;
- credential scope, isolated injection, redaction, expiry and cleanup on success/failure/cancel/timeout/recovery;
- immutable Snapshot delivery, digest verification, per-execution isolation and bounded cleanup;
- execution cancellation, timeout, fencing-token rejection, recovery and Unknown outcome handling;
- digest-preserving Artifact/Publication receipt and append-only redacted evidence;
- telemetry Builder scope, freshness, provenance, integrity and explicit unsupported dimensions;
- local failure isolation from global platform availability.

**SHOULD pass**

- verifiable cache state and health diagnostics;
- idempotent execution receipts and Builder-scoped capacity explanations;
- deterministic replay of negotiation, policy version and placement evidence;
- provider upgrade tests against retained historical evidence.

**MAY provide**

- Agent projections, provider-specific optimization and read-only diagnostics that cannot affect Placement or
  lifecycle authority.

Unknown or failed conformance means the provider is `Degraded` or `Unavailable`; it may only serve explicitly allowed
static/manual paths. No capability may be advertised merely because a driver binary or endpoint exists.

## Version And Compatibility Contract

The provider declares separate `SDKVersion`, `ProviderCapabilityVersion`, `DriverVersion` and
`EvidenceSchemaVersion`. A Driver version is diagnostic and cannot substitute for capability compatibility.

| Change | Compatibility | Required behavior |
| --- | --- | --- |
| additive optional field or diagnostic metric | forward/backward compatible | older readers ignore it; provider declares default semantics |
| additive capability marked optional | compatible | matcher treats it as absent when unsupported |
| changed required capability, credential scope, delivery semantics or evidence meaning | breaking | increment capability/schema major; old plans are not silently reinterpreted |
| provider implementation upgrade with unchanged contract versions | compatible only after conformance | bind new instances after validation; retain old execution evidence |
| retired capability or driver | breaking for new selection | historical plans remain readable; execution rejects unless the frozen contract is still proven |

Provider upgrades MUST declare whether they are forward-compatible for frozen Execution Plans and replayable Evidence.
When compatibility cannot be proven, new execution and dynamic Placement fail closed; in-flight work follows Task Runtime
recovery, never an implicit provider swap. A migration may explicitly re-conform a frozen placement, but may not change
its Target, credential scope or Snapshot identity without creating a new Plan under Build authority.

`PolicyVersion` is part of Placement Evidence so replay can explain a selection; it is not a Provider lifecycle state.

## Failure And Security Rules

The SPI maps provider failures to the existing Task Runtime taxonomy (`Transient`, `Permanent`, `Configuration`,
`Authorization`, `Infrastructure`, `Provider`, `Internal`, `Unknown`). Providers MUST return a typed category and
redacted diagnostic; they MUST NOT claim success for an unknown external outcome or retry outside Task Runtime policy.

Credential, Snapshot, fencing, digest, isolation or cleanup violations are fail-closed security/provider failures. No
fallback to a local target, default Docker credential store, alternate Builder, stale telemetry or unapproved delivery
mode is permitted. Provider-local degradation does not update `PlatformAvailabilityStore` or trigger a global service
unavailable page.

`credential_cleanup_unverified` is an `Internal` failure with a `Needs Attention` disposition. Providers return only its
stable code and redacted cleanup evidence; Task Runtime maps the constrained outcome, does not auto-retry it, and does
not reuse its credential session or Reservation fence.

## Adoption Gates

1. **Phase 1:** register one validated provider, manual single Builder, capability negotiation, secure credential
   injection, Snapshot delivery and minimum reservation evidence.
2. **Phase 2:** complete Builder Instance/Profile, Driver, Template and Workspace adapter behavior; Pool dynamic choice
   remains disabled.
3. **Phase 3:** permit static Pool `RoundRobin`, auditable `Random` and `Manual` only; telemetry is diagnostic.
4. **Phase 4:** permit conformant telemetry, reservation recovery and dynamic `LeastLoad`, `Capacity` or `Affinity`.

Future Provider integrations (for example Podman, remote Build, GitHub Actions or Tekton) implement this SPI and pass
the conformance suite before entering Matcher or Placement. They do not modify Build Domain authority or introduce a
second orchestration path.

## Platform Architecture Review Record

- `authority_summary`: Runtime Target owns connection and provider binding; Build owns requirements, matching, Profile,
  placement, PolicyVersion, reservation and evidence association; Task Runtime owns execution lifecycle; existing
  Credential Provider and telemetry facade remain the only read/secret boundaries.
- `platform_fit`: compile-time registration and narrow adapters preserve modular-monolith runtime surfaces; no dynamic
  plugin marketplace, second DI mechanism, scheduler, event bus or global health registry is introduced.
- `lifecycle_summary`: Provider lifecycle is local and explicit; Task Runtime remains the only execution state machine;
  cancellation, retries and recovery are delegated to it.
- `deletion_candidates`: provider-specific Build-side credential, endpoint, queue, health and evidence stores remain
  unnecessary and must not be added; future adapters should remove any Docker-only branching they replace.
- `decisions_needed`: concrete Provider SDK language bindings and registration location are implementation follow-ups;
  this RFC deliberately defines semantics only.
- `acceptance`: every provider passes MUST conformance, unsupported behavior is fail-closed, replay uses frozen evidence,
  and local provider failure leaves global platform availability unchanged.
