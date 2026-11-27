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
  join a new AGENT_REQUIRED_SCENARIOS. Recovery scenarios use a documented store stand-in
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
- Correction 4 (live run 2e16e610c): both recovery probes hit 503 — the execution-admission latch had
  fired earlier; probes pre-check readiness, run before the kill-heavy scenarios, and start the agent
  only after the reconcile re-arms the stage.
- Correction 5 (thread 4647483896): the budget probe's trailing agent restart overwrote the budget
  diagnostics via a journal replay; converge, record, leave the agent down.
- Correction 6 (live run agent-smoke2): 401 token-ageing cascade; api() re-authenticates once on
  expiry; port-squat preflight added; claim wait split.
- Correction 7 (PR #383): the preflight flagged own containers in CI because the official instance
  carries the -official suffix; it now takes the legitimate project list per fixture and excludes by
  compose label. The claim wait accepted the task reaching a terminal status.
- Correction 8 (thread 4721373969): name-prefix exclusion let sibling projects bypass contamination;
  exclusion is exact compose label only — sibling-prefixed, label-less and foreign holders of recorded
  ports all fail loudly.
- Correction 9 (thread 4721691577): the preflight ran after login, hiding port squatters behind login
  failures; it now precedes login (ordering regression added).
- Corrections 4-9 evidence: harness and validate backend green after each round; live reruns drove
  each fix.

## 2026-11-21 Correction 10 (PR #383, live rerun run-3 failure chain)

- Written chain from the retained run-3 evidence: the resume probe's restarted agent never claimed the
  re-armed stage (stage pending/attempt 1, task running, sampled across both claim windows) and that
  single stuck agent explains every later failure — compose-up task 6 and interruption task 8 were
  accepted but never executed (no container), claim-admission observed nothing because a stuck agent
  never claims, and budget stayed unverified. No 401 cascade this run (token refresh worked); the
  admission latch did not fire (budget up 202, agent-offline 202).
- Probes now clean up their own failures: claim retry with one agent bounce, then on every failure path
  a best-effort API cancel plus fixture-level SQL terminalization of the stuck task and its pending or
  unknown stages, so no later scenario inherits a blocked application or a stuck agent-claim target.
- Evidence: harness 56 ok; validate backend 0; diff --check clean. Live rerun pending.
