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
