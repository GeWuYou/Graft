# High Frequency Log Governance Trace

## 2026-07-16 Intake

- Confirmed a new long-running refactor with no active topic owner.
- Work Contract selected an active topic and `graft-multi-agent-loop`; no ADR, separate design, or separate roadmap is required because the user supplied the final architecture decisions.
- Locked the runtime design: typed category constants and registry, thin facade over existing `*zap.Logger`, TRACE for normal high-frequency diagnostics, aggregate category configuration, and bounded static governance.
- Initial batch sequence is logger foundation, high-frequency migration, then App Log/OpenAPI/web contract evaluation.

## 2026-07-16 Batch 1: logger category foundation

- Added the logger-owned typed category registry, strict aggregate `GRAFT_LOG_CATEGORIES` parsing, and a thin
  `CategoryLogger` facade without changing `*zap.Logger` DI.
- Added process-output-only TRACE below DEBUG, including console and JSON encoder support. Category TRACE defaults off;
  false category rules suppress every level in the Zap core path.
- Added focused registry/config/category-core/lazy tests and aligned deployment, architecture, audit logging, logger
  README, public topic, and Compose documentation.
- Next batch owns governed high-frequency call-site migration and static literal coverage; App Log/OpenAPI/web remain
  downstream and unchanged.

## 2026-07-16 Batch 2: high-frequency migration and static governance

- Migrated the normal Docker CPU calculation diagnostic to `CategoryDockerStats` TRACE with the required explicit
  `Enabled(TRACE)` guard and `TraceLazy` field construction.
- Preserved WARN for collector, event-manager, project stream, monitor trend, and system-config cache failures while
  categorizing them with typed logger constants; no normal per-tick or cache-hit logs were added.
- Moved user Ent debug output to `CategoryDatabaseEnt` TRACE and prevented Ent argument formatting when disabled.
- Added the bounded production-Go `logger.Category` literal guard and attached it to the existing backend lint stage.
- Batch 3 owns App Log persistence, OpenAPI, and web-consumer contract evaluation; those downstream surfaces remain
  unchanged in this batch.

## 2026-07-16 Batch 3: App Log category contract

- Made category a logger-owned persisted `app_logs` field. Existing durable rows and uncategorized AppLogger calls use
  the registered `runtime.stats` default; category remains distinct from the explicit component path.
- Added typed `AppLogger.Category(LogCategory)`, with the existing category core gate checked before sanitization,
  serialization, and durable queue submission. TRACE remains process-output-only and is not added to App Log severity.
- Extended the canonical OpenAPI source and regenerated the bundled spec, backend bindings, runtime docs asset, and
  web schema. The Explorer repository/query binding, saved views, filter state, list column, and detail panel now
  consume the same server-owned category value.
- Added migration `202607160001_app_log_category.sql` and its category/time/id Explorer index. Batch completion now
  awaits loop-owner archive-readiness evaluation and the scoped commit result.

## 2026-07-16 Batch 4: category authority repair

- Restored `server/internal/logger` as the only category inventory authority by registering `application` as the
  default AppLogger category and binding it before all write-side sanitization, serialization, Zap, and persistence
  work. `application` is intentionally distinct from high-frequency `runtime.stats`.
- Added the logger-owned forward migration that reclassifies Batch 3's legacy default and updates the column default.
  This is a bounded compatibility repair: historical rows do not record whether `runtime.stats` was explicit, so the
  migration treats those Batch 3 values as the incorrect default.
- Reduced OpenAPI to wire-shape authority and made web category filtering text-based, preserving arbitrary deep links
  and saved-view values for server Registry validation. No downstream category array/type guard remains.
- Hardened the bounded Go scanner for the logger APIs named by policy, added bypass regression coverage, and queued
  archive-readiness as the next loop step after final validation and commit.

## 2026-07-16 Archive readiness

- Confirmed the category Registry is the only inventory authority, production Go category calls are typed, and the bounded scanner rejects the governed literal bypass forms.
- Confirmed disabled categories return before lazy fields, sanitization, serialization, Zap encoding, and durable App Log enqueue; normal high-frequency diagnostics are TRACE while failure logs retain WARN or ERROR.
- `git diff --check`, `python3 scripts/validate_sql_migrations.py`, `python3 scripts/validate_ai_plan_structure.py`, `python3 scripts/validate_ai_governance.py`, and `cd server && go run ./cmd/graft validate backend` passed.
- `cd web && bun run check` completed its static gates and App Log tests but failed in the unrelated Project Configuration Workspace test at `src/modules/project/pages/configuration-workspace/index.test.ts:1457`. This archived topic does not claim a full green Web suite.
- Topic reached `archive-ready`; the active-topic router no longer lists it.
