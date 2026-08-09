# Build Domain v2 Trace

## 2026-08-08: Phase 3 Static Pool Release Gate

- Authority discovery kept placement policy in Build, capability facts in Runtime Target, execution lifecycle in Task
  Runtime, and the wire enum in `openapi/**`; no downstream compatibility projection was added.
- Static Pool exposure is now limited to `manual`, persisted `round_robin`, and deterministic seeded `random`. Labels
  are eligibility evidence only. Placement freezes policy/version, candidate fingerprint, selected Instance, and the
  cursor or seed; BuilderReservation fencing and immutable Workspace Snapshot/Materializer ownership remain unchanged.
- Historical dynamic-policy rows remain readable for migration compatibility but are rejected by new Build writes and
  omitted from the selector projection. No telemetry, LeastLoad, Capacity, Affinity, Region, second Task Runtime,
  scheduler, Build queue, event store, or global health registry was introduced.
- Validation: `go run ./cmd/graft validate backend`, `bun run check`, `just openapi-check`,
  `python3 scripts/validate_sql_migrations.py`, and focused Build tests passed.

## 2026-08-08: Phase 4 Pre-Gate Recovery Reconciliation

- Authority review found no missing Phase 3 implementation: the static Pool gate is satisfied by the Build-owned
  `manual`, `round_robin` and deterministic `random` paths, frozen placement evidence, and the OpenAPI static enum.
- Tracking and trace loop state had remained at an earlier recovery point. Phase 2 intent/materialization conformance
  and Phase 3 static Pool placement are now recorded as completed; the only pending batch is Phase 4 dynamic placement
  and distributed Build.
- This reconciliation changes recovery metadata only. It does not enable telemetry, dynamic policies, Region, a second
  scheduler/runtime, or any Build-owned queue or event store.

## 2026-08-07 phase-1-secure-credential-execution-and-manual-reservation implementation

- Replaced the new v2 publication execution edge with the `RuntimeExecutionAdapter` contract. Runtime Target requests a
  short-lived `CredentialProvider` session, creates an isolated per-operation Docker config, and revokes/removes it on
  every adapter terminal path. New publication rejects historical `docker-runtime-store` execution mode; no environment
  default or platform-wide availability mutation is introduced.
- Routed OCI artifact-promotion copy through the same adapter. Source pull and destination push each receive their own
  scoped ephemeral session in one isolated Docker config; direct capability registration was removed so a caller cannot
  select the historical Docker-store path through DI.
- Added Build-owned `build_builder_reservations` capacity leases. Execution-plan materialization reserves the selected
  manual Builder and freezes a deterministic fence; the Task executor transitions the matching lease to `running` and
  releases or marks it `abandoned` without creating a second Task Runtime or scheduler.
- Tightened adapter cleanup semantics: credential revoke and isolated Docker directory cleanup are now part of the
  returned execution error, and focused Runtime Target tests cover provider failure and expired-session paths.
- Reservation fence derivation is attempt-scoped and deterministic; a retry receives a new reservation row and fence
  after the preceding live attempt is abandoned.
- Core now registers a file-backed `CredentialProvider` only when the explicit absolute
  `GRAFT_REGISTRY_CREDENTIALS_FILE` source is valid. The provider reloads an expiring, endpoint/repository/operation-
  scoped entry for every request, exposes only an opaque session and writes Docker auth only into the adapter-created
  `0700` directory. Missing configuration still omits the adapter; an invalid configured source prevents Runtime Target
  registration. No default credential store or environment authentication is introduced.
- Focused provider, core DI and Runtime Target registration tests cover source validation, scope mismatch, expiry,
  revocation, isolated-target permissions and plaintext-free failure messages. A live registry publication remains a
  deployment conformance check requiring an operator-provisioned expiring secret file; it was not claimed by local
  tests.
- Validation evidence: `python3 scripts/validate_sql_migrations.py`; focused Go tests for `internal/moduleapi`, Build,
  Registry, Runtime Target and Container passed. Full backend, OpenAPI and web validation is recorded with this slice.

## 2026-08-06 authority-bootstrap

- Work Intake classified Build Domain v2 as a long-running cross-boundary feature requiring repository-level design,
  roadmap, active topic recovery and loop-driven phased delivery.
- The canonical model is `Workspace -> Workspace Snapshot -> Execution Plan -> Task execution -> Artifact ->
  Publication -> Deployment`.
- Docker-first direct create fields are historical-only; future writes use the v2 resource model.
- Builder capability remains Runtime Target authority; Builder Profile/Instance/Pool are logical Build resources and
  do not create another connection directory.
- The loop controller accepted the batch after `git diff --check`, the bounded ai-plan structure validation and
  commit `084ae531`.

## Locked Decisions

- A retry uses the frozen Snapshot and Execution Plan; rebuilding current source is explicit new submission.
- Artifact identity is immutable digest evidence; tag, manifest, promotion and copy are Publications.
- Registry Connection and Artifact Repository are Infrastructure resources; OCI Registry is only the first Artifact
  Destination provider, not the universal output model.

## 2026-08-06 phase-1-single-builder recovery

- Runtime Target build-candidate capability implementation is validated but intentionally uncommitted while the
  original Phase 1 batch is incomplete.
- The repair scope expands to Container build/push execution, the canonical Build write route, and controlled
  Application Workspace materialization; direct replacement remains mandatory.
- Existing user-owned route-prefix work is preserved and incorporated rather than reverted.

## 2026-08-06 phase-1-single-builder implementation recovery

- The first recovered worker completed target capability checks, v2 submission input shaping and private
  Application Workspace snapshot materialization. Focused Build, Project, Runtime Target, Container and
  moduleapi tests passed after the authorized import repair.
