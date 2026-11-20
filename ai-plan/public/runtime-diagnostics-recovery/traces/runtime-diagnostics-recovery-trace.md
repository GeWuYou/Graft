# Runtime Diagnostics Recovery Trace

## 2026-11-20 Delivered Diagnostics History (PR #376, merged 9abe8cc1c)

- Additive-only contract locked (claim admission via new schema, frozen diagnostic_code enum).
- Delivered: settlement diagnostics, agent claim admission telemetry, provider redaction, readiness
  projection, web presentation; corrections 1-3 fixed admission timing, summary provenance, and
  latest-snapshot scheduling eligibility (d51d047f5).

## 2026-11-21 Bounded Auto-Recovery Round (worker track1-recovery, PR 2)

- Branch feature/bounded-execution-recovery from main 9abe8cc1c; scope: server/modules/task +
  scripts/validate_docker_isolated.py + its unit tests + this topic; no openapi edits.
- Store: candidates cover lease-expired unknown retry_if_idempotent external stages plus unclaimable
  pending stages at attempt==max (startup hang defect, fixed); ResumeInterruptedRecovery atomically
  resumes the needs_attention task with all matching unknown stages (one recovery_resolved, rollback
  when nothing resumable); ConvergeRecoveryBudgetExhausted converges stage+task to needs_attention with
  auto_recovery_budget_exhausted/internal/failure_stage=NULL diagnostics; recovery_required exactly once.
  Runtime: reconcileInterruptedRecovery runs on the 30s convergence ticker (expireSubmissions renamed
  convergeExpiredWork); per-task dedupe, conflicts skipped as lost races.
- Decisions: auto path resumes only the runtime-recorded interruption signature
  (failure_code=external_execution_lease_expired); manual_reconcile stages and explicit
  needs_attention verdicts are never auto-replayed. Consumer modules declaring retry_if_idempotent for
  idempotent operations is deferred to the owning modules per the product roadmap (every current
  consumer submits manual_reconcile with MaxAttempts=1, so no delivered stage changes behavior).
- Harness: compose-material-stage-code joins LOCAL_REQUIRED_SCENARIOS; three agent recovery scenarios
  join a new AGENT_REQUIRED_SCENARIOS (local fixtures forbid the agent service). Recovery scenarios
  declare retry_if_idempotent/budget via a documented store stand-in (RECOVERY_POLICY_STAND_IN_NOTE)
  solely to exercise the runtime policy path.
- Primary-checkout follow-up (needs Docker + a registered task fixture): run validate_docker_isolated.py
  init/up/probe with --execution-mode agent per tests/conformance/docker-runtime-agent/README.md.

## 2026-11-21 Correction 1 (PR #380)

- CI run 44777511166 (local): compose-material-stage-code asserted the agent shape; local settlements
  deliver the stage attribution as failure_code=local_material_resolve_failed (stage folded into the
  code, failure_stage=NULL). Probe now asserts each owner's delivered shape; regressions cover both.

## 2026-11-21 Correction 2 (PR #380, lead-verified REAL x3)

- Admission probe: generation mutation cannot reject fresh claims (identity middleware derives the
  generation from the server binding). The probe now diverges the capability binding version so
  agentCapabilityAllowed rejects every fresh claim (400 -> identity_rejected) while the active trust
  generation keeps telemetry intact; original value restored exactly.
- Scope amendment (agent expired-journal path only): an expired persisted journal renewed first and
  blocked every cycle, so server-resumed stages were never claimed. recoverExecutionJournals now
  settles expired journals as the delivered interruption receipt (needs_attention /
  agent_execution_interrupted) and clears them; identity rejections still fail closed, and a
  server-superseded receipt clears the stale journal so polling proceeds.
- Mixed-budget legs: whole-task resume refuses with ErrRecoveryBlocked while any unknown stage is over
  budget; reconcile converges exhausted legs first and leaves the task needs_attention with per-leg
  diagnostics so the blocked leg stays visible.
- Evidence: go test modules/task + agent internal ok; validate backend 0; harness unittest 45 ok;
  git diff --check clean. Live agent-mode scenarios remain CI/primary-checkout verification.
