# Build Domain v2 Roadmap

## Delivery Principle

Each phase is independently releasable and preserves the v2 authority chain: immutable Workspace Snapshot and
Execution Plan, Runtime Target-owned build capability, Task Runtime-owned execution, first-class Artifact, and
provider-neutral Artifact Destination.

## Phase 1: Authority And Single Builder

- Publish Build Domain v2 authority and retire Docker-first new-write semantics.
- Add Runtime Target `build` capability discovery and a controlled Application Workspace Snapshot adapter.
- Add immutable Execution Plan submission, one Docker-compatible Driver/Profile/Instance, and one eligible Runtime
  Target build capability.
- Add Registry Connection, Artifact Repository, OCI Registry Destination, Artifact evidence and Publication records.
- Deliver the platform create flow and direct v2 API write-contract replacement. Legacy completed jobs remain readable.

**Release gate:** a user can freeze an authorized Application Workspace Snapshot, submit a plan to one compatible Builder
Instance, publish an immutable Artifact to an authorized OCI Repository, and retry from the original frozen Plan.

## Phase 2: Workspace And Build Intent

- Add reusable Workspaces and Snapshot materialization for Git, uploaded archive, generated Workspace and approved
  target-local sources.
- Add managed Template versions and Driver compatibility contracts, beginning with additional OCI/Dockerfile Drivers
  only where the Runtime Target capability proves support.
- Add source-transfer and retention contracts plus additional Artifact Destinations where delivery requirements justify
  them.

**Release gate:** supported source types create immutable, auditable Snapshots; a mutable branch or Application Workspace
cannot alter an already-submitted plan.

## Phase 3: Builder Pools And Multi-platform

- Add Builder Profiles, Instances and Pools with capacity, labels, affinity and region eligibility evidence.
- Add pool scheduling policies: round robin, least load, label selection, affinity and region.
- Extend Task Runtime only after its design explicitly supports distributed build legs; then add platform fan-out and
  final OCI manifest publication.

**Release gate:** a multi-platform plan selects compatible Instances deterministically, records every platform Artifact,
and publishes a manifest only when every required leg succeeds.

## Phase 4: Artifact Supply Chain And Automation

- Add Artifact promotion/copy, remote builders, distributed cache and additional OCI Artifact kinds.
- Add SBOM, provenance, signatures, attestations and supply-chain policy gates as Artifact-linked evidence.
- Add pipeline and deployment handoff that reference Artifacts and Publications rather than Build Jobs.

**Release gate:** promoted, scanned and signed Artifacts can be deployed independently of the Build Job that produced
them, with immutable provenance and publication history.

## Cross-phase Constraints

- Do not introduce a Graft-built Registry service; external registry providers remain Infrastructure integrations.
- Do not turn Builder Pool into a second Task Runtime or scheduler.
- Do not expose Docker-specific context paths, endpoints, credentials or implementation flags in the generic Build
  contract or review UI.
- Do not make a mutable tag the identity of an Artifact.
