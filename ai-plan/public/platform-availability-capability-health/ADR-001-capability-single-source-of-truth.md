# ADR-001: Platform Availability And Capability Health Authorities

- Status: accepted
- Scope: server capability contracts and web availability projection

## Decision

`PlatformAvailabilityStore` is the only browser authority for platform reachability. `CapabilityCoordinator` is the
only server authority for capability health. Query caches, Dashboard cards, banners, Monitor, and module APIs may
observe, contribute bounded observations, or project snapshots, but may not maintain a competing health truth.

Capability descriptors standardize `category` and `impact` (`platform`, `feature`, `advisory`). Runtime targets,
resources, and operations remain outside the global capability registry.
