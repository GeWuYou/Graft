# Build Domain v2 Topic

## Objective

Evolve the Docker-first Build Center into a Build Domain with immutable source and artifact authority, Task Runtime-owned
execution, secure Registry credential execution and evidence-backed Builder placement. The current design authority is
[Build Domain v2 Credential And Telemetry Authority RFC](../../design/architecture/build-domain-v2-credential-and-telemetry-authority.md).

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `docs/automation` for this RFC alignment; subsequent implementation may be `cross-boundary`
- recovery source: `parent topic`
- authority summary: Task Runtime owns execution lifecycle; Runtime Target owns connections and provider entry; Registry
  Connection owns endpoint/repository policy and credential reference; Credential Provider owns secret issuance; Build
  owns Builder resources, capability matching, placement evidence and reservations.

## Current Decision

- New Build execution must not use `docker-runtime-store`, a default Docker credential store or environment-default
  Registry authentication. Historical records remain readable only.
- Builder/Registry-local failure is a local Build capability failure. It cannot alter `PlatformAvailabilityStore` or
  `CapabilityCoordinator` global availability decisions.
- `RuntimeTargetBuilderTelemetryReader` remains the Build-visible read facade, but it does not yet have a real Provider
  source. UI summaries, Monitor charts, Docker/host metrics and Task JSON cannot enable dynamic placement.
- Existing Pool/Placement material is retained, but public capability exposure is reset to the RFC's four phases.
  `least_load`, `affinity` and `region` are latent/disabled until Phase 4 evidence exists.

## Owned Scope

- `ai-plan/design/architecture/build-domain-v2-credential-and-telemetry-authority.md`
- `ai-plan/design/architecture/build-domain-v2.md`
- `ai-plan/roadmap/build-domain-v2.md`
- this active topic's tracking and trace materials

Implementation scope, when activated, spans Build, Runtime Target, Infrastructure Registry, Secret/Credential Provider,
Task Runtime, OpenAPI and Build web contracts. It must repair the highest authority first and must not create a second
Task Runtime, Scheduler, Registry resource model, event store, evidence database or global health registry.

## Pending Direction

1. Phase 1 implementation: replace new Registry Push execution with Credential Provider plus Runtime Execution Adapter;
   introduce matcher, manual single-Builder evidence and fenced Reservation lifecycle.
2. Phase 2: complete Driver, Template, Instance and immutable Workspace materialization conformance.
3. Phase 3: expose static Pool policies only.
4. Phase 4: implement real Provider telemetry, dynamic policies and Task Runtime-owned distributed execution.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/build-domain-v2/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
