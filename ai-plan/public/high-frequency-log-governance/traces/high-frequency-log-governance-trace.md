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
