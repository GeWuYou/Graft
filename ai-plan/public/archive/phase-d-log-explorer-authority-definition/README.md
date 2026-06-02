# Phase D Log Explorer Authority Definition

## Status

- Topic: `phase-d-log-explorer-authority-definition`
- Status: `archived`
- Task class: `cross-boundary`
- Recovery source: `none`

## Goal

本 topic 只定义 `Log Explorer` 的 authority model。

本轮禁止：

- 实现日志页面
- 新增 API
- 修改运行时代码
- 修改 OpenAPI 运行时契约
- 引入 Metrics / Tracing / OpenTelemetry

本轮必须回答：

- 未来 `Log Explorer` 应建立在什么 authority 之上
- `Audit` 与 `Log Explorer` 的边界如何划分
- retention authority 目前是否存在
- future runtime Phase D 还缺什么

## Authority Summary

- `server/internal/logger/**`
  - current `AppLogger` / `Error Log` authority
- `server/internal/httpx/**`
  - current request correlation, `Access Log`, and `Security Event` bridge authority
- `server/internal/audit/**` + `server/modules/audit/**`
  - current `Audit Event` / persisted `Security Event` authority
- `server/modules/monitor/**`
  - monitor anomaly, evidence link, and short-retention trend authority
- `openapi/**`
  - future shared wire contract authority once a later runtime Phase D topic explicitly approves log-explorer APIs
- `web`
  - downstream consumer only

## Deliverables

### Log Ownership Matrix

| Surface | Owner | Authority | Storage | Retention | Consumer | Operator |
| --- | --- | --- | --- | --- | --- | --- |
| AppLogger / App Log | `server/internal/logger/**` | runtime app logging baseline | current process logger output only | undefined | developers, future log explorer | operator may inspect in future app-log explorer |
| AccessLogger / Access Log | `server/internal/httpx/**` | request fact logging baseline | current structured logger output only | undefined | operators, future log explorer | request troubleshooting |
| AuditRecorder / Audit Log | `server/internal/audit/**` + `server/modules/audit/**` | audit evidence truth | `audit` module persistence | undefined | audit module, compliance/operator workflow | security/investigation workflow |
| SecurityEvent | `server/internal/httpx/**` publish, `audit` path persist | security evidence truth | audit persistence path | inherits audit retention until separately governed | audit incident, security workflow | operator investigation |
| MetricsEmitter | none | no canonical runtime authority | n/a | n/a | none beyond bounded monitor payloads | not available |

### Retention Matrix

| Surface | Current authority | Current definition | Verdict |
| --- | --- | --- | --- |
| Audit Log | none | no formal retention rule found in current design or runtime authority | governance gap |
| Access Log | none | no formal retention rule found in current design or runtime authority | governance gap |
| App Log | none | no formal retention rule found in current design or runtime authority | governance gap |
| Monitor trend evidence | `server/modules/monitor/**` | 1h bounded evidence window, 2h storage TTL | defined but monitor-only |

### Contract Ownership Matrix

| Contract surface | Authority owner | Generated artifact | Downstream consumer |
| --- | --- | --- | --- |
| App log semantics | `server/internal/logger/**` | none yet | future log-explorer backend/web consumers |
| Access log semantics | `server/internal/httpx/**` | none yet | future log-explorer backend/web consumers |
| Security event ingest/publish semantics | `server/internal/httpx/**` | `pluginapi.AuditEvent` event payload consumption | `server/internal/audit/**`, `server/modules/audit/**` |
| Audit log / incident read model | `server/modules/audit/**` | OpenAPI generated audit schema | `web/src/modules/audit/**` |
| EvidenceLink shared drilldown | backend authority + `openapi/**` | generated OpenAPI artifacts | `web/src/modules/audit/**`, `web/src/modules/monitor/**` |
| Future Log Explorer HTTP contract | `openapi/**` after explicit approval | `server/internal/contract/openapi/**`, `web/src/contracts/openapi/generated/**` | future `web` log-explorer module |

### Investigation Workflow

Canonical Investigation Workflow:

1. Start from one seed: `requestId`, `traceId`, `audit event id`, `user`, or `ip`.
2. If the seed is an audit incident or security event, `audit` owns the investigation root.
3. From audit, operators may jump to related access/app logs only through canonical correlation fields and bounded time windows.
4. If the seed is an access log, logging authority owns request fact truth; audit is only a linked evidence consumer when a matching audit record exists.
5. If the seed is an application log, logging authority owns runtime fact truth; operators may follow request correlation into access logs, then into audit/security evidence when present.
6. If the seed is monitor evidence, monitor may link into audit through `EvidenceLink`, but monitor does not own log-explorer truth.

### Governance Gap List

- `Audit Log` retention authority is undefined.
- `Access Log` retention authority is undefined.
- `App Log` retention authority is undefined.
- No approved runtime storage authority yet exists for future access/app log exploration.
- No approved OpenAPI contract yet exists for future log-explorer APIs.
- Current system still lacks a canonical runtime `MetricsEmitter`; metrics must not be smuggled into Phase D as log-explorer scope.

### Recommended Runtime Phase D Plan

1. Define retention authority first:
   - choose canonical owner for audit/access/app retention policy
   - document purge/archive lifecycle before implementation
2. Define runtime storage authority for access/app logs:
   - decide whether Phase D uses current logger outputs only, a structured sink, or another explicit authority path
3. Define minimal shared contract in `openapi/**`:
   - only after storage and retention authority are explicit
4. Implement `Access Log Explorer` first:
   - request facts are already the clearest current authority
5. Implement `App Log Explorer` second:
   - only after app-log storage/query semantics are explicit
6. Add bounded `Audit -> Log` and `Log -> Audit` investigation navigation:
   - no mixed storage, no fake unified dataset

## Final Verdict

- Verdict: `Archive Ready`

Basis:

- `Audit Domain` vs `Log Explorer Domain` boundary is now explicit.
- canonical owner chain for future log explorer work is explicit.
- retention is truthfully marked as a governance gap rather than invented.
- operator investigation workflow is explicit enough to constrain later runtime work.

## Next Runtime Topic

`Re-run startup preflight from root AGENTS.md. Governance source: root AGENTS.md. Task class: cross-boundary. Recovery source: parent topic phase-d-log-explorer-authority-definition. Owned scope: server/internal/logger/**, server/internal/httpx/**, openapi/**, web future log-explorer consumer boundary, and related ai-plan governance docs only as needed. Start new bounded topic: phase-d-log-explorer-runtime-authority-repair-and-minimal-access-log-explorer. First repair retention authority and runtime storage authority before adding APIs or pages.` 
