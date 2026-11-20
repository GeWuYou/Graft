# Runtime Diagnostics Recovery Trace

## 2026-11-20 Delivered Diagnostics History (PR #376, merged 9abe8cc1c)

- Additive-only contract locked (claim admission via new schema, frozen diagnostic_code enum); delivered
  settlement diagnostics, agent claim admission telemetry, provider redaction, readiness projection, web
  presentation; corrections 1-3 fixed admission timing, summary provenance, and latest-snapshot
  scheduling eligibility (d51d047f5).

## 2026-11-21 Bounded Auto-Recovery Round (worker track1-recovery, PR 2)

- Branch feature/bounded-execution-recovery from main 9abe8cc1c; scope: server/modules/task +
  scripts/validate_docker_isolated.py + its unit tests + this topic; no openapi edits.
- Store: candidates cover lease-expired unknown retry_if_idempotent external stages plus unclaimable
  pending stages at attempt==max (startup hang defect, fixed); ResumeInterruptedRecovery atomically
  resumes the needs_attention task with all matching unknown stages; ConvergeRecoveryBudgetExhausted
  converges stage+task to needs_attention with budget-exhaustion diagnostics and exactly-once recovery
  events. Runtime: reconcileInterruptedRecovery runs on the 30s convergence ticker.
- Decisions: auto path resumes only the runtime-recorded interruption signature
  (failure_code=external_execution_lease_expired); manual_reconcile and explicit needs_attention
  verdicts are never auto-replayed. Consumer retry_if_idempotent adoption is deferred to the owning
  modules per the product roadmap (all current consumers submit manual_reconcile MaxAttempts=1).
- Harness: compose-material-stage-code joins LOCAL_REQUIRED_SCENARIOS; three agent recovery scenarios
  join a new AGENT_REQUIRED_SCENARIOS. Recovery scenarios use a documented store stand-in
  (RECOVERY_POLICY_STAND_IN_NOTE) solely to exercise the runtime policy path.
- Primary-checkout follow-up (needs Docker + a registered task fixture): run validate_docker_isolated.py
  init/up/probe with --execution-mode agent per tests/conformance/docker-runtime-agent/README.md.

## 2026-11-21 Correction History (PR #380)

- Correction 1 (CI run 44777511166): local settlements deliver the material stage attribution as
  failure_code=local_material_resolve_failed with failure_stage=NULL; the probe asserts each owner's
  delivered shape (agent projects failure_stage=material_resolve).
- Correction 2: admission probe diverges the capability binding version so fresh claims are rejected
  with telemetry intact; agent expired journals settle the delivered interruption receipt and clear;
  whole-task resume refuses with ErrRecoveryBlocked while any unknown stage is over budget.
- Correction 3 (threads 4645868796 + 4645868804): the candidate query excludes resumable legs whose task
  holds any unknown over-budget stage (bounded progress past blocked tasks — regression: 100 blocked +
  1 newer recoverable); the resume transaction re-checks cancel_requested_at and rejects recorded
  cancellations (no stranding); ResumeInterruptedRecovery split to pass cyclop <= 10.

## 2026-11-21 Correction 4 (standing-authorized, agent-mode live run 2e16e610c)

- Both recovery probes recorded unverified: the ups were refused 503 common.task_execution_unavailable —
  the runtime's execution-admission latch (execution_ownership_lost) had fired earlier in the run; the
  latch trigger is not in the captured evidence (transient ownership-ping failure suspected — platform
  follow-up, permanent until backend restart).
- Probes now pre-check the target execution readiness projection and record its diagnostic, run before
  the kill-heavy scenarios, and start the agent only after the reconcile re-arms the stage: the agent's
  expired-journal settlement is then always superseded and can no longer overwrite the
  external_execution_lease_expired resume signal (a real journal-vs-reconcile race found while tuning).
- Evidence: go test modules/task + agent internal ok; validate backend 0; harness unittest 47 ok;
  git diff --check clean. Agent-mode scenarios remain live CI verification.
