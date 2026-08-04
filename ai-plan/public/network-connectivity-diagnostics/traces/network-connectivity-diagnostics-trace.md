# Network Connectivity Diagnostics Trace

## 2026-08-04 Work Intake Bootstrap

- Classified as a long-running cross-boundary feature.
- Created recovery materials for a three-phase topic-completion loop.
- Locked target-addressed diagnostics, a shared connectivity store, extensible probe results, and protected Exit IP disclosure.

## 2026-08-04 Phase 1 Authority, Registry, And Probe Core

- Added the Network-owned canonical Connectivity Target Registry alongside the existing temporary HTTP diagnostic adapter.
- Added typed target/probe capabilities and versioned, immutable, sanitized `ConnectivityReport`/`ProbeResult[]` contracts.
- Adapted Platform Update to declare its target capabilities and produce the new report envelope without exposing its internal URL.
- Completed the network design, ADR-021, and phased roadmap.
- Focused tests cover registry ordering/uniqueness, immutable capability/report snapshots, sensitive text/full Exit-IP exclusion, and Platform Update adaptation.
- This trace records round evidence only; loop batch-state transitions remain controller-owned.

## 2026-08-04 Controller Settle: Phase 1

- Independently verified `ca21f69a`, including focused Network/Update tests and vet.
- Accepted Phase 1 into the completed batch set and advanced the active batch to persistence, APIs, and security.

## 2026-08-04 Controller Settle: Phase 2

- Accepted `5e932f1c` and `a0677d65` after independent targeted verification.
- The Network module now owns target/check/report persistence, target-addressed batch and detailed APIs, typed OpenAPI
  contracts, bounded report retention, route explanation, and custom HTTP(S) target management.
- Custom targets reject non-public destinations and redirect traversal, pin each direct dial to a freshly validated
  literal address, and are deliberately unavailable while an outbound proxy is enabled because proxy resolution would
  invalidate the DNS-rebinding control.
- Exit IP remains masked in storage, history, and exports. The reserved read permission is retained for a future
  live-only, audited disclosure surface; Phase 3 must not infer or display an unmasked value.
- Advanced the active batch to the shared ConnectivityStore and platform connectivity web experience.

## 2026-08-04 Controller Settle: Phase 3 And Archive Readiness

- Completed a single Network-owned ConnectivityStore that shares targets, latest checks, aggregates, reports, trace,
  history, export, and custom target state between the batch and target-detail pages.
- Batch health now provides aggregate context (last run, success rate, average and worst latency), automatic reruns,
  latency/status sorting, compact health status, and permission-gated custom HTTP(S) target management.
- Target diagnostics remains target-addressed across reruns and history selection; it renders typed trace evidence,
  capability-driven route/Exit-IP/history surfaces, and exports only the sanitized report representation.
- Focused Network Vitest, TypeScript, i18n, OpenAPI frontend governance, stylelint, hygiene, and diff checks pass.
- `bun run build:release` remains blocked by an unrelated repository dependency: Monaco imports a missing
  `monaco-editor/esm/vs/languages/definitions/dockerfile/register.js` file. No Network source is in that error path.
- Final acceptance review confirms target identity, shared execution/persistence model, extensible probe/report shape,
  and masked Exit-IP behavior. The topic is archive-ready.

## 2026-08-04 Phase 4 HTTP Status Summary Projection

- Added a nullable `http_status` field to the Network-owned `platform_connectivity_checks` summary table through a
  forward-only migration, with a database constraint limiting stored response codes to 100–599.
- Extended the sanitized `ProbeResult` contract with an optional HTTP response code. The constructor removes the field
  from non-HTTP probes, invalid values, and caller-owned pointers before reports are persisted or projected.
- The SQL store derives the summary only from the final HTTP probe: a received valid response is persisted and read
  through latest/history/batch/run responses; a target without HTTP or without a response remains explicitly null.
- The canonical OpenAPI schemas, bundle, generated Go types, runtime path artifacts, module migration embed, and web
  generated schema were regenerated together. Batch Connectivity now renders a compact HTTP Status column with an
  explicit unavailable value, while detailed traces remain target-detail only.
- Focused and full Go tests, Network Vitest, web type/i18n checks, SQL migration validation, and OpenAPI checks pass.
  `bun run build:release` remains blocked by the unrelated absent Monaco Dockerfile registration module.

## 2026-08-04 Controller Settle: Phase 4 And Final Archive Readiness

- Independently re-ran the Network/Update/moduleapi tests, full server test suite, vet, SQL migration validation,
  OpenAPI closure, Network Vitest, web typecheck, and strict i18n validation for `4c61b485`.
- Accepted the structured nullable HTTP-status summary as the final missing batch-health field. It retains the compact
  batch projection while trace data remains available only from target diagnostics.
- All product acceptance conditions are met: target identity, shared store/execution model, capability-driven probes,
  route explanation, history/trace/export, custom target management, aggregate health, and masked Exit-IP handling.
- The topic remains archive-ready. The release build limitation is confined to the pre-existing missing Monaco
  Dockerfile registration module and is recorded without attributing it to Network.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-1-authority-registry-probe-core",
    "phase-2-persistence-api-security",
    "phase-3-web-experience-integration",
    "phase-4-http-status-summary-projection"
  ],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
