# Runtime Diagnostics Recovery Trace

## 2026-11-20 Work Intake Bootstrap

- Classified as a bounded cross-boundary optimize topic created by delegated worker `track1-diagnostics` on branch
  `feature/runtime-diagnostics-projection`.
- Created the recovery materials before implementation commits per graft-work-intake.
- Locked the additive-only contract rule, the sanitized diagnostic summary boundary (no paths, credentials, or raw
  provider stderr), and unchanged task `ExecutionAdmission` semantics.
- Next step: implement the contract/backend settlement slice, then agent admission, provider redaction, runtime-target
  projection, web presentation, and the OpenAPI closure.

## 2026-11-20 Contract Shape Decision: Additive claim_admission

- The brief suggested extending `RuntimeTargetAgent.diagnostic_code` with the claim-admission tri-state, but the
  repository oasdiff gate classifies `response-property-enum-value-added` as an unapproved breaking change (level 3),
  and `openapi/approved-breaking.json` has an empty approvals ledger, so the enum-extension shape fails the
  ADDITIVE-ONLY gate with 9 findings.
- Decision (approved by the lead): keep `RuntimeTargetAgent.diagnostic_code` frozen and surface the tri-state through
  the new additive schema `runtime-target-agent-claim-admission.yaml` projected as `RuntimeTargetAgent.claim_admission`
  with `class` (`no_work` / `identity_rejected` / `transport_failed`) and `observed_at`. Tri-state distinguishability
  is preserved without opening the approved-breaking ledger.
- `just openapi-breaking` reports 0 findings after the change; recorded here as the topic's contract-shape precedent.

## 2026-11-20 Delivery Round (worker track1-diagnostics)

- Implemented the full bounded surface on branch `feature/runtime-diagnostics-projection` in six logical commits:
  docs(ai-plan) topic creation, feat(task) receipt settlement diagnostics, feat(runtime) agent claim admission,
  feat(contract) additive schema closure, feat(runtime-target) readiness projection, feat(web) presentation.
- Validation evidence at the committed content: `just openapi-check` exit 0; `just openapi-breaking` exit 0 with 0
  findings; `cd server && go run ./cmd/graft validate backend` exit 0 (migration gate, OpenAPI, lint guards,
  golangci-lint production + test configs, go test ./..., build); `cd web && bun run check` exit 0 (315 test files,
  2202 tests).
- Environment note: the local golangci-lint binary was stale (v2.12.2, go1.26 build) and could not load the go1.27.4
  target; the repo-pinned v2.15.0 was installed into the user Go bin directory to run the mandated blocking lint.
- Remaining known gaps: the migration has not been applied to a live database (numbered worktree forbids live
  migrations; bootstrap container check passed); the Runtime Target projection adds one bounded per-target read in
  the list path, consistent with the existing per-target agent trust reads; the identity-rejection re-enrollment loop
  cadence is pre-existing platform behavior and out of this topic's scope.

## 2026-11-20 Correction Round 1 (worker track1-diagnostics)

- Applied all five lead-verified Greptile findings; none required contract changes and the approved additive
   shape was kept.
- Agent admission: bounded persistence refresh keeps an ongoing rejection failing the local healthcheck; unchanged
  evidence re-reports on a 2-minute refresh interval so backend receipts are refreshed before expiry; concurrent
  per-capability claim results merge with rejection > transport > no_work priority and only a fully healthy cycle
  clears evidence.
- Task store: retry and recovery resets clear failure_stage/failure_class alongside the existing failure fields;
  CAS transitions reset the diagnostic columns; store regression covers the cleared retry facts and fresh attempt.
- Web: the trusted sanitized summary renders as its own segment next to the code label; drawer-level and helper tests
  assert actual rendering; runtime-target detail tests cover claim admission classes and the no-receipt absence.
- Validation after corrections: just openapi-check exit 0; cd server && go run ./cmd/graft validate backend exit 0
  (both lint configs 0 issues); cd web && bun run check exit 0.

## 2026-11-20 Correction Round 2 (worker track1-diagnostics)

- REAL 1 (fixed): `collectClaimResults` returned only after executions finished, delaying admission observation behind
  long-running leases. The collector now returns its WaitGroup and `pollExecutionCapabilities` persists the merged
  claim conclusion and reports telemetry before waiting for executions; rank merge and clear semantics unchanged.
  Regression `TestPollExecutionCapabilitiesObservesAdmissionBeforeExecutionWait` blocks a renew to prove the
  persisted admission lands while the execution is still running.
- REAL 2 (fixed): the summary is now trusted only with provenance — `taskDiagnosticSummary` renders `failure_message`
  only when the row carries `failure_stage` or `failure_class` (the settlement path always writes stage+class+summary
  together); ordinary runtime errors saved without attribution stay unexposed. Single-line/length checks retained.
  The drawer now composes through the shared `taskFailureDetail` helper for the task alert and every stage
  description. Tests: presentation trust-gate cases in both locales (summary-without-stage rejected), composition
  asserts the unprovenanced path text is withheld, and a drawer-level case renders an ordinary failure row with the
  code label only.
- STALE dispositions (verified locally, no code change):
  (a) "Retries keep old failure details" — stale: retry/CAS/launch paths already NULL failure_stage/failure_class at
  `server/modules/task/store/sql_repository.go:445/476/845/1271/1514` since commit `0672c14c9`, with regression
  tests `TestExternalExecutionRetryClearsFailureDiagnostics` and `TestExternalExecutionRetryCreatesFreshFencedAttempt`.
  (b) "Saved summaries never appear" — stale: the drawer renders the summary segment through the shared composition
  since commit `350a4e58a`; `TaskDetailDrawer.test.ts` asserted it before this round and now also proves the
  provenance gate.
  (c) admission.go:97 comment — stale duplicate of correction 1 finding: bounded persistence refresh landed in
  commit `34ae6011f` (`admissionPersistenceInterval`).
  (d) admission.go:132 comment — stale duplicate of correction 1 finding: unchanged-evidence refresh landed in
  commit `34ae6011f` (`admissionTelemetryRefreshInterval`).
