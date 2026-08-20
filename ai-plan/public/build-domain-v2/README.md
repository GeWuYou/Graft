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

- Docker-only Phase 4 dynamic admission is complete. A target remains fail-closed unless its active Agent generation
  supplies fresh controlled-ledger evidence. The Phase 7 multi-platform API release gate is complete: Buildx is the
  only public multi-platform Driver, Build freezes per-platform Pool placements, and Task Runtime owns final manifest
  aggregation.
- New Build execution must not use `docker-runtime-store`, a default Docker credential store or environment-default
  Registry authentication. Historical records remain readable only.
- Builder/Registry-local failure is a local Build capability failure. It cannot alter `PlatformAvailabilityStore` or
  `CapabilityCoordinator` global availability decisions.
- `RuntimeTargetBuilderTelemetryReader` remains the Build-visible read facade. A historical generic signed ingress is
  not a real Docker Builder Agent source and cannot admit a Provider. UI summaries, Monitor charts, Docker/host metrics
  and Task JSON cannot enable dynamic placement.
- ADR-026 promotes the provisioned Docker Agent into the single out-of-process `docker-runtime-agent`. Runtime Target
  retains mTLS identity and capability binding, Task Runtime owns external execution leases, and Build retains Plan,
  Placement, Reservation, Artifact and Publication authority. The current CLI boundary is now a migration bridge with
  SDK conformance and deletion as its cleanup trigger.
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

1. Preserve Docker-only Provider admission: every dynamic Placement must retain active-generation and fresh
   controlled-ledger evidence, and all incomplete evidence must fail closed.
2. Keep BuildKit, Kaniko and Kubernetes as future extensions until their independent conformance gates pass.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/build-domain-v2/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
