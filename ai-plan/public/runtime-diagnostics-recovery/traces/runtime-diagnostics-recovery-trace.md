# Runtime Diagnostics Recovery Trace

## 2026-11-20 Delivered Diagnostics History (PR #376, merged 9abe8cc1c)

- Additive-only contract locked (claim admission via new schema, frozen diagnostic_code enum); delivered
  settlement diagnostics, agent claim admission telemetry, provider redaction, readiness projection, web
  presentation; corrections 1-3 fixed admission timing, summary provenance, latest-snapshot eligibility.

## 2026-11-21 Bounded Auto-Recovery Round (worker track1-recovery, PR 2)

- Branch feature/bounded-execution-recovery from main 9abe8cc1c; scope: server/modules/task +
  scripts/validate_docker_isolated.py + its unit tests + this topic; no openapi edits.
- Store: candidates cover lease-expired unknown retry_if_idempotent external stages plus unclaimable
  pending stages at attempt==max; ResumeInterruptedRecovery atomically resumes the needs_attention task
  with all matching unknown stages; ConvergeRecoveryBudgetExhausted converges stage+task to
  needs_attention with budget-exhaustion diagnostics; reconcileInterruptedRecovery runs on the 30s
  convergence ticker.
- Decisions: auto path resumes only the runtime-recorded interruption signature
  (failure_code=external_execution_lease_expired); manual_reconcile and explicit needs_attention
  verdicts are never auto-replayed. Consumer retry_if_idempotent adoption is deferred to the owners.
- Harness: compose-material-stage-code joins LOCAL_REQUIRED_SCENARIOS; three agent recovery scenarios
  join a new AGENT_REQUIRED_SCENARIOS with a documented store stand-in
  (RECOVERY_POLICY_STAND_IN_NOTE) to exercise the runtime policy path only.
- Primary-checkout follow-up: run validate_docker_isolated.py init/up/probe with --execution-mode agent
  per tests/conformance/docker-runtime-agent/README.md.

## 2026-11-21 Correction History (PR #380 / #382 / #383)

- Corrections 1-11: material stage attribution owner shapes; capability-binding claim rejection
  injection; agent expired-journal settlement; ErrRecoveryBlocked whole-task gating; bounded progress
  past blocked tasks; cancel re-check; exact compose label exclusion; preflight before login; agent
  bounce and cleanup on failure. Full detail in archive/corrections-history.md.
- Correction 12 (platform defect, live-converged) is recorded below; corrections 4-11 drove it.

## 2026-11-21 Correction 12 (PR #387, platform defect, live-converged)

- Root cause of every 503 cascade: the execution-admission monitor latched permanently on ONE failed
  ownership-connection ping (1s monitor), refusing claims/submissions with execution_ownership_lost
  until backend restart. The monitor now keeps a bounded state machine: single ping failures stay
  ready; 5 consecutive failures degrade with the additive internal code execution_admission_degraded
  (refuses new claims/submissions, in-flight work continues); degraded attempts bounded re-acquisition
  (unlock old session, fresh advisory lock); recovery or successful re-acquisition restores ready
  without restart; only a failed re-acquisition (another runtime owns execution) latches lost with the
  existing semantics. Claim requests are explicitly gated by the admission state.
- Contract closure (lead-approved additive): execution_admission_degraded joined the frozen
  RuntimeTargetExecutionDiagnosticCode enum (runtime-target-execution.yaml + regenerated server/web
  artifacts); the four affected runtime-target operations carry response-property-enum-value-added
  approvals in openapi/approved-breaking.json (breaking gate 0 findings).
- Regressions: ownership state machine (N-1 ready, N degraded, self-heal, conflict latch, bounded
  backoff) and a runtime test that mid-run transient failures neither refuse admission nor strand the
  accepted task.
- Evidence: harness 56 ok; go test modules/task + store ok; both golangci configs 0 issues; validate
  backend 0; openapi-check 0; openapi-breaking 0 (4 approved additive entries); diff --check clean.
  Live rerun pending (recovery scenarios should pass with self-healing).
