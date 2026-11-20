# Runtime Diagnostics Recovery Trace

## 2026-11-20 Delivered Diagnostics History (PR #376, merged 9abe8cc1c)

- Additive-only contract locked (claim admission via new schema, frozen diagnostic_code enum).
- Delivered: receipt settlement diagnostics, agent claim admission telemetry, provider redaction,
  runtime-target readiness projection, web presentation; corrections 1-3 fixed admission timing,
  summary provenance, and latest-snapshot scheduling eligibility (d51d047f5).
- Evidence then: openapi-check 0, validate backend 0, web check 0, no live-DB migration in worktree.

## 2026-11-21 Bounded Auto-Recovery Round (worker track1-recovery, PR 2)

- Branch feature/bounded-execution-recovery from main 9abe8cc1c; scope: server/modules/task +
  scripts/validate_docker_isolated.py + its unit tests + this topic. No openapi/agent/runtime-target edits.
- Store (recovery_repository.go): ListInterruptedRecoveryCandidates covers lease-expired unknown
  retry_if_idempotent external stages AND unclaimable pending stages at attempt==max left by startup
  recovery (confirmed hang defect, fixed); ResumeInterruptedRecovery atomically resumes needs_attention
  task + all matching unknown stages (exactly one recovery_resolved, rollback when nothing resumable);
  ConvergeRecoveryBudgetExhausted converges stage+task to needs_attention with failure_code=
  auto_recovery_budget_exhausted, failure_class=internal, failure_stage=NULL, sanitized message;
  recovery_required stays exactly once via the unresolved guard.
- Runtime: periodic convergence goroutine (expireSubmissions renamed convergeExpiredWork) runs
  reconcileInterruptedRecovery on the same 30s ticker; per-task resume dedupe; state conflicts skipped
  as lost races; RetryBackoffMS honored as retryAt; no new goroutines or context.Background.
- Decision: auto path resumes only the runtime-recorded interruption signature
  (failure_code=external_execution_lease_expired); manual_reconcile stages and explicit
  needs_attention verdicts (provider receipts, consumer dispositions) are never auto-replayed.
- Follow-up decision point: consumer modules declaring retry_if_idempotent for idempotent operations is
  deferred to the owning modules per the product roadmap; every current consumer still submits
  manual_reconcile with MaxAttempts=1, so no delivered task stage changes recovery behavior.
- Harness: compose-material-stage-code joins LOCAL_REQUIRED_SCENARIOS; agent-claim-admission-rejected,
  agent-recovery-resume, agent-recovery-budget-exhausted join a new AGENT_REQUIRED_SCENARIOS enforced on
  agent probe runs (local fixtures forbid the agent service, so the local gate cannot require them).
  The two recovery scenarios declare retry_if_idempotent/budget through a documented fixture-level store
  stand-in (RECOVERY_POLICY_STAND_IN_NOTE in the script) solely to exercise the runtime policy path.
- Evidence: go test ./modules/task/... ok; python3 -B -m unittest scripts.test_validate_docker_isolated
  45 tests ok; cd server && go run ./cmd/graft validate backend exit 0; git diff --check clean;
  python3 scripts/validate_ai_plan_structure.py exit 0. Full live harness not run in the worktree.
- Primary-checkout follow-up (needs Docker + a registered task fixture):
  `python3 scripts/validate_docker_isolated.py init --project graft-isolated-<task> --source-revision <sha40>
  --backend-image <img> --agent-image <img> --driver-image <img> --execution-mode agent --directory <dir>`,
  then `... up --directory <dir>`, then `... probe --directory <dir>`; see tests/conformance/docker-runtime-agent/README.md.
