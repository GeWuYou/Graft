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
  events. Runtime: reconcileInterruptedRecovery runs on the 30s convergence ticker; per-task dedupe,
  conflicts skipped as lost races.
- Decisions: auto path resumes only the runtime-recorded interruption signature
  (failure_code=external_execution_lease_expired); manual_reconcile and explicit needs_attention
  verdicts are never auto-replayed. Consumer retry_if_idempotent adoption is deferred to the owning
  modules per the product roadmap (all current consumers submit manual_reconcile MaxAttempts=1).
- Harness: compose-material-stage-code joins LOCAL_REQUIRED_SCENARIOS; three agent recovery scenarios
  join a new AGENT_REQUIRED_SCENARIOS (local fixtures forbid the agent service). Recovery scenarios use
  a documented store stand-in (RECOVERY_POLICY_STAND_IN_NOTE) solely to exercise the runtime policy path.
- Primary-checkout follow-up (needs Docker + a registered task fixture): run validate_docker_isolated.py
  init/up/probe with --execution-mode agent per tests/conformance/docker-runtime-agent/README.md.

## 2026-11-21 Correction History (PR #380)

- Correction 1 (CI run 44777511166): local settlements deliver the material stage attribution as
  failure_code=local_material_resolve_failed with failure_stage=NULL; the probe asserts each owner's
  delivered shape (agent projects failure_stage=material_resolve).
- Correction 2: admission probe diverges the capability binding version so fresh claims are rejected
  with telemetry intact; agent expired journals settle the delivered interruption receipt and clear;
  whole-task resume refuses with ErrRecoveryBlocked while any unknown stage is over budget.

## 2026-11-21 Correction 3 (threads 4645868796 + 4645868804 + lint)

- Candidate starvation (4645868796): blocked tasks' resumable legs matched every pass and, under the
  oldest-first limit of 100, starved newer recoverable tasks. The candidate query now excludes resumable
  legs whose task still holds any unknown over-budget stage; exhausted converge-once legs vacate after
  one pass, so every candidate slot makes bounded progress. Regression: 100 blocked tasks + 1 newer
  recoverable task -> the newer leg is the only candidate and the reconcile resumes it.
- Cancel stranding (4645868804): a cancellation recorded between candidate selection and the resume
  transaction was overwritten (task reopened running with only a pending external stage). The resume
  transaction now re-checks cancel_requested_at and its guarded UPDATE rejects recorded cancellations,
  so the resume rolls back untouched and the cancel path completes; manual retry shares the protection.
- Lint: ResumeInterruptedRecovery split into begin/guard/resume-stage helpers, all functions in
  recovery_repository.go under cyclop 10; both golangci configs 0 issues on the moved merge-base.
- Evidence: go test modules/task + agent internal ok; validate backend 0; harness unittest 45 ok;
  git diff --check clean. Agent-mode scenarios remain live CI verification.
