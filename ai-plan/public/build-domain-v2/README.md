# Build Domain v2 Topic

## Objective

Evolve the Docker-first Build Center into a Build Domain with immutable source and artifact authority, Task Runtime-owned
execution, secure Registry credential execution and evidence-backed Builder placement. The current design authority is
[Build Domain v2 Credential And Telemetry Authority RFC](../../design/architecture/build-domain-v2-credential-and-telemetry-authority.md).
Provider integration follows [Build Domain v2 Provider SDK And SPI RFC](../../design/architecture/build-domain-v2-provider-sdk-spi.md).
Deployable Agent trust and wire semantics follow
[ADR-023](../../design/decisions/ADR-023-runtime-target-agent-trust-model.md),
[ADR-024](../../design/decisions/ADR-024-runtime-target-agent-delivery-grant-binding.md), and the
[Credential Vault And Runtime Target Agent Protocol RFC](../../design/architecture/credential-vault-and-runtime-target-agent-protocol.md).

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `docs/automation` for this RFC alignment; subsequent implementation may be `cross-boundary`
- recovery source: `parent topic`
- authority summary: Task Runtime owns execution lifecycle; Runtime Target owns connections and provider entry; Registry
  Connection owns endpoint/repository policy and credential reference; Credential Provider owns secret issuance; Build
  owns Builder resources, capability matching, placement evidence and reservations.

## Current Decision

- Phase 4 is `active-incomplete`; the previous `archive-ready` closeout is superseded by the documented admission and
  capacity gaps below.
- New Build execution must not use `docker-runtime-store`, a default Docker credential store or environment-default
  Registry authentication. Historical records remain readable only.
- Builder/Registry-local failure is a local Build capability failure. It cannot alter `PlatformAvailabilityStore` or
  `CapabilityCoordinator` global availability decisions.
- `RuntimeTargetBuilderTelemetryReader` remains the Build-visible read facade. A historical generic signed ingress is
  not a real Docker Builder Agent source and cannot admit a Provider. UI summaries, Monitor charts, Docker/host metrics
  and Task JSON cannot enable dynamic placement.
- Runtime Target now couples the real Docker CLI execution boundary to a durable Driver-controller ledger and permits
  one enabled Agent scope per target. This is source evidence, not an out-of-process Agent deployment protocol; no
  transport, private-key bootstrap or operator lifecycle authority exists yet.
- Existing Pool/Placement material is retained, but public capability exposure is reset to the RFC's four phases.
  `least_load`, `capacity`, `affinity` and `region` are latent/disabled until Phase 4 evidence exists.
- Provider lifecycle and adapter conformance are defined by the separate SDK/SPI RFC; no concrete new provider is
  claimed by this documentation batch.

## Owned Scope

- `ai-plan/design/architecture/build-domain-v2-credential-and-telemetry-authority.md`
- `ai-plan/design/architecture/build-domain-v2.md`
- `ai-plan/design/architecture/build-domain-v2-provider-sdk-spi.md`
- `ai-plan/roadmap/build-domain-v2.md`
- this active topic's tracking and trace materials

Implementation scope, when activated, spans Build, Runtime Target, Infrastructure Registry, Secret/Credential Provider,
Task Runtime, OpenAPI and Build web contracts. It must repair the highest authority first and must not create a second
Task Runtime, Scheduler, Registry resource model, event store, evidence database or global health registry.

## Pending Direction

1. Phase 4 is active and incomplete. `credential_cleanup_unverified` now resolves to `Internal` / `Needs Attention`
   through the Task Runtime executor-outcome boundary, with no automatic retry or reservation/credential reuse.
2. Complete `CapabilityMatcher` and frozen negotiation evidence for every manual, static, dynamic and distributed-leg
   Placement.
3. Replace the per-Instance live-lease rule with slot-aware Builder Reservation, then prove a provisioned Docker Builder
   Agent telemetry source and Provider admission before enabling any dynamic policy.
4. Keep dynamic rows readable but non-executable. BuildKit, Kaniko and Kubernetes remain future extensions until their
   own conformance gates pass.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/build-domain-v2/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