- Controller verification found the release gate still unmet: the submitted v2 plan is not persisted as an
  independent aggregate, the artifact and publication authority is still legacy job-shaped, and the canonical
  OpenAPI/web create contract still accepts Docker-first fields.
- The next repair round retains the same logical batch and expands only to the required vertical path: Registry
  Connection/Artifact Repository infrastructure authority, Build plan/artifact/publication persistence,
  target-selected Container build/push execution, and direct v2 write-contract replacement. Existing legacy
  job records remain read-only and the route-prefix correction remains preserved.

## 2026-08-06 phase-1-single-builder authority integration

- Registry Connection, Artifact Repository and explicit actor publication assignments are now Infrastructure-owned
  records exposed through a narrow non-secret `RegistryDestinationResolver`.
- Build v2 submission resolves and canonicalizes that destination before freezing the Execution Plan; the Build
  module now depends on Registry and never stores endpoint or credential data in task metadata.
- Container and the legacy executor now expose and prefer a Runtime Target-bound Docker build capability. The
  capability revalidates target identity, provider, health and driver before entering the controlled Docker path.
- The v2 executor remains fail-closed until an opaque Registry-owned publication binding and artifact/publication
  settlement path exist. This is an intentional authority boundary, not a compatibility fallback.

## 2026-08-06 phase-1-single-builder validation and target correction

- The v2 executor now performs target-bound Docker build and OCI publication, then settles a digest-addressed Artifact
  and its mutable Publication in one Build-owned transaction.
- Runtime Target selection was narrowed to the system-managed Local Docker target. A remote Docker target must not
  silently execute through Container's local Docker process; it requires the provider-owned execution adapter now
  implemented by this Phase 1 repair.
- Registry endpoint and opaque credential references remain execution-only. The historical environment/credential-helper
  path is retained only as evidence; new publication uses the explicit credential execution contract below.
- Focused and full Go tests, backend validation, OpenAPI validation, web validation, migration gates and diff checks
  passed. Existing backend DTO-boundary warnings are unrelated baseline warnings.

## 2026-08-06 phase-1-registry-credential-execution

- The historical Phase 1.5 mode delegated authentication to the selected Docker Runtime credential store. That path is
  superseded for new writes by the ephemeral credential session and isolated adapter contract; historical evidence
  remains readable only.

## 2026-08-06 phase-1-registry-credential-execution validation

- Registry publication bindings now carry a typed `AuthExecution` mode. Phase 1 selects the Docker Runtime credential
  store mode, and Container rejects unsupported modes before target execution.
- Build Publication evidence records only the selected mode; credential references and values remain private to
  Registry/Runtime execution adapters.
- Focused Build, Container and Registry tests, backend lint, migration validation, OpenAPI validation and topic
  structure checks passed. Phase 2 is now the next bounded batch.

## 2026-08-06 phase-1.75-snapshot-materialization-retention

- Phase 2 exploration identified a missing foundation rather than a blocker: Application's source adapter created an
  unowned temporary directory and Build stored its absolute path, while global content-digest uniqueness could merge
  Snapshots from different provenance.
- Build now adopts the source adapter's temporary capture into its managed Snapshot directory and removes unreferenced
  copies on failed or idempotently activated submissions. The execution reference remains private to Build persistence.
- Snapshot persistence records Build ownership, availability state and retention policy. The global digest uniqueness
  constraint was removed so source provenance remains part of immutable Snapshot identity.
- This is a completed foundation phase; generic Workspace resources, explicit expiration/purge operations, Git and
  archive materializers remain Phase 2 authority work rather than being inferred from an Application filesystem path.

## 2026-08-06 phase-2-workspaces-templates-drivers

- Build now owns a reusable Workspace aggregate and persists Snapshot ownership through `workspace_id`; Project remains
  a narrow authorized Application Workspace source adapter.
- v2 submission and the create UI consume `workspace_id`. A Build Workspace creation endpoint validates the Application
  source before persisting its stable source reference; host paths never enter the request or response.
- Template and Driver intent is versioned and resolved through a Build-owned compatibility registry. The submitted Plan
  freezes canonical `oci-dockerfile/default@v1` and `docker-engine@v1` references, and execution revalidates the same
  compatibility contract against Runtime Target capabilities.
- Git, uploaded archives, generated materializers, explicit purge operations and remote transfer remain later bounded
  source/provider slices; no unsupported source is accepted by the current Application-only creation endpoint.

## 2026-08-06 phase-3-builder-resources-foundation

- Added Build-owned Builder Profile, Builder Instance and Builder Pool resource contracts and migration constraints.
- Profiles reference versioned Drivers and policy; Instances bind only a stable Runtime Target identity; Pools store
  selection policy and selectors without copying endpoint, credentials, Task state or creating a scheduler loop.
- Scheduling execution, load telemetry and multi-platform fan-out remain the next Phase 3 slice.

## 2026-08-06 phase-3-pools-scheduling-platforms

- Builder Pool membership is now an explicit Build-owned relation. It has stable priority ordering, soft-delete
  history and no ownership of Runtime Target connection data or Task state.
- The first supported policy is transactional `round_robin`: the persisted pool cursor is locked and advanced during
  selection, and Build revalidates the selected Runtime Target capability and caller assignment before it can be
  consumed by a future frozen Plan write contract.
- `least_load`, labels, affinity and region remain stored policy vocabulary only. They are not enabled until the
  corresponding telemetry and placement authority exists.
- A formerly implicit gap was initially tracked as Phase 5 contract work; the executable gap is now tracked as Phase 6:
  Task Runtime must first own coordinated distributed legs and their
  aggregate recovery semantics. Multi-platform fan-out and OCI manifest aggregation cannot run inside a single Build
  executor loop.

