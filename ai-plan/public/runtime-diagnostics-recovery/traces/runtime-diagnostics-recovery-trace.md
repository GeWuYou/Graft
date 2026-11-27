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

- Correction 1 (CI run 44777511166): local settlements deliver the material stage attribution as
  failure_code=local_material_resolve_failed with failure_stage=NULL; the probe asserts each owner shape.
- Correction 2: admission probe diverges the capability binding version so fresh claims are rejected
  with telemetry intact; agent expired journals settle the interruption receipt and clear; whole-task
  resume refuses with ErrRecoveryBlocked while any unknown stage is over budget.
- Correction 3 (threads 4645868796 + 4645868804): bounded progress past blocked tasks; the resume
  transaction re-checks cancel_requested_at; cyclop refactor.
- Correction 4/5 (live run 2e16e610c): both probes hit 503 from the earlier execution-admission latch;
  probes now pre-check readiness and start the agent only after the reconcile re-arms the stage; the
  budget probe never restarts after convergence (a journal replay would overwrite the diagnostics).
- Correction 6/7 (live run agent-smoke2, PR #383): 401 token-ageing cascade — api() re-authenticates
  once on expiry; the preflight takes the legitimate project list per fixture (the -official instance
  was rejecting its own parent containers) and the claim wait accepts a settled task.
- Correction 8/9 (threads 4721373969 + 4721691577): exclusion is exact compose label only (siblings
  bypassed name-prefix matching) and the preflight now precedes login (squatters hid behind login
  failures); both with ordering/exclusion regressions.
- Correction 10 (run-3 chain): the restarted agent never claimed the re-armed stage and every later
  agent-dependent scenario failed behind it; probes bounce the agent once for the claim and clean up
  their own failures (API cancel + fixture-level SQL terminalization).
- Correction 11 (PR #387, thread 4722371822): a claimed replay that never reached success skipped
  cleanup, occupying the application; the timeout path now calls cleanup_recovery_task too.
  Regressions assert cleanup on failed outcomes, none on success.
- Evidence: harness 56 ok; validate backend 0; diff --check clean. Live rerun pending.
