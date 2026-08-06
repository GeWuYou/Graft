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

## 2026-08-07 Phase 2

- Added a request-to-store availability bridge without coupling Axios to Pinia.
- Network failures and 502/503/504 report candidate failures; healthz, auth errors, and business 500 responses do not.
- The store controls TanStack Query online state and retry eligibility; unavailable requests and NDJSON streams fail fast.
- Focused validation passed: lint, typecheck, request tests, and diff check.

## 2026-08-07 Phase 3

- Added a shared realtime availability registry used by WebSocket, SSE, and Terminal transports.
- Platform freeze closes active transports and cancels reconnect timers; transition-based recovery avoids reconnect storms.
- Terminal reconnects only when its previous resize context is still valid, then requests a new server ticket.
- Focused validation passed: websocket, SSE, and terminal session tests plus typecheck.

## 2026-08-07 Phase 4

- Added static capability snapshot contract and generated OpenAPI bindings for `GET /api/platform/capabilities`.
- Registered `CapabilityCoordinator` as core authority with PostgreSQL, Redis, and outbound-network observation providers.
- Network module exposes its existing latest connectivity aggregate through `CapabilityObservationSource`; no duplicate persistence or probe pipeline was added.
- Registered a dashboard health widget that consumes coordinator observations through the existing widget contribution system.
- Validation passed: `go test ./internal/... ./modules/network`, `bun run typecheck`, `bun run lint`, OpenAPI bundle/runtime-path freshness, and `git diff --check`.

## 2026-08-07 Phase 5

- Service Unavailable now offers a single retry path, return-home action, and copyable diagnostic context.
- Diagnostic copy reuses the shared observability clipboard helper and TDesign message feedback; no new notification authority was added.
- Backend tests/build/OpenAPI checks and frontend typecheck/lint/contract checks passed.
- Full `bun run check` remains blocked only by the pre-existing missing Monaco Dockerfile language import; focused container detail tests passed (101 tests).

## 2026-08-07 Corrective Review

- Dashboard health projection now maps capability-only `checking`, `unavailable`, and `unsupported` states to the existing dashboard enum instead of leaking invalid wire values.
- Regression coverage added at the app projection seam; `go test ./internal/... ./modules/network`, backend build, OpenAPI validation, frontend lint/typecheck, and focused web tests passed.
