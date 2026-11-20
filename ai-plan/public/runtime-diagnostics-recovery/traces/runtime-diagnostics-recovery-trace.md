# Runtime Diagnostics Recovery Trace

## 2026-11-20 Bootstrap (worker track1-diagnostics)

- Bounded cross-boundary optimize topic. Locked: additive-only
  contract, sanitized summary (no paths/credentials/raw stderr), unchanged task ExecutionAdmission semantics.

## 2026-11-20 Contract Shape Decision

- Extending RuntimeTargetAgent.diagnostic_code enum fails the ADDITIVE-ONLY gate (enum-value-added level 3;
  empty approvals ledger). Lead-approved: keep it frozen; surface the tri-state via additive schema
  runtime-target-agent-claim-admission.yaml as claim_admission.class (no_work/identity_rejected/transport_failed)
  + observed_at. just openapi-breaking = 0 findings after change.

## 2026-11-20 Delivery Round (worker track1-diagnostics)

- Six logical commits: docs(ai-plan), feat(task) receipt settlement, feat(runtime) agent claim admission,
  feat(contract) additive schema, feat(runtime-target) readiness projection, feat(web) presentation.
- Evidence: openapi-check/breaking 0; validate backend 0 (migration gate, OpenAPI, lint, go test ./..., build);
  web bun run check 0 (315 files/2202 tests); repo-pinned golangci-lint v2.15.0 installed (stale v2.12.2
  could not load go1.27.4).
- Gaps: no live-DB migration (worktree forbids it); projection keeps one bounded per-target read; re-enrollment
  cadence is pre-existing platform behavior, out of scope.

## 2026-11-20 Correction Round 1 (worker track1-diagnostics)

- Five lead-verified findings fixed, no contract changes: admission persistence refresh keeps an ongoing
  rejection failing the local healthcheck; unchanged evidence re-reports on a 2-minute interval; concurrent
  claims merge rejection > transport > no_work, only a fully healthy cycle clears; task store retry/recovery/CAS
  resets clear failure_stage/failure_class with regressions; web renders the trusted summary with tests.
  openapi-check 0, validate backend 0, bun check 0.

## 2026-11-20 Correction Round 2 (worker track1-diagnostics)

- REAL 1 (fixed): collectClaimResults delayed admission behind long leases; it now returns its WaitGroup and
  pollExecutionCapabilities persists the merged conclusion and reports telemetry before waiting. Regression
  TestPollExecutionCapabilitiesObservesAdmissionBeforeExecutionWait.
- REAL 2 (fixed): taskDiagnosticSummary trusts failure_message only with failure_stage/failure_class provenance;
  drawer composes via shared taskFailureDetail; locale + drawer tests prove the gate.
- STALE: retries keep old details — reset since 0672c14c9 with regressions; summaries never appear — rendered
  since 350a4e58a; admission.go:97/132 comments — duplicates of 34ae6011f intervals.

## 2026-11-20 Correction Round 3 (worker track1-ledger-fix, user-authorized budget-exceeded)

- REAL (fixed): ListActiveDockerAgentLedgerSnapshots filtered receipt_diagnostic IN ('', no_work) +
  receipt_expires_at before the per-target pick, so an older unexpired healthy receipt masked the agent's latest
  transport_failed/identity_rejected (or expired) receipt; the target stayed schedulable while the projection
  showed degraded. Cause: row-level filter before per-target dedup.
- Disposition: eligibility decided by each target's LATEST snapshot; the query ranks snapshots per target first
  (ROW_NUMBER() OVER (PARTITION BY runtime_target_id ORDER BY sequence DESC, id DESC); DISTINCT ON is
  PostgreSQL-only and would break the SQLite store tests), then applies the diagnostic+expiry filter to the
  rank-1 row. Kept provider/generation/consumed_at/receipt_available/deleted guards, ordering, contract.
- Regression TestAgentLedgerSchedulingEligibilityFollowsLatestSnapshot: latest transport_failed + older no_work →
  not schedulable; latest no_work + older transport_failed → schedulable with latest snapshot; expired latest →
  not schedulable. Fails against the pre-fix query on both blocking scenarios.
- Evidence: go test runtime-target/... + build/... + task/store/... exit 0; validate backend exit 0; git diff
  --check clean; committed in this round's fix(runtime-target) commit (not pushed).
