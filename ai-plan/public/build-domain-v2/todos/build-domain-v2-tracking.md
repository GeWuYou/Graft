# Build Domain v2 Tracking

## Topic

Build Domain v2

## Scope

Replace the Docker-first Build Center with a platform Build Domain covering immutable source snapshots, execution plans,
Runtime Target build capability, artifact delivery, and the approved staged evolution.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/AGENTS.md`
- `ai-plan/design/architecture/build-domain-v2.md`
- `ai-plan/roadmap/build-domain-v2.md`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Runtime Target owns build capability; Build owns immutable Snapshots, Execution Plans, Artifacts and Publications; Task Runtime owns execution state.
requires:
  design: true
  topic: true
  roadmap: true
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - ai-plan/design/architecture/build-domain-v2.md
    - ai-plan/roadmap/build-domain-v2.md
    - ai-plan/public/build-domain-v2/README.md
    - ai-plan/public/build-domain-v2/startup-prompt.md
    - ai-plan/public/build-domain-v2/todos/build-domain-v2-tracking.md
    - ai-plan/public/build-domain-v2/traces/build-domain-v2-trace.md
    - ai-plan/public/README.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- `authority-bootstrap` was accepted and committed as `084ae531` after structure and diff validation.
- Current batch: Phase 8 placement foundation is complete. Remaining Phase 8 scope is scheduler-policy truth beyond
  round robin. Phase 9 is a separately releasable provider-backed Snapshot delivery and remote execution gap closure.
- Recovery authority: Phase 1 completed the v2 vertical path. Phase 1.5 completed the Registry-to-Runtime credential
  execution boundary. Phase 1 permits only the system-managed Local Docker Runtime Target, because Container has no
  remote Docker execution adapter. Preserve the existing route-prefix correction without reverting or overwriting it.
- Repair eligibility: the user authorized all approved phases and normal in-scope `execute_repair` actions in the
  current workspace. Continue repairs and their validation without requesting repeated authorization; widened scope
  must still be required authority repair, not a compatibility path.

## Task Checklist

- [x] Settle authority/bootstrap design, roadmap and topic recovery materials.
- [x] Phase 1: Runtime build capability, Application Snapshot adapter, single Builder and OCI Registry publication.
- [x] Phase 1.5: Registry credential execution binding for the selected Runtime adapter.
- [x] Phase 1.75: Snapshot materialization ownership, retention state and provenance uniqueness foundation.
- [x] Phase 1.75 foundation: Build-owned expired materialization cleanup lease and private-path enforcement.
- [x] Phase 2: Build-owned Workspaces/Snapshots, versioned Templates/Drivers and Application source materialization.
- [ ] Phase 3: Builder Profiles/Instances/Pools, scheduling and multi-platform fan-out.
- [x] Phase 3 foundation: Builder Pool membership, transactional Round Robin selection and Pool-bound plan freezing.
- [ ] Phase 4: promotion, OCI supply-chain evidence, remote/distributed builders and deployment/pipeline handoff.
- [x] Phase 4 foundation: Build-owned v2 Artifact read model exposes digest-addressed Artifact facts independently of
  legacy Docker Build Job projections, including the canonical `GET /api/build/artifacts` OpenAPI and HTTP contract.
- [x] Phase 4 Artifact read delivery: Build > Artifacts is a canonical `/build/artifacts` navigable read model with
  immutable digest, media type, platforms, size and creation facts; Promotion, supply-chain evidence and mutable
  Publication workflows remain later Phase 4 scope.
- [x] Phase 4 promotion Task Runtime stage: Build freezes a Publication-selected immutable digest source and a
  Registry-authorized destination, while the existing Task Runtime owns copy execution, cancellation and manual recovery.
  The public OpenAPI/HTTP promotion write contract remains pending.
- [x] Phase 5: Task Runtime distributed-leg coordination contract and manifest aggregation authority.
- [ ] Phase 6: Persisted distributed-leg coordinator, shared cancellation, recovery and Build manifest publication.
- [x] Phase 6 foundation: coordinated Stage group/leg persistence, parallel-safe claim eligibility and multi-stage runtime tracking.
- [x] Phase 6 foundation: multi-instance untracked coordinated-leg cancellation and restart cancellation recovery.
- [x] Phase 6 foundation: Build-owned per-platform immutable Artifact persistence with non-overwriting leg settlement.
- [ ] Phase 7: Driver-owned OCI Manifest publication capability and multi-platform API release gate.
- [x] Phase 7 foundation: complete-platform Artifact validation and Driver publication input contract.
- [x] Phase 7 foundation: final OCI Manifest Artifact and Publication settlement after Driver-proven digest result.
- [x] Phase 7 foundation: Container Buildx provider adapter with target-declared `docker-buildx` capability gate.
- [x] Phase 7 execution slice: coordinated Build legs now build and publish per-platform immutable digests, then
  attempt provider-owned OCI Manifest publication and Build-owned settlement.
- [ ] Phase 8: Platform-aware Builder Pool scheduling across multiple Runtime Targets (additional gap).
- [x] Phase 8 foundation: immutable per-platform Builder Placement is included in the Execution Plan digest and used
  by coordinated Task legs and the Build executor; targets must declare `build-snapshot` locality.
- [x] Phase 8 foundation: deterministic `labels` Pool selection freezes validated selector evidence in each Placement.
- [ ] Phase 8A: Runtime/Infrastructure Builder Telemetry Authority for fresh capacity, queue, region and affinity evidence.
- [x] Phase 8A contract: `BuilderTelemetrySnapshot` and `RuntimeTargetBuilderTelemetryReader` define the provider-neutral
  freshness/provenance boundary; no source implementation is claimed yet.
- [ ] Phase 8B: Builder Capacity And Telemetry Source Authority. Runtime/Infrastructure must provide a real,
  restart-safe target telemetry source; Build must add atomic Instance reservations and Task lifecycle reconciliation
  before dynamic policies can be enabled.
- [x] Phase 9: Provider-backed Snapshot delivery and Remote Builder execution (Docker provider slice).
- [x] Phase 9 foundation: Runtime Target exposes Snapshot delivery modes and providers require an explicit delivery
  capability for the selected target.
- [x] Phase 9 foundation: provider-owned Snapshot delivery contract is invoked before every v2 build leg; Local Docker
  verifies the Build-managed materialization root and returns an identity-matching delivery proof.
- [x] Phase 9A: Delivery Contract Foundation is complete; identity mismatch and unsupported delivery modes fail closed.
- [x] Phase 9B: Remote Docker provider adapter enablement. Runtime Target owns private connection validation; the
  provider transfers the frozen Snapshot context through Docker CLI and executes target-bound build/publication/
  manifest operations. No local fallback is allowed.
- [x] Phase 9B foundation: Runtime Target owns a private Docker connection lookup; endpoint remains outside public
  Build/runtime summaries and is consumed only by the target-bound provider.
- [x] Phase 9B foundation: Runtime Target explicitly registers target-bound build/publication/manifest/delivery
  capabilities; Container remains a legacy local capability owner for standalone use.
- [x] Phase 9B foundation: private Docker connection lookup rechecks availability, `image_build` capability and
  endpoint/connection-kind consistency before provider use.
- [x] Phase 9C foundation: provider-neutral execution conformance contract is registered below Runtime Target/provider
  boundaries; Docker is the reference implementation and v2 execution fail-closes before Snapshot delivery when
  provider evidence is incomplete. No concrete non-Docker provider is claimed.
- [x] Phase 9C foundation: Build persists immutable, non-secret provider execution evidence per Execution Plan and Task
  stage; retries cannot overwrite a prior stage's proof.
- [x] Phase 9C foundation: `TargetBoundProviderDriverExecutionCapability` isolates provider-neutral Driver input
  (frozen Snapshot identity and delivery proof) from digest-addressed execution evidence. No non-Docker provider is
  registered by this contract alone.
- [ ] Phase 9C remaining: add provider-specific connection/credential authorities and integration conformance fixtures
  for each future non-Docker implementation.
- [ ] Phase 9D: Kubernetes/BuildKit/Kaniko provider adapters and target-specific execution credentials, gated by Phase 9C.
- [x] Phase 10: Build-owned selector read model and platformized create flow. Workspace, Build-capable Runtime Target and
  authorized Builder Pool selectors are now loaded through Build-owned APIs; opaque ID text inputs are removed.

## Acceptance Conditions

- Every submitted build has exactly one immutable Workspace Snapshot and Execution Plan.
- Retries retain their original source and plan; rebuild-current creates a new plan.
- Artifacts are digest-addressed, reusable outside Build Job and published through provider-neutral destinations.
- Runtime endpoint and credential details, arbitrary host paths and a second Build execution runtime never enter v2.
- Provider connection facts are read only through Runtime Target's private provider reader; Build selectors, plans and
  Task metadata receive no endpoint or credential material.
- Phase 9C must establish executable provider registration and conformance evidence before any non-Docker target is
  selectable; Phase 9D is the first phase allowed to claim a concrete Kubernetes/BuildKit/Kaniko adapter.
- Phase 3 partial release evidence: Pool selection is now a public builder selector and freezes the selected Pool and
  Instance; multi-platform fan-out remains gated by Phase 6 coordination plus Phase 7 Driver manifest publication.
- Phase 7 partial release evidence: multi-platform submission is materialized as Task Runtime coordinated legs and the
  executor performs per-leg publication plus manifest finalization when all immutable platform Artifacts are present.
  Placement is now frozen per platform; scheduler policy truth is Phase 8 and provider-backed cross-target delivery is
  Phase 9. Neither phase permits a local execution fallback for another Runtime Target.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "authority-bootstrap",
    "phase-1-single-builder",
    "phase-1-registry-credential-execution",
    "phase-1.75-snapshot-materialization-retention",
    "phase-2-workspaces-templates-drivers",
    "phase-3-pool-round-robin-foundation",
    "phase-5-task-coordination-contract",
    "phase-6-coordinated-leg-foundation",
    "phase-7-manifest-publication-foundation",
    "phase-8-placement-foundation",
    "phase-8-label-scheduling",
    "phase-9b-remote-docker-provider",
    "phase-10-build-selector-read-model",
    "phase-9c-provider-conformance-evidence",
    "phase-9c-provider-driver-contract",
    "phase-8a-builder-telemetry-contract"
  ],
  "pending_batches": [
    "phase-3-pools-scheduling-platforms",
    "phase-4-artifact-supply-chain-automation",
    "phase-8-platform-aware-builder-placement",
    "phase-8a-builder-telemetry-authority",
    "phase-8b-builder-capacity-telemetry-source-authority",
    "phase-9c-provider-execution-foundation",
    "phase-9d-provider-adapters"
  ],
  "current_batch": "phase-8a-builder-telemetry-authority",
  "next_batch": "phase-9c-provider-connection-authority-or-phase-9d-provider-adapter",
  "closeout_status": "recovery-required"
}
```