## 2026-08-06 phase-3-pool-selection-contract

- The public v2 create contract now accepts exactly one builder selector: a direct Runtime Target or a Build-owned
  Builder Pool. Pool selection runs before plan freezing and rechecks Driver, platform, Runtime Target capability and
  caller assignment.
- The frozen Execution Plan records both the selected Pool and the concrete Instance when Pool selection is used; the
  selected Runtime Target remains the execution adapter identity. This keeps retry and audit independent of later Pool
  membership changes.
- The current Application Workspace adapter still requires the selected target to match its authorized local source
  locality. Remote transfer and cross-target Workspace materialization remain unsupported rather than being inferred.

## 2026-08-06 phase-5-task-runtime-contract-foundation

- Task Runtime now owns a versioned `CoordinatedTaskPlan`/`CoordinatedLegPlan` contract with validation for unique leg
  and platform identities, stable Builder/Runtime bindings and aggregate stage identity.
- The initial contract deliberately rejected coordinated submission until the persisted coordinator foundation was
  available. Build continues to reject multi-platform submissions before Task creation because per-platform Builder
  selection and manifest publication are not yet complete; no executor-local loop or second scheduler is introduced.

## 2026-08-06 phase-6-distributed-leg-coordinator-gap

- The remaining implementation gap is explicitly promoted to Phase 6: Task Runtime must persist coordinated legs,
  claim them transactionally in parallel, propagate cancellation, aggregate terminal state and preserve recovery
  facts; Build then aggregates immutable platform Artifacts into a manifest Publication.
- Multi-platform submission remains fail-closed until the Phase 6 release gate is satisfied.

## 2026-08-06 phase-6-parallel-leg-foundation

- `task_stages` now persists an optional coordination group and stable leg identity. The claim query permits separate
  workers to claim pending legs in the same group without relaxing serial ordering for normal stages.
- Runtime in-process tracking now keys running work by Stage rather than Task, so cancellation can reach every tracked
  leg of a coordinated task. `SubmitCoordinated` materializes the consumer's one-stage template into those Task
  Runtime-owned legs.

## 2026-08-06 phase-6-untracked-leg-cancellation

- A cancellation request received by an API instance without local worker ownership now atomically settles every
  running leg of the coordinated Task. This prevents a multi-instance cancel from releasing only one leg while leaving
  the owner Task active.
- Restart recovery already cancels every requested running Stage before settling the parent Task. Remaining Phase 6
  release work is per-platform Artifact result ingestion and Build-owned manifest publication.

## 2026-08-06 phase-6-platform-artifact-authority

- Build now persists each coordinated leg result as an immutable per-platform Artifact relation. A plan cannot attach
  two results to the same leg or platform, and retry can only replay the same digest.
- The current Docker adapter cannot build or publish an OCI manifest list. A Driver-owned manifest merge/publication
  capability remains the release gate before multi-platform Build submission is enabled.

## 2026-08-06 phase-7-manifest-input-foundation

- Build now constructs an OCI Manifest publication input only after every frozen platform has one immutable Artifact.
  Missing, duplicate or tag-derived platform results fail closed before any Driver invocation.
- The input contract is ready for a provider returning a trusted published Manifest digest. The Docker Engine adapter
  remains intentionally unregistered because its current controlled surface cannot prove that digest.

## 2026-08-06 phase-7-manifest-settlement-foundation

- Given a Driver-proven OCI Manifest digest, Build now rechecks platform completeness and atomically records the
  final immutable Manifest Artifact plus its mutable Publication. The settlement cannot bypass the per-platform
  Artifact gate.
- A concrete Driver adapter remains the final release dependency; no API flag or Docker command compatibility path is
  used to simulate it.

## 2026-08-06 phase-7-buildx-provider-adapter

- Container now owns a Buildx `imagetools` adapter that creates an OCI Manifest from immutable digest references, reads
  the raw registry Manifest and returns the inspected Manifest digest to Build.
- The adapter requires a Runtime Target to explicitly declare `docker-buildx`. The system-managed Local Docker Target
  continues to declare only `docker-engine`, so the public multi-platform gate remains unchanged until target
  capability provisioning and coordinated Build orchestration are completed.

## 2026-08-06 phase-7-coordinated-build-orchestration

- Build v2 now materializes one Task Runtime coordinated leg per frozen platform. Each leg uses its frozen Workspace
  Snapshot, publishes through a temporary platform reference, records a normalized immutable digest, and only then
  attempts manifest publication after Build confirms the full Artifact set.
- The remaining authority gap is not executor-local fan-out: a Builder Pool still freezes one compatible Instance for
  the plan. Phase 8 will add scheduler-owned per-platform placement across Runtime Targets, with replayable load,
  label, affinity and region evidence.

## 2026-08-06 phase-8-platform-placement-foundation

- Execution Plan now persists one immutable Builder Placement per platform, including the chosen Builder Instance,
  Runtime Target and scheduling-policy evidence. Task Runtime coordinated legs and Build execution resolve their
  target only from this frozen data.
- `build-snapshot` is a Runtime Target locality capability for Build-owned snapshot materialization. It prevents the
  old Application-target equality check from standing in for real source transfer. Provider implementations still
  need to prove cross-target materialization and execution before Phase 8 can claim distributed scheduling complete.

## 2026-08-06 phase-1.75-materialization-cleanup

- Build now leases expired Snapshot materializations before deleting only paths under its private snapshot root. A
  successful cleanup changes the materialization state to `purged` without deleting Snapshot, Execution Plan, Artifact
  or Publication evidence.
