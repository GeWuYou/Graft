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

## 2026-07-19 Batch 2 Closeout

- Project now owns lifecycle Task submission failure reporting because it is the first layer with both Task submission cause and application/action context. It uses the Batch 1 AppLogger and ReportError contracts, so the returned error remains cause-preserving and marked as reported.
- The Project route fallback now uses the shared AppError HTTP mapping. This keeps existing mapped Project client errors unchanged while allowing only unreported internal failures to be logged at the HTTP boundary.
- Task request routes no longer include raw causes in `data.error`; typed safe descriptors retain existing 400, 404, and 409 status behavior while unknown failures remain safe 500 responses.
- Regression coverage proves one correlated Project business error with no duplicate HTTP fallback and no Task internal-cause response leak. Focused tests, the full server suite, and backend validation passed.

## 2026-07-19 Batch 3 Closeout

- Migrated the remaining administrative synchronous request boundaries without changing their existing local 4xx or i18n contract mappings.
- Auth/User/Security use `ReportError` where the route still owns sufficient operation semantics; other modules now route unknown failures to `AbortAppError`, which supplies exactly one correlated fallback cause log.
- Removed ordinary raw route ERROR calls from the migrated paths. Runtime-target retains the injected runtime logger only for the HTTP fallback boundary.
- Narrowed the next batch to operational synchronous modules; asynchronous recovery and static governance guard work remain separate.

## 2026-07-19 Batch 4 Closeout

- Container, audit, monitor, and access-log explorer now route unexpected synchronous failures through `ReportError` when the route owner has operational context, or through the existing HTTP fallback when it does not.
- Container and audit regression tests assert that owner reports are marked and do not produce an HTTP duplicate; the access-log explorer regression asserts one fallback error and no internal cause in the response.
- Realtime ticket/auth and dashboard widget paths were reviewed without change: their visible failures are expected client outcomes or are already reported by the widget-load owner. Async recover boundaries remain reserved for Batch 5.

## 2026-07-19 Batch 5 Closeout

- Eventbus, scheduler notifier goroutines, Task Runtime executor/cancellation/worker paths, and dashboard widget recovery now capture the real panic-site stack instead of recording a later or absent stack. Async logs retain their event, task, stage, executor, run, or widget context while expected cancellation behavior remains unchanged.
- Task Runtime uses the module-injected AppLogger for its asynchronous panic owner; normal stage completion and persisted state transitions remain unchanged. Dashboard loader errors now retain their internal cause through `Unwrap` while its public widget error remains unchanged.
- Repaired propagated `%v` and string-rebuild cause-loss paths across handwritten backend production code with `%w` or `errors.Join`. The Batch intentionally left safe user-facing message rendering alone.
- Added `check_error_logging_governance.py`, wired it into the existing backend lint guard, and registered it as a bounded shared validation helper. Its scope is deliberately static and narrow; runtime tests/review remain authoritative for exact-once semantics.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "Batch 1 - authority, topic bootstrap, and core error/logging foundation",
    "Batch 2 - Project/Task priority request failure chain",
    "Batch 3 - remaining synchronous administrative HTTP modules",
    "Batch 4 - operational synchronous HTTP modules (container, audit, monitor, realtime, dashboard, httpx explorer)",
    "Batch 5 - asynchronous recovery, cause integrity, and logging governance guard"
  ],
  "pending_batches": [
    "Batch 6 - full audit, validation, and archive readiness"
  ],
  "current_batch": "Batch 5 - asynchronous recovery, cause integrity, and logging governance guard",
  "next_batch": "Batch 6 - full audit, validation, and archive readiness",
  "closeout_status": "batch-5-complete"
}
```

## 2026-07-19 Batch 6 Closeout

- Final production-Go audit repaired the remaining direct internal-error response paths that still discarded causes before the shared HTTP fallback. Shared authorization, App Log Explorer, realtime subscription issuance, and container/project authorization now preserve causes and write one correlated internal error record while retaining the existing safe response envelope.
- The final search found no remaining handwritten cause-loss wrappers, Access Log ERROR/WARN promotion, raw internal cause serialization, or async panic stack gap in the topic scope. The only direct 500 response sites left are unified panic recovery, which records its panic stack, and OpenAPI rendering, which records through its runtime AppLogger before returning generic text.
- Full backend entrypoint, full Go tests, Python governance suite, bounded error logging guard, and whitespace checks passed. The topic reached `archive-ready` with no pending batch.
