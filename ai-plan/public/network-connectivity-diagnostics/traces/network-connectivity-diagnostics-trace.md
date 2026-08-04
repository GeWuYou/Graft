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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-1-authority-registry-probe-core",
    "phase-2-persistence-api-security"
  ],
  "pending_batches": [
    "phase-3-web-experience-integration"
  ],
  "current_batch": "phase-3-web-experience-integration",
  "next_batch": null,
  "closeout_status": "phase-2-settled"
}
```
