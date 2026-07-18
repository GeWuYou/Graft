# Backend Error Logging Governance Trace

## 2026-07-18 Batch 1 Foundation

- Work Intake classified the request as a long-running refactor requiring existing design-authority updates and an active topic, without a roadmap or ADR.
- Authority discovery found that Access Log does not swallow errors; missing cause logs, ERROR-level access outcomes, caller misattribution, and `logger -> httpx` correlation coupling jointly obscure failures.
- Batch 1 keeps business-module migration out of scope except compile-required context adaptations.

## Locked Decisions

- Move request-context storage to a neutral server core boundary so logger and httpx can share it without an import cycle.
- Construct one runtime AppLogger and inject it directly instead of resolving or rebuilding module-local instances.
- Preserve the existing HTTP error envelope and key-first message contract.

## 2026-07-19 Batch 1 Closeout

- Added the HTTP-neutral `server/internal/requestctx` contract and changed `AppLogger` to depend on it directly; `httpx` retains its internal forwarding API so existing module callers keep the same context semantics.
- Added the typed `apperror` contract, `logger.ReportError`, caller attribution fixes, global correlation middleware, safe AppError mapping, INFO-only Access Log output, and correlated panic recovery.
- Verified focused foundation packages, the full server test suite, backend validation, ai-plan structure, shared asset registry, and whitespace checks.
- Batch 1 is ready for scoped commit; Batch 2 owns the Project/Task priority failure-chain migration.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "Batch 1 - authority, topic bootstrap, and core error/logging foundation"
  ],
  "pending_batches": [
    "Batch 2 - Project/Task priority request failure chain",
    "Batch 3 - remaining synchronous HTTP modules",
    "Batch 4 - asynchronous recovery, cause integrity, and governance guard",
    "Batch 5 - full audit, validation, and archive readiness"
  ],
  "current_batch": "Batch 2 - Project/Task priority request failure chain",
  "next_batch": "Batch 2 - Project/Task priority request failure chain",
  "closeout_status": "batch-1-complete"
}
```
