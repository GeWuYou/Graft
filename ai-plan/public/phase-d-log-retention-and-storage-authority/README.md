# Phase D Runtime Retention And Storage Authority

## Status

- Topic: `phase-d-log-retention-and-storage-authority`
- Status: `archive-ready`
- Task class: `server`
- Recovery source: `parent topic`
  - `phase-d-log-explorer-authority-definition`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/logger/**` owns app-log runtime semantics
  - `server/internal/httpx/**` owns access-log semantics and security-event publish bridge
  - `server/internal/audit/**` + `server/modules/audit/**` own audit/security durable persistence
  - no upstream `web` authority is required for this bounded runtime-governance topic

## Goal

本 topic 只修复 runtime retention / storage / cleanup authority。

本轮禁止：

- 开发 `Log Explorer` 页面
- 新增日志查询 API
- 扩展 metrics / tracing / OpenTelemetry
- 把 access/app log 强行落到新的未批准存储

## Runtime Logging Matrix

| Surface | Write path | Current storage | Lifecycle | Owner |
| --- | --- | --- | --- | --- |
| AppLogger | shared `*zap.Logger` direct write | process logger output only | core runtime init / close | `server/internal/logger/**` |
| AccessLogger | `httpx` middleware direct write | process logger output only | HTTP middleware | `server/internal/httpx/**` |
| AuditRecorder | request middleware or eventbus candidate -> recorder -> repository | PostgreSQL `audit_logs` | audit module + shared DB runtime | `server/internal/audit/**` + `server/modules/audit/**` |
| SecurityEvent | auth/authz guard publish -> eventbus -> audit recorder | persisted into `audit_logs` when policy includes it | publish in `httpx`, persist in audit plugin | publish: `server/internal/httpx/**`; persistence: audit path |

## Retention Matrix

| Surface | Retention authority | Retention decision | Cleanup owner |
| --- | --- | --- | --- |
| Audit Log | recommended: `server/modules/audit/**` | must be explicitly defined in a later audit-owned runtime slice; current value is undefined | future audit-owned scheduler job |
| Access Log | none today | no repository retention authority while storage is only process output | none |
| Application Log | none today | no repository retention authority while storage is only process output | none |
| Security Event | inherits audit persistence line | same as audit log while persisted in `audit_logs` | same as audit cleanup owner |

## Storage Authority Matrix

| Candidate storage | Surface | Verdict | Notes |
| --- | --- | --- | --- |
| PostgreSQL `audit_logs` | audit log / persisted security event | allowed current authority | already canonical durable store |
| process logger output | app log | allowed current authority | not queryable durable storage |
| process logger output | access log | allowed current authority | not queryable durable storage |
| PostgreSQL for access/app logs | access/app log | forbidden-now | no approved schema, lifecycle, or contract |
| Redis for access/app logs | access/app log | forbidden-now | current Redis authority is monitor-only short retention |
| external log platform | app/access log | future observability | outside MVP |

## Cleanup Ownership Matrix

| Operation | Surface | Authority |
| --- | --- | --- |
| retention cleanup | audit log | future `server/modules/audit/**` |
| archive | audit log | none in MVP |
| purge | audit log | future `server/modules/audit/**` after retention rule approval |
| retention cleanup | access log | none |
| archive | access log | none |
| purge | access log | none |
| retention cleanup | app log | none |
| archive | app log | none |
| purge | app log | none |

## Governance Gap List

- audit retention window is still undefined
- no audit cleanup executor exists
- no archive authority exists for any log surface
- app/access logs have semantics authority but no durable storage authority
- runtime config has log level only; it does not define log sink retention/storage policy
- any future log-explorer contract remains blocked until storage and retention authority are explicit

## Recommended Next Topic

`phase-d-access-log-contract-definition`

Constraint:

- only after the next topic keeps access log contract bounded to currently truthful process-output authority and does not backdoor a new storage platform

## Final Verdict

- Verdict: `Archive Ready`

Basis:

- current runtime can now answer where each log surface writes
- current runtime can now answer which surfaces have durable storage authority and which do not
- cleanup authority is now explicit: only future audit-owned cleanup is even eligible inside MVP runtime
- app/access logs are truthfully marked `not-ready` for repository-owned retention cleanup until storage authority changes