- Cleanup claims use a recoverable `purging` lease. Invalid paths and deletion failures return the Snapshot to
  `expired`, preventing a stale database reference from expanding filesystem deletion scope.

## 2026-08-06 phase-9-provider-delivery-gap

- Phase 8 now freezes a replayable per-platform Builder Placement, but the remaining constraint is not a reason to
  block the released placement foundation: it is a distinct provider capability gap.
- Phase 9 owns immutable Snapshot delivery and Remote Builder execution proof. A target's `build-snapshot` locality
  declaration remains a placement eligibility signal, not evidence that Remote Docker, Kubernetes, BuildKit or Kaniko
  can receive the materialization.
- The Local Docker provider remains intentionally local-only. A remote Target must stay fail-closed until its provider
  adapter transfers or materializes the frozen Snapshot, verifies its identity and executes the declared Driver.

## 2026-08-06 snapshot-retention-scheduling

- Build registers one retention Cron declaration that invokes its existing lease-based materialization cleanup method.
  The Cron runtime owns invocation; Build retains cleanup and immutable-evidence authority. No Build worker or queue
  is introduced.

## 2026-08-06 repair-authorization

- The user pre-authorized normal in-scope `execute_repair` actions for the active Build Domain v2 workspace. Repairs
  continue through ordinary authority, ownership and validation checks without repeated confirmation.

## 2026-08-06 phase-8-label-scheduling

- Builder Pools with `labels` policy now select the first deterministically ordered compatible Instance whose persisted
  labels satisfy the validated selector. Runtime assignment, availability, Driver, locality and platform checks still
  run before selection is frozen.
- The selector JSON is copied into `BuilderPlacement.SchedulingEvidence`, so the Execution Plan digest and retry path
  retain the scheduling decision even if Builder labels change later. `least_load`, `affinity` and `region` remain
  fail-closed until their telemetry authorities exist.

## 2026-08-06 phase-9-delivery-capability-foundation

- Runtime Target build summaries now expose `SnapshotDeliveryModes` separately from `WorkspaceLocalities`.
- The current Local Docker executor requires the explicit `target-local` delivery mode. `provider-transfer` is reserved
  for a provider-owned remote adapter and cannot be selected merely because a target advertises `build-snapshot`.
- Build execution now invokes the provider-owned delivery contract before target-bound Docker execution. The Local
  adapter accepts only Build-managed snapshot materializations and returns a matching Snapshot identity proof; no
  endpoint, credential or host path is copied into Task metadata or the public plan.

## 2026-08-06 phase-9b-remote-adapter-audit

- A read-only authority audit confirmed that Runtime Target currently exposes only the system-managed local Docker
  target. Container owns one global Docker runtime and local CLI build path; there is no target-scoped connection or
  credential resolver, Snapshot transfer/import API, or Kubernetes execution adapter.
- Phase 9B therefore remains an explicit next slice rather than an emulated implementation. Enabling
  `provider-transfer` requires adding those provider-owned authorities first; Build must remain fail-closed until the
  adapter can prove Snapshot identity, execute the selected Driver and settle the existing Artifact/Publication facts.
- Runtime Target now owns a private Docker connection lookup for provider-side use. This repairs the first missing
  authority without widening the public Build contract; no endpoint is returned by `BuildRuntimeTargetSummary`, and no
  remote target becomes executable until the complete adapter exists.
- Build now resolves `TargetBoundDockerImageBuildCapability` explicitly. Container registers the current Local Docker
  implementation under that boundary, so a future Runtime Target provider can replace it without changing Build's
  execution-plan or Task contracts.
- Runtime Target private connection lookup now fails closed for unavailable targets, missing `image_build` capability,
  malformed endpoint values and mismatched endpoint schemes. These checks stay below public Build projections.

## 2026-08-06 phase-9b-remote-docker-provider

- Runtime Target now registers the production target-bound Docker provider capabilities. Container retains only the
  legacy local capability and a fallback for standalone focused tests; production service registration does not
  create duplicate target-bound providers.
- Validated Unix-socket targets use `target-local`; validated TCP/SSH targets use `provider-transfer`. The provider
  proves the Build-managed Snapshot root and invokes Docker with the private target endpoint for build, image
  publication and Buildx OCI manifest publication. Build plans, Task metadata and HTTP remain endpoint/credential-free.
- Remote target selection now reads all assigned live Docker build targets and rejects malformed or unavailable
  connections before placement. No local Docker fallback is used. Kubernetes, Kaniko and BuildKit pod providers remain
  a future Phase 9D slice behind the Phase 9C provider foundation.
- Build submission and execution now carry the selected target's declared Snapshot delivery mode through the same
  Runtime Target reader. Remote `provider-transfer` plans therefore no longer fail against a hard-coded local mode;
  identity and requested-mode proof are both checked before the target-bound Driver runs.
- A regression test freezes the Phase 9D gate: a Kubernetes target declaring only `image_build` is still rejected until
  Runtime Target owns a Kubernetes connection/credential authority and a real provider adapter.

## 2026-08-06 phase-10-build-selector-read-model

- The create page no longer treats Workspace, Runtime Target or Builder Pool identifiers as Docker-form text fields.
- Build now exposes selector read endpoints and returns only non-secret projections. Runtime Target assignment remains the
  authority for target visibility; Builder Pools are visible only when a member target is authorized for Build and the Pool
  policy is currently executable (`round_robin` or validated `labels`). Unsupported load/affinity/region policies remain
  fail-closed and are not offered as selectable UI options.
- The frontend consumes those Build-owned contracts through generated OpenAPI types and TDesign Select controls, with
  loading, empty and error states. No Runtime Target private API is imported by the Build web module.
