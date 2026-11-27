# Runtime Diagnostics Recovery — Corrections Archive (PR #380 / #382 / #383)

## Correction 1 (CI run 44777511166)

- Local settlements deliver the material stage attribution as failure_code=local_material_resolve_failed
  with failure_stage=NULL (agent settlements project failure_stage=material_resolve); the probe asserts
  each owner's delivered shape.

## Correction 2

- Admission probe diverges the capability binding version so fresh claims are rejected with telemetry
  intact (generation mutation cannot reject — the identity middleware derives the generation from the
  server binding).
- Agent expired journals settle the delivered interruption receipt (needs_attention /
  agent_execution_interrupted) and clear so server-resumed stages are claimed again; identity
  rejections still fail closed.
- Whole-task resume refuses with ErrRecoveryBlocked while any unknown stage is over budget.

## Correction 3 (threads 4645868796 + 4645868804)

- The candidate query excludes resumable legs whose task holds any unknown over-budget stage (bounded
  progress past blocked tasks; regression: 100 blocked + 1 newer recoverable).
- The resume transaction re-checks cancel_requested_at and rejects recorded cancellations (no
  stranding).
- ResumeInterruptedRecovery split into begin/guard/resume-stage helpers (cyclop <= 10).

## Correction 4/5 (live run 2e16e610c, thread 4647483896)

- Both recovery probes hit 503 from the earlier execution-admission latch; probes pre-check readiness,
  run before the kill-heavy scenarios, and start the agent only after the reconcile re-arms the stage
  (a real journal-vs-reconcile race found while tuning).
- The budget probe never restarts after convergence (a journal replay would overwrite the budget
  diagnostics); GO regression pins the delivered supersede semantics.

## Correction 6/7 (live run agent-smoke2)

- 401 AUTH_TOKEN_EXPIRED cascade — api() re-authenticates once on expiry (never on login paths) and
  retries the same request with its original idempotency key.
- The port-squat preflight takes the legitimate project list per fixture: the -official instance was
  rejecting its own parent containers; the claim wait accepts a settled task.
- Correction 6 also exposed the restarted agent never claiming the re-armed stage; the claim wait
  separated that failure mode from a bare success timeout.

## Correction 8/9 (threads 4721373969 + 4721691577)

- Exclusion is exact compose label only — sibling-prefixed (smoke vs smoke-old), label-less and foreign
  containers holding recorded ports all fail loudly.
- The preflight now precedes login (squatters hid behind login failures).

## Correction 10/11 (run-3 chain, PR #387 thread 4722371822)

- The restarted agent never claimed the re-armed stage and every later agent-dependent scenario failed
  behind it; probes bounce the agent once for the claim and clean up their own failures (best-effort
  API cancel + fixture-level SQL terminalization of the stuck task and its pending or unknown stages).
- A claimed replay that never reached success also skipped cleanup; the timeout failure path calls
  cleanup_recovery_task too. Regressions assert cleanup on failed outcomes, none on success.
- Open question: WHY the restarted agent process stops claiming is still not visible in fixture
  evidence (needs agent-process logs on the next live run).
