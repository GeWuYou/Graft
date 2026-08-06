# Build Domain v2

## Current Status Summary

- Topic objective: replace the Docker-first Build Center authority with an immutable, Runtime Target-capability based
  Build Domain that can evolve across the approved staged phases.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running cross-boundary feature requiring repository design, roadmap, active topic recovery and
  loop-driven phased delivery.
- Canonical authority:
  - `ai-plan/design/architecture/build-domain-v2.md`
  - `ai-plan/roadmap/build-domain-v2.md`
- Completed so far: Work Intake, v2 authority/bootstrap, immutable Snapshot/Execution Plan persistence, Runtime Target
  build capability, Registry Connection/Repository authority, and the first single-builder server/web contract slice.
- Completed: Phase 1.5 Registry credential execution authority for target-bound OCI publication.
- Completed: Phase 1.75 Snapshot materialization ownership, provenance uniqueness and retention-state authority.
- Completed: Phase 2 Build-owned Workspaces/Snapshots, versioned Templates/Drivers and Application source materialization.
- Completed foundation: Phase 3 Builder Pools, transactional Round Robin selection and Pool-bound plan freezing.
- Completed foundation: Phase 6 coordinated Task Runtime legs, cancellation/recovery and per-platform Artifact facts.
- Completed execution slice: Phase 7 Docker Buildx Manifest publication and Build-owned finalization path.
- Current: Phase 8 placement foundation freezes one Builder Placement per platform; its remaining work is truthful
  scheduling policy beyond round robin. Static `labels` selection and selector evidence are now supported; load,
  affinity and region policies remain fail-closed until their telemetry is authoritative. Phase 9B Docker remote
  execution is complete; the provider execution foundation is a separately gated Phase 9C, while concrete
  Kubernetes/BuildKit/Kaniko adapters remain Phase 9D.
- Phase 8A contract delivered: `BuilderTelemetrySnapshot` and `RuntimeTargetBuilderTelemetryReader` define the
  freshness/provenance boundary for future capacity, queue, region and affinity scheduling. No Runtime/Infrastructure
  source implementation exists yet, so the unsupported policies remain fail-closed.
- Phase 8B is the explicit prerequisite for implementing that contract: a real Runtime/Infrastructure telemetry
  source plus Build-owned capacity reservations and Task lifecycle reconciliation. It does not block `round_robin` or
  static `labels`.
- Phase 9A completed: provider-owned Snapshot delivery contract and Local Docker identity proof.
- Phase 9B completed for Docker: Runtime Target owns private target-scoped connection validation and registers the
  target-bound Docker provider. Unix-socket targets use `target-local`; validated TCP/SSH targets use
  `provider-transfer` and execute build/publication/manifest commands against the selected target. Kubernetes and other
  providers remain fail-closed for future Phase 9D work; Phase 9C first establishes the provider execution foundation.
- Phase 9C foundation delivered: Runtime Target owns the aggregate Docker reference adapter and private generic connection
  reader; Build verifies and persists non-secret conformance evidence after Snapshot delivery. Conflicting evidence
  replays fail with `ErrConflict`; no concrete non-Docker provider is claimed.
- Phase 10 completed: Build-owned selector APIs expose authorized Workspaces, Build-capable Runtime Targets and Builder
  Pools; the create page uses controlled selectors with loading, empty and error states.
- Phase 4 Artifact read delivery completed: Build exposes immutable Artifact facts through the canonical API and the
  Build > Artifacts browser; legacy Job projections remain separately readable.
- Phase 4 Promotion Task Runtime stage completed: Build freezes a Publication-selected digest source and Registry-authorized
  destination, then delegates copy, cancellation, manual recovery and settlement to the existing Task Runtime. The public
  OpenAPI/HTTP promotion write contract is now complete; authorized Publication discovery and the Artifact page workflow
  remain a later contract.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary: Runtime Target owns build capability, Build owns immutable source/plan/artifact authority, and
  Task Runtime owns execution state.

## Owned Scope

- `ai-plan/design/architecture/build-domain-v2.md`
- `ai-plan/roadmap/build-domain-v2.md`
- `server/modules/build/**`, Runtime Target, Infrastructure registry owners, Task, OpenAPI and Build web contracts.

Out of scope:

- Implementing a Graft Registry server or exposing Runtime endpoint/credential details.
- Creating a second Build execution, queue, log or realtime runtime.

## Locked Decisions

1. Build consumes immutable Workspace Snapshots and Execution Plans; retry never refreshes source.
2. Artifact is first-class and immutable; Publication binds it to mutable tags, manifests, promotions or copies.
3. Runtime Target reports physical build capability; Builder Profile, Instance and Pool are Build-owned logical resources.
4. OCI Registry is the Phase 1 deployment-grade Destination, but Artifact Destination remains provider-neutral.

## Phase Plan

- `authority-bootstrap`
- `phase-1-single-builder`
- `phase-1-registry-credential-execution`
- `phase-1.75-snapshot-materialization-retention`
- `phase-2-workspaces-templates-drivers`
- `phase-3-pools-scheduling-platforms`
- `phase-4-artifact-supply-chain-automation`
- `phase-5-task-runtime-distributed-legs`
- `phase-6-distributed-leg-coordinator`
- `phase-7-driver-owned-manifest-publication`
- `phase-8-platform-aware-builder-placement`
- `phase-8b-builder-capacity-telemetry-source-authority`
- `phase-9-snapshot-delivery-remote-execution`
- `phase-9c-provider-execution-foundation`
- `phase-9d-concrete-provider-adapters`

## Current Recovery Point

- `authority-bootstrap` was accepted and committed as `084ae531`.
- Legacy Docker Build Center history is preserved, but its new-write model is superseded.
- Phase 1 through Phase 7 foundations are implemented, including immutable plan/artifact facts and coordinated
  Buildx manifest settlement. Phase 8 has immutable per-platform placement and deterministic labels selection;
  round-robin is the only other enabled policy. Docker provider execution covers local Unix-socket and validated
  remote TCP/SSH targets; non-Docker providers remain fail-closed.

## Work Intake

- This topic was created through `Work Intake`.
- Persisted Work Contract: `ai-plan/public/build-domain-v2/todos/build-domain-v2-tracking.md`.

## Pending Batch Direction

- Phase 10 closes the selector/read-model gap independently. Phase 8 continues with evidence-backed placement policies.
  Phase 9C now closes the provider execution foundation gap; Phase 9D then adds non-Docker provider-backed Snapshot
  delivery and remote execution. The user has pre-authorized normal in-scope `execute_repair` actions for this topic; preserve
  authority and validation checks without repeating authorization requests.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/build-domain-v2/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