- Phase 10 is independently releasable and does not change Snapshot, Execution Plan, Driver, Artifact or Publication
  authorities. Phase 9C now defines the provider execution foundation; Phase 9D remains pending for concrete non-Docker
  provider execution.

## 2026-08-06 phase-9c-provider-execution-foundation

- The non-Docker gap is explicitly split: Phase 9C owns provider-neutral connection, credential, Snapshot attestation,
  cancellation/recovery and conformance authority; Phase 9D owns concrete Kubernetes/BuildKit/Kaniko adapters.
- Capability declarations are not execution proof. Selectors and scheduling expose only providers with a registered,
  health-checked implementation and replayable conformance evidence. Incomplete providers remain fail-closed.

## 2026-08-06 phase-9c-provider-contract-implementation

- `TargetBoundProviderExecutionConformanceCapability` is now a stable Runtime Target-owned boundary. It receives only
  frozen target/Driver/platform/Snapshot identity and delivery mode, and returns provider identity, contract version and
  lifecycle evidence flags without endpoint, credential or path details.
- The Docker target provider implements and registers this conformance contract. v2 Build execution requires a complete
  conformance result before Snapshot delivery for both single-platform and coordinated platform legs.
- Unsupported Drivers, mismatched delivery modes and incomplete conformance evidence fail closed; focused Build and
  Runtime Target tests cover these gates. Concrete Kubernetes/BuildKit/Kaniko execution remains Phase 9D.
- Build now persists the non-secret conformance result in an append-only Build-owned evidence record keyed by Execution
  Plan and Task stage. Duplicate stage replay is idempotent and cannot overwrite a prior proof.
- Runtime Target additionally registers the aggregate `TargetBoundDockerBuildProvider` contract for the Docker reference
  adapter. It is deliberately Docker-specific; Phase 9D providers must not inherit Docker-specific types.
- The v2 executor now requires that aggregate Docker contract in addition to its typed sub-capabilities. A partially
  assembled Docker provider graph therefore fails before plan execution.
- Corrected manifest aggregation error propagation: failure to prepare the complete platform Artifact set now reaches
  Task Runtime instead of returning a false successful leg.
- Provider evidence replay now compares the stored non-secret proof fields. Identical replays commit successfully;
  conflicting provider identity, target, platform, version or proof flags return `ErrConflict` instead of being silently
  ignored by `ON CONFLICT DO NOTHING`.
- Evidence persistence now follows successful Snapshot delivery proof in both single-platform and coordinated legs;
  a failed transfer cannot leave a misleading delivery capability fact behind.

## 2026-08-06 phase-9d-provider-prerequisite-audit

- 当前环境和 `server/go.mod` 均没有 `kubectl`、`buildctl`、Kaniko 或 Kubernetes client authority；仓库现有
  Kubernetes 代码只负责 Runtime metadata/classification，不能证明 Build 执行。
- 因此 Phase 9D 暂不声称 concrete provider 已完成。后续 adapter 必须先补齐 Runtime Target 私有连接/凭据
  authority，并在真实 provider 工具或 client 可验证的环境中通过 Snapshot、执行、发布、取消和清理证据。
- Runtime Target 现在提供私有 `RuntimeTargetProviderConnectionReader`，统一校验目标存活、`image_build` 能力和
  无凭据嵌入的 endpoint；它不进入 Build target summary，且不会单独使 Kubernetes provider 可执行。

## 2026-08-06 phase-9c-provider-driver-contract

- `TargetBoundProviderDriverExecutionCapability` now defines the provider-neutral Driver boundary. Its request carries
  only frozen Snapshot identity, verified delivery proof, target, Driver and platform; its result carries only
  digest-addressed execution evidence.
- The contract is intentionally not registered by a Kubernetes, BuildKit, Kaniko or Buildah implementation. Docker's
  existing reference-provider interfaces remain unchanged, so no concrete-provider completion is implied.

## 2026-08-06 phase-8a-builder-telemetry-contract

- Runtime Target currently has no Build-facing authority for builder capacity, running/queued work, freshness,
  region or affinity. Runtime UI summaries and Monitor host metrics are intentionally excluded from scheduler input.
- `BuilderTelemetrySnapshot` and `RuntimeTargetBuilderTelemetryReader` now define the narrow provider-neutral boundary.
  Snapshots carry Runtime Target identity, capacity/load dimensions, observed/expiry times and source/region/affinity
  claims; Build binds the accepted fact to its Builder Instance in placement evidence. Stale, unavailable or
  inconsistent facts fail closed through `FreshAt`.
- The first contract draft incorrectly included a Builder Instance identifier in Runtime Target telemetry. It was
  removed before source implementation: Builder Instance remains Build-owned, while Runtime Target reports only its
  physical target facts. This preserves the Runtime -> Build dependency direction.
- Docker daemon availability was verified locally, but Docker Engine does not expose a Graft-scoped, restart-safe Build
  capacity or queue fact. An in-memory provider counter would omit external builds and reset on process restart, so it
  is not accepted as telemetry authority and cannot enable `least_load`.

## 2026-08-06 phase-8b-builder-capacity-telemetry-source-authority

- The telemetry audit confirmed that Runtime Target persistence, Runtime UI summaries, Container stats, Monitor trends
  and Task Runtime JSON cannot provide the target-scoped, restart-safe queue/capacity authority required by Phase 8A.
- Phase 8B is added as the explicit prerequisite: Runtime/Infrastructure owns a real provider telemetry source and
  Build owns atomic Builder Instance reservations. Task lifecycle outcomes reconcile those reservations, while frozen
  placement evidence retains the accepted source and reservation fact.
