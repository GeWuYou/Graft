# Observability Governance Closeout

## Status

- Topic: `observability-governance-closeout`
- Status: `archive-ready`
- Loop mode: `topic-completion-loop`
- Task class: `cross-boundary`
- Recovery source: `none`

## Goal

本 topic 只做 Observability Phase C 治理收口：

- 不新增业务功能
- 不扩展 Metrics、Tracing、APM
- 不开始 Phase D Log Explorer
- 将已落地的 Audit、Monitor、EvidenceLink、Plugin Capability、OpenAPI Authority 收敛为正式治理规范
- 判断当前仓库是否达到 Phase C `archive-ready`

## Scope

- `ai-plan/design/**`
- `ai-plan/public/observability-governance-closeout/**`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `openapi/**` 文档引用
- 与治理规范直接相关的文档

## Authority Summary

- `server/plugins/audit/**` owns audit facts, incident read model, and audit analytics
- `server/plugins/monitor/**` owns monitor facts, anomaly, trend, and monitor evidence semantics
- `openapi/**` owns shared HTTP contract authority
- generated code in `server/internal/contract/openapi/**`, `openapi/dist/**`, and `web/src/contracts/openapi/generated/**` are derived artifacts
- `web` consumes authority and may own only UI consumer context

## Maturity Matrix

| Phase | Latest status | Evidence |
| --- | --- | --- |
| Phase A | done | archived `observability-development-governance` and `logging-governance` established logging development standard |
| Phase B | archived as complete design authority | archived `audit-monitor-phase-b-integration` completed authority-discovery and integration design without runtime widening |
| Phase C | partially closed in implementation, not yet archive-ready as governance closeout | runtime/archive evidence exists, but this topic still needs formal closeout governance and readiness judgment |
| Phase D | not started | no current authority proof for log explorer, tracing, or broader logging product scope |
| Phase E | deferred | no approved authority for wider observability platform expansion |

## Governance Assessment

### 1. Authority Ownership

已落实：

- `Audit` owns audit facts, incident read model, and audit analytics
- `OpenAPI` owns shared contract authority
- generated artifacts are explicitly downstream
- `web` governance already states generated/frontend consumers are not authority

部分落实：

- `Monitor` owns monitor facts in code and schemas, but the repository lacked one current formal governance closeout matrix before this topic

未落实：

- no single current observability closeout doc previously declared all authority boundaries together

### 2. EvidenceLink Governance

已落实：

- one shared `EvidenceLink` shape exists across audit DTO, monitor capability mapping, and OpenAPI schema
- frontend target construction is already centralized through shared contract helpers

部分落实：

- monitor pages still choose the first available link as a consumer strategy
- monitor origin query is a frontend navigation context, but this distinction had not been written down formally

未落实：

- audit UI still uses metadata fallback for some display and correlation-derived values

治理结论：

- `EvidenceLink` is a backend-owned canonical drilldown contract
- frontend may consume and route from canonical fields only
- frontend origin/query state is UI-only context, never evidence authority
- metadata fallback remains a governance gap, not an authority source

### 3. Plugin Capability Governance

已形成标准：

- `server/internal/pluginapi/**` is the stable cross-plugin capability boundary
- current auth / user / rbac / monitor capabilities follow narrow interface + DTO patterns

存在特例：

- some observability lineage still exists as mirrored DTOs across `pluginapi`, audit store, and OpenAPI
- event-name style contracts such as `AuditRecordEventName` remain narrow but not yet described as observability capability governance examples

推荐规范：

- capability only exposes stable business ability
- observability capability should be classified as bounded evidence or event-ingest ability
- docs must record canonical owner and mapping lineage when a DTO intentionally appears in multiple derived layers

### 4. Observability Boundary

| Domain | Owns | Consumes | Must not do |
| --- | --- | --- | --- |
| Audit | audit facts, incident read model, audit analytics | monitor evidence links | infer monitor anomaly truth |
| Monitor | monitor facts, anomaly, trend, evidence links | audit incident seed context | infer audit policy or incident truth |
| Logging | app/error/access/security bridge facts | request context and runtime failures | replace audit/monitor business authority |
| Metrics | current monitor trend-like payload only | monitor runtime samples | mix with audit analytics or fake a general metrics backend |
| Tracing | no independent MVP authority | requestId alias semantics only | claim distributed tracing is already present |

### 5. Logging Governance Readiness

Judgment: `Partially Ready`

Evidence:

- `AppLogger` authority is established
- `AccessLogger` authority is established
- `AuditRecorder` and `SecurityEvent -> Audit` path are implemented and documented in archives
- `MetricsEmitter` remains future-boundary only
- no current unified closeout had previously combined these surfaces into one formal Phase D readiness gate

## Governance Gap List

- metadata fallback still exists in audit UI consumer code and remains a governance gap
- observability authority and boundary rules previously depended on archive evidence more than one current closeout document
- plugin capability governance lacked an explicit observability/evidence classification note
- Phase D readiness could not be marked `Ready` because `MetricsEmitter` remains a future boundary

## Recommended Next Topic

If this topic reaches `archive-ready`, the recommended Phase D topic is:

- `phase-d-log-explorer-authority-definition`
  - scope only authority, retention, contract owner, and operator workflow
  - do not begin implementation until canonical owner and data source are approved

If this topic does not reach `archive-ready`, the remaining governance closeout work is:

- finalize and validate the formal closeout docs
- decide whether metadata fallback is acceptable as a documented temporary gap for Phase C archive
- confirm Phase D readiness remains `Partially Ready`

## Current Judgment

- Final judgment: `Archive Ready`
- Basis:
  - authority ownership now has one current formal closeout document instead of relying only on archive evidence
  - EvidenceLink governance, plugin capability governance, and observability boundary rules are now explicitly written into current governance docs
  - remaining issues are documented governance gaps, not unresolved authority ambiguity or hidden feature scope
  - Phase D readiness remains truthfully limited to `Partially Ready`, so this closeout does not overclaim future implementation readiness
