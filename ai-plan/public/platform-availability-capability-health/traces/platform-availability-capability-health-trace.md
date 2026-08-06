# Platform Availability And Capability Health Trace

## 2026-08-07 bootstrap

- Confirmed `cross-boundary` authority chain and empty implementation worktree.
- Accepted `CapabilityImpact` values `platform`, `feature`, `advisory` and a standard category field.
- Started Phase 0 with server contracts and coordinator ownership.

## Locked Decisions

- `PlatformAvailabilityStore` is the browser availability authority.
- `CapabilityCoordinator` is the server capability-health authority.

## 2026-08-07 Phase 0

- Added stable `moduleapi` capability DTOs and provider contract.
- Added deterministic static registry and coordinator with provider error normalization and TTL-based freshness projection.
- Capability status is limited to `unknown`, `checking`, `healthy`, `degraded`, `unavailable`, `disabled`, and `unsupported`.
- Expired observations project to `unknown` while preserving `ObservedAt`, `ExpiresAt`, and `Stale` freshness metadata.
- Registry validates descriptor category, impact, key, provider, and non-negative TTL; invalid provider statuses normalize to `unavailable`.
- Focused validation: `go test ./internal/moduleapi ./internal/capability`; `git diff --check`.
- Query, Dashboard, Banner, Monitor, and module APIs are projections or observation sources only.

## Loop Batch State

```json
{"loop_mode":"topic-completion-loop","completed_batches":["phase-0-server-foundation"],"pending_batches":["phase-1-web-availability","phase-2-query-gating","phase-3-realtime","phase-4-capability-projection","phase-5-recovery-diagnostics"],"current_batch":null,"next_batch":"phase-1-web-availability","closeout_status":"batch-complete"}
```

## 2026-08-07 Phase 1

- Added the shell-owned PlatformAvailabilityStore, anonymous healthz probe, static service-unavailable route, and preserved-route recovery path.
- Router guards redirect only after the store has confirmed platform unavailability; authentication remains separate.
- Focused frontend validation passed: lint, typecheck, Provider and route-guard tests, and diff check.
- Full `bun run check` remains blocked by existing Monaco language-definition imports and unrelated container-detail test failures.