- This phase is not a blocker for `round_robin` or static `labels`; it is the release gate for `least_load`, `region`
  and `affinity`. No fake Docker counter or UI-metric compatibility path is permitted.

## 2026-08-06 phase-4-artifact-read-foundation

- `build_v2_artifacts` is now exposed through a Build-owned store/service read boundary as an immutable, digest-addressed
  resource. It deliberately excludes legacy Docker Job artifact rows and does not resolve mutable Publication references.
- `GET /api/build/artifacts` now exposes that read model through the canonical OpenAPI-derived server and web contracts.
  The legacy Build Job artifact schema is separately named `BuildJobArtifact`, so the v2 Artifact contract cannot alter
  historical Docker Job projections.
- The visible `Build > Artifacts` resource at `/build/artifacts` consumes only that immutable projection. It displays
  digest, media type, platforms, size and creation facts; it does not treat a mutable repository/tag Publication as
  Artifact identity. Promotion and supply-chain actions remain out of this read-only delivery slice.
- No Runtime/Infrastructure source implementation or scheduler consumer is claimed in this phase. `round_robin` and
  static `labels` remain executable; `least_load`, `region` and `affinity` remain disabled until the authority release
  gate is met.

## 2026-08-06 phase-4-promotion-settlement-foundation

- Runtime Target's Docker provider already executes a digest-preserving OCI copy from a Registry-authorized private
  binding. Build now owns the corresponding settlement boundary: it rechecks the provider digest and media type against
  the immutable source Artifact before recording the destination Publication.
- The former unique destination/reference constraint was removed. A deterministic Publication identity now includes the
  immutable digest and destination binding, so idempotent replay retains one row while a later digest at the same mutable
  reference preserves the historical publication relationship.
- This remains a settlement foundation only. Task Runtime promotion orchestration, retry/cancellation semantics and the
  public promotion write contract remain Phase 4 work; no second Build worker or direct API execution path was added.

## 2026-08-06 phase-4-promotion-task-runtime-stage

- Build now resolves a source only from its persisted Publication records, asks Registry to authorize pull and push, and
  freezes the resulting non-secret source, destination and Runtime Target identities in one Task Runtime stage.
- The stage resolves Registry private bindings only while executing, delegates the digest-preserving copy to the selected
  Runtime Target provider, and settles the Publication only after matching provider proof. It has no Build-local worker,
  queue or synchronous copy path.
- Cancellation is passed to the provider through the Task Runtime Stage executor. Recovery remains `manual_reconcile`, so
  a process interruption cannot silently repeat an externally uncertain copy. Build registers Task owner authorization for
  reading, cancellation and manual retry; the public OpenAPI/HTTP promotion write contract is still pending.

## 2026-08-07 phase-4-promotion-public-write-contract

- OpenAPI now owns the canonical `POST /api/build/artifact-promotions` write contract. The Build route requires
  `build.create` and a bounded `Idempotency-Key`, takes the actor from the authenticated request context, and returns
  only the existing Task receipt with HTTP `202`; malformed input is `400` and idempotency conflicts are `409`.
- The typed web client exposes the same operation without adding an Artifact page action. The current Artifact list has
  no authorized `publication_id` discovery contract, so no selector, tag inference, or modal was introduced. Registry
  endpoint, credential and private binding facts remain outside the HTTP request, Task input and response.

## 2026-08-07 credential-and-telemetry-authority-rfc

- Accepted the repository-wide `Build Domain v2 Credential And Telemetry Authority RFC` as the authority for Registry
  credential execution, capability matching, Builder Reservation, telemetry, Placement, Evidence, event projection and
  failure taxonomy. It complements rather than replaces the immutable Snapshot/Plan/Artifact model.
- Historical `docker-runtime-store` evidence remains readable, but all new publication must resolve a scoped ephemeral
  credential through `CredentialProvider` and inject it through a provider-owned `RuntimeExecutionAdapter`. Default
  Docker credential-store access and environment-default authentication are prohibited and fail closed.
- The existing `RuntimeTargetBuilderTelemetryReader` contract remains the only Build-visible facade, but no source
  implementation is claimed. Docker/host metrics, UI/Monitor projections and Task JSON are diagnostic only; they cannot
  enable `least_load`, `capacity`, `affinity` or `region` placement.
- Existing Pool and Placement work remains an implementation asset. Feature exposure is reset to four RFC gates:
  manual single Builder; complete intent/materialization; static Pool policies; then real telemetry, dynamic placement
  and Task Runtime-owned distributed Build. Builder/Registry-local failure does not enter global availability state.
- Documentation validation remains `git diff --check` plus the bounded `validate_ai_plan_structure.py` guard. Runtime,
  OpenAPI and web validation are intentionally deferred because this batch changes documentation only.

## 2026-08-07 provider-sdk-spi-rfc

- Added the standalone Provider SDK/SPI RFC. Provider lifecycle, capability negotiation, telemetry, credential,
  Workspace, execution and evidence adapters are now one compile-time integration surface.
- The SPI requires MUST conformance for lifecycle ordering, isolated credentials, Snapshot identity, cancellation,
  fencing, cleanup, freshness and redacted evidence. Unknown or failed conformance remains static/manual only and does
  not enter dynamic Placement.
- The SPI deliberately excludes a second runtime, scheduler, queue, event store, evidence database, Registry model and
  global health registry. Concrete Provider language bindings and registration locations remain implementation work.

## 2026-08-08 phase-4a-telemetry-authority

- Runtime Target now registers a private `RuntimeTargetBuilderTelemetryControlPlane` plus concrete
  `controlPlaneBuilderTelemetryProvider` behind the only Build-visible `RuntimeTargetBuilderTelemetryReader` facade.
  The control plane provisions target-bound Builder Agent public keys, verifies signed reports and derives integrity
  before appending the latest durable observation per Runtime Target to the
  `runtime_target_builder_telemetry_observations` ledger.
