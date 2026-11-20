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
  pending stages at attempt==max; ResumeInterruptedRecovery atomically resumes the needs_attention task
  with all matching unknown stages; ConvergeRecoveryBudgetExhausted converges stage+task to
  needs_attention with budget-exhaustion diagnostics. Runtime: reconcileInterruptedRecovery runs on the
  30s convergence ticker.
- Decisions: auto path resumes only the runtime-recorded interruption signature
  (failure_code=external_execution_lease_expired); manual_reconcile and explicit needs_attention
  verdicts are never auto-replayed. Consumer retry_if_idempotent adoption is deferred to the owning
  modules (all current consumers submit manual_reconcile MaxAttempts=1).
- Harness: compose-material-stage-code joins LOCAL_REQUIRED_SCENARIOS; three agent recovery scenarios
  join a new AGENT_REQUIRED_SCENARIOS. Recovery scenarios use a documented store stand-in
  (RECOVERY_POLICY_STAND_IN_NOTE) solely to exercise the runtime policy path.
- Primary-checkout follow-up (needs Docker + a registered task fixture): run validate_docker_isolated.py
  init/up/probe with --execution-mode agent per tests/conformance/docker-runtime-agent/README.md.

## 2026-11-21 Correction History (PR #380 / #382)

- Correction 1 (CI run 44777511166): local settlements deliver the material stage attribution as
  failure_code=local_material_resolve_failed with failure_stage=NULL; the probe asserts each owner shape.
- Correction 2: admission probe diverges the capability binding version so fresh claims are rejected
  with telemetry intact; agent expired journals settle the delivered interruption receipt and clear;
  whole-task resume refuses with ErrRecoveryBlocked while any unknown stage is over budget.
- Correction 3 (threads 4645868796 + 4645868804): bounded progress past blocked tasks; the resume
  transaction re-checks cancel_requested_at; cyclop refactor.
- Correction 4 (agent-mode live run 2e16e610c): both recovery probes hit 503 — the execution-admission
  latch (execution_ownership_lost) had fired earlier; probes pre-check readiness, run before the
  kill-heavy scenarios, and start the agent only after the reconcile re-arms the stage.
- Correction 5 (thread 4647483896): the budget probe's trailing agent restart overwrote the budget
  diagnostics via a journal replay; converge, record, leave the agent down; GO regression pins the
  delivered supersede semantics.

## 2026-11-21 Correction 6 (agent-mode live run agent-smoke2)

- The rerun failed on a 401 AUTH_TOKEN_EXPIRED cascade from the claim-admission probe onward (56 401s):
  the login token aged out mid-run; the orphan stack's port presence remains a possible
  confounder. api() now re-authenticates once on AUTH_TOKEN_EXPIRED (never on login paths) and retries
  the same request.
- A port-squat preflight (assert_fixture_ports_exclusive) fails the run before any scenario when a
  foreign compose project holds the recorded fixture ports, naming the squatter.
- The rerun also exposed the restarted agent never claiming the re-armed stage (attempt stayed 1):
  the resume probe now waits for the claim (bounded) and records the unclaimed stage evidence;
  root cause needs agent logs.
- Evidence: harness 52 ok; validate backend 0; diff --check clean. Live rerun pending.
