# High Frequency Log Governance Tracking

## Work Contract

```yaml
version: v1
kind: refactor
scope: long-running
authority_summary: server/internal/logger owns runtime logging; server/internal/config owns startup configuration; App Log, OpenAPI, and web are downstream consumers.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - ai-plan/public/high-frequency-log-governance/README.md
    - ai-plan/public/high-frequency-log-governance/startup-prompt.md
    - ai-plan/public/high-frequency-log-governance/todos/high-frequency-log-governance-tracking.md
    - ai-plan/public/high-frequency-log-governance/traces/high-frequency-log-governance-trace.md
closeout:
  archive: true
  lessons_review: true
```

## Loop State

- loop_mode: `topic-completion-loop`
- completed_batches:
  - `batch-1-logger-category-foundation`
  - `batch-2-high-frequency-migration-and-static-governance`
  - `batch-3-app-log-category-contract`
  - `batch-4-category-authority-repair`
- current_batch: `batch-4-category-authority-repair`
- pending_batches:
  - `archive-readiness-evaluation`
- next_batch: `archive-readiness-evaluation`

## Acceptance Conditions

- TRACE is available without changing existing `*zap.Logger` injection.
- Categories are typed constants in a logger-owned registry and config accepts no business-code string literals.
- Disabled Category prevents lazy field creation, encoding, serialization, and durable persistence.
- High-frequency normal diagnostics use TRACE; periodic failures remain visible and bounded.
- App Log category persistence, query contract, default gate semantics, and web consumer behavior are validated with the final Batch 4 change.

## Current Risks

- TRACE requires a custom zap level and encoder handling; compatibility must be covered by observer tests.
- Archive readiness still requires loop-owner acceptance review and scoped commit confirmation.
- Category literal static checking must remain bounded to production server code and must not become a whole-repository linter.

## 2026-07-16 Batch 2 Receipt

- Migrated Docker CPU calculation diagnostics to `CategoryDockerStats` TRACE with an explicit `Enabled(TRACE)` guard and
  `TraceLazy`; calculation fields are not built while the category is disabled.
- Migrated identified collector, runtime event, project topic stream, monitor trend, and system-config cache failure or
  debug paths to typed categories while preserving their existing severity.
- Moved user Ent debug output to `CategoryDatabaseEnt` TRACE and guard argument formatting behind category enablement.
- Added `scripts/check_log_category_governance.py`, limited to handwritten production Go under `server`, and wired it
  into the existing backend lint stage. The guard rejects `logger.Category(..., "literal")` but permits logger typed
  constants.

## 2026-07-16 Batch 3 Receipt

- Added the logger-owned `app_logs.category` durable field, with `runtime.stats` as the registered legacy default and
  explicit SQL backfill/default behavior in forward-only migration `202607160001_app_log_category.sql`.
- Added typed `AppLogger.Category(LogCategory)`. When the selected category is disabled by the existing Zap category
  gate, the call returns before field serialization and the durable queue; ordinary AppLogger calls retain their
  existing output/persistence behavior and persist under the legacy default category.
- Added category filtering to the logger repository, Explorer binding, saved-view validation, OpenAPI source and
  generated contracts, plus App Log list filter, column, and detail presentation. TRACE remains process-output-only.
- Focused Go and Vitest coverage, SQL validation, OpenAPI generation/freshness, web i18n/type generation checks, and
  final backend/web completion validation are required before archive readiness is accepted.

## 2026-07-16 Batch 4 Receipt

- Added `CategoryApplication` as the logger Registry default for low-frequency general application logs. Every
  AppLogger instance binds its effective category before message sanitization, field serialization, Zap output, and
  durable enqueue; disabling `application` or an explicit selected category suppresses all of those write paths.
- Added forward-only migration `202607160002_app_log_application_category.sql`, which changes Batch 3's incorrect
  `runtime.stats` legacy default and rows to `application`. The bounded compatibility assumption is documented: no
  source metadata can distinguish an explicit historical runtime-stat record from a Batch 3 default row.
- Removed category inventories from OpenAPI and the App Log web module. The server validates arbitrary category text
  against its Registry; web preserves category route and saved-view values until that server validation occurs.
- Extended the production-Go scanner to reject literal `logger.Category`, `logger.WithCategory`, and qualified
  `logger.LogCategory("...")` forms without widening into unrelated `.Category` APIs.
- Archive readiness remains the next loop batch after the scoped commit result. Its first gate is the reproducible
  out-of-scope web failure in `project/pages/configuration-workspace/index.test.ts`; Batch 4's App Log-focused tests,
  typecheck, generated-contract freshness, and backend completion entrypoint pass.