- Each observation carries Builder scope, running Builds, queue, allocatable slots, health, capability profile/version,
  redacted source/provenance/integrity, observation window and explicit unsupported dimensions. The provider has no
  Docker client, host metrics reader, Task JSON, UI or Monitor dependency, and its durable table excludes endpoint and
  credential fields.
- Admission is fail-closed for missing, stale, malformed, duplicate, unauthenticated or required-unsupported
  observations. The focused provider tests cover control-plane registration, signature rejection, durable readback,
  latest-observation selection plus missing, expired and unsupported required dimensions.
- This establishes only Phase 4a authority. It does not enable `least_load`, `capacity` or `affinity`, change OpenAPI
  or web, add a scheduler/queue/health registry, implement a Task Runtime, Reservation recovery or target reselection.

## 2026-08-08 phase-4a-controller-settlement

- Controller accepted `ba114b35` and the ingress repair `44cd1297` after verifying the Runtime Target control-plane
  boundary, durable observations, signature admission and Build-visible facade isolation.
- Phase 4a remains a telemetry authority gate only. The loop advances to Phase 4b Task Runtime distributed coordination;
  Phase 4c Reservation recovery/dynamic placement and Phase 4d OpenAPI/web projection remain pending in order.

## 2026-08-08 phase-4b-task-runtime-distributed-coordination

- Task Runtime now expands each frozen coordinated template into its parallel platform legs plus one final serial
  aggregate stage. The normal persisted stage state machine therefore owns cancellation, retry, restart recovery and
  the single aggregate terminal transition; Build cannot complete the Task from a leg.
- Build leg execution records only its immutable platform Artifact. The final aggregate stage is the sole caller of
  provider manifest publication and Build-owned Artifact/Publication settlement, after all leg Artifacts exist.
- This establishes the Phase 4b coordination boundary only. It does not add a scheduler, queue, second Task Runtime,
  event store, telemetry source, dynamic policy, OpenAPI or web projection. Reservation recovery still blocks dynamic
  placement acceptance.

## 2026-08-08 phase-4b-controller-settlement

- Controller accepted `9c1fe6d8` after verifying that Task Runtime owns one aggregate stage and Build legs cannot
  invoke manifest publication or final settlement.
- The loop advances to Phase 4c Reservation recovery and dynamic placement. Phase 4d contract/web projection remains
  pending until a dynamic policy is executable.

## 2026-08-08 phase-4c-reservation-recovery-and-dynamic-placement

- Build enables only `least_load`, `capacity` and `affinity` after Runtime Target telemetry admission. Selection uses
  the narrow reader, rejects missing, stale or non-conformant observations, and freezes policy/version, candidate
  fingerprint, capability profile/version, source, provenance and integrity into placement evidence.
- Retry derives its Builder identity only from the persisted Placement. Before a retry creates a new attempt-scoped
  Reservation fence, the V2 executor re-confirms that same Runtime Target through the telemetry reader; it fails
  closed instead of querying a Pool or selecting another target. `region` remains disabled.
- Controller accepted `95a25b57` and follow-up repair `cfec66f2` after focused Build/Task/Runtime Target tests and
  `graft validate backend`. Phase 4d OpenAPI and web projection is now the sole pending batch.

## 2026-08-08 phase-4d-contract-and-web-projection (historical, superseded)

- The canonical Builder Pool OpenAPI enum now exposes exactly `least_load`, `capacity` and `affinity` in addition to
  the static policies. Generated web schema and localized Build selector labels follow that contract; no browser
  telemetry, policy evaluation, target selection or `region` path was introduced.
- Controller accepted `6bc99b07` after OpenAPI freshness and the Build selector page test. This historical acceptance
  claim was superseded by the later Phase 4 recovery entries; the topic remains active-incomplete.

## 2026-08-08 Phase 4 Closeout Recovery

- The prior Phase 4 closeout is superseded: it treated a generic signed ingress, production Placement matching and a
  per-Instance exclusive Reservation as sufficient dynamic evidence. They are not.
- Phase 4 is active and incomplete until Task Runtime maps `credential_cleanup_unverified` to `Internal` / `Needs
  Attention`; every Placement freezes a real `CapabilityMatcher` negotiation; Reservation claims explicit capacity units;
  and a provisioned Docker Builder Agent proves target-bound telemetry from its controlled execution ledger or Driver
  controller.
- Dynamic policy is disabled pending Provider admission. Historical dynamic Pool rows remain readable but cannot execute;
  BuildKit, Kaniko and Kubernetes are future extensions until each passes equivalent conformance. Dynamic retries keep
  the frozen target and must never reselect from a Pool. Local Builder/Registry failure remains outside global platform
  availability.

## 2026-08-08 Phase 4 Docker Driver-Controller Ledger Integration

- Runtime Target now updates the unique enabled Docker Agent's durable execution ledger immediately around the actual
  Docker CLI build boundary. The provider owns only queue/running completion facts; it does not select targets, retry,
  settle Task state or create a second runtime.
- The target-only frozen Placement contract is made explicit with one enabled Agent scope per Runtime Target. A focused
  provider conformance test observes `running=1` during the Docker command and `running=0` after completion, without
  Docker stats, host metrics, Task JSON or UI input.
- This is not an out-of-process Agent deployment protocol. Transport, private-key bootstrap and operator lifecycle lack
  a canonical authority and remain a release blocker; Phase 4 stays active-incomplete.

## 2026-08-08 Phase 4 Capability Contract And Key-Rotation Alignment

