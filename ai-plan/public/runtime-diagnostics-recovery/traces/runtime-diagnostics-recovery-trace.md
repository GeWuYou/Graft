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