- The current bounded OCI path freezes `TemplateRef=oci-dockerfile/default@v1`, `DestinationKind=oci_registry`,
  `CachePolicy=disabled` and `SecurityPolicy=default`; feature modes are `registry-login` (required), `provenance`
  (preferred) and `sbom` (optional). These are interim Build-owned defaults until a broader typed caller intent has
  matching OpenAPI, provider conformance and execution support.
- Runtime Target telemetry key rotation requires an explicit `enabled=false` transition before a public-key change;
  same-key upsert preserves the accepted sequence, while a changed key resets it only after disable.

## 2026-08-08 Phase 4 Resolved Policy Freezing

- The bounded OCI `disabled/default` cache and security policies are now immutable Execution Plan facts. They are
  included in the plan digest, persisted with the plan, restored for execution, and used to reconstruct dynamic retry
  capability requirements; policy drift fails closed against frozen placement evidence.
- The current route intentionally exposes no caller-selectable policy modes because no additional cache or security
  capability has provider conformance. Extending the policy vocabulary requires the typed Build-owned request and
  OpenAPI contract, persistence, matcher semantics and provider proof in the same slice.

## 2026-08-08 Phase 4 Admission Audit Repair

- Docker execution now freezes the Agent identity selected at build start and settles that exact durable ledger row;
  disable or rotation cannot redirect completion accounting to a newly active Agent.
- Dynamic telemetry rejects unknown `unsupported_dimensions`; only the explicitly optional `cache_state` dimension is
  accepted for the current bounded OCI contract.
- Focused server tests, backend validation, OpenAPI, web checks, migration guards and diff checks pass. Deployable
  enrollment/bootstrap, mutually authenticated transport, service packaging, restart reconciliation and operator audit
  remain the separate Docker Agent admission release blocker.

## 2026-08-08 runtime-target-agent-trust-model ADR

- Accepted ADR-023 as the Runtime Target Agent identity and lifecycle authority before PR1 contracts. Vault PKI is the
  reference managed backend; Agent identity is an exact target-bound URI SAN; vault-managed deployment delivery
  materializes enrollment and private-key material directly for the Agent; reports use TLS 1.3 mTLS plus one-time
  controller snapshots; the canonical package is a signed OCI image; revocation is checked per report, propagated over
  the control channel, and fail-closed with a 60-second maximum cache and 24-hour certificate lifetime.
- Updated the credential/telemetry authority RFC to point to ADR-023 for the concrete protocol. Registry file-backed
  credentials remain a separate `CredentialProvider` concern.
- Added the Credential Vault And Runtime Target Agent Protocol RFC. It fixes the PR1 experimental wire/module contract
  scope and rejects server push, streaming, legacy dual admission and Runtime Target secret custody.

## 2026-08-09 runtime-target-agent-delivery-grant-binding ADR

- Accepted ADR-024 to close the bootstrap-delivery authority gap in ADR-023. Runtime Target owns provider-neutral
  Agent enrollment, identity and lifecycle, including the one-to-one delivery grant, peppered token verifier and final
  membership activation. Docker is the sole MVP delivery implementation; credential-vault remains the provider-neutral
  CSR-based Vault PKI issuer only.
- Automation reuses the existing deployment trust boundary for handoff and receipt submission. Runtime Target consumes
  only the already verified automation identity; it does not add an automation PKI, trust bundle, listener or signing
  scheme. The boundary releases the grant and plaintext token once, creates the bound Docker secret, and returns
  redacted delivery evidence without exposing the token in an Operator API or persistent Graft record. That receipt is
  evidence only and never activates or binds an Agent. The Agent supplies token plus CSR only to Graft's
  server-authenticated bootstrap listener.
- The RFC and roadmap now defer all Operator and Agent OpenAPI publication until delivery receipt verification,
  bootstrap authorization, issuance, durable activation, mTLS reconnect, revocation and Docker ledger conformance pass
  as one slice. Docker is the only MVP runtime and delivery mechanism; no unsupported-provider implementation, menu,
  API or placeholder is authorized.

## 2026-08-08 startup authority alignment

- Added ADR-023 and the Credential Vault And Runtime Target Agent Protocol RFC to the reusable startup prompt and the
  topic tracking repository-truth list. Recovery sessions implementing the Docker Builder Agent protocol must read
  both documents before changing trust, enrollment, credential delivery, telemetry, or revocation behavior.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "authority-bootstrap",
    "phase-1-single-builder",
    "phase-1-registry-credential-execution-historical",
    "phase-1.75-snapshot-materialization-retention",
    "phase-2-workspaces-templates-drivers",
    "phase-2-intent-materialization-conformance",
    "phase-3-pool-round-robin-foundation",
    "phase-3-static-pool-placement",
    "phase-9b-remote-docker-provider",
    "phase-10-build-selector-read-model",
    "phase-9c-provider-conformance-evidence",
    "phase-9c-provider-driver-contract",
    "phase-8a-builder-telemetry-contract",
    "credential-and-telemetry-authority-rfc",
    "provider-sdk-spi-rfc",
    "phase-1-secure-credential-execution-and-manual-reservation",
    "phase-4-cleanup-failure-taxonomy",
    "phase-4-mandatory-capability-matching",
    "phase-4-slot-aware-reservation",
    "phase-4-capability-intent-and-frozen-negotiation",
    "phase-4-resolved-policy-freezing"
  ],
  "pending_batches": [
    "phase-4-docker-builder-agent-admission",
    "phase-4-provider-admission-and-dynamic-retry"
  ],
  "current_batch": "phase-4-docker-builder-agent-admission",
  "next_batch": "phase-4-provider-admission-and-dynamic-retry",
  "closeout_status": "active-incomplete"
}
```
