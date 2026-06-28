# Access Log Explorer Authority

## 1. Goal

This document defines the canonical authority for future `Access Log Explorer` work in `Graft`.

This document began as contract-definition evidence.

Current implementation status:

- access-log OpenAPI list/detail contracts are implemented under `openapi/**`
- HTTP handlers, repository query methods, retention jobs, and runtime registration are implemented under `server/internal/httpx/**`
- the frontend explorer is implemented under `web/src/modules/access-log/**`
- app-log explorer, metrics, tracing, OpenTelemetry expansion, and audit/security authority changes remain outside this surface

## 2. Authority Summary

- runtime request-fact semantics
  - `server/internal/httpx/**`
- current durable access-log storage shape
  - `server/internal/httpx/**`
- shared explorer wire contract
  - `openapi/**`
- generated consumers
  - `server/internal/contract/openapi/**`
  - `web/src/contracts/openapi/generated/**`
- frontend explorer consumer
  - `web/src/modules/access-log/**`

`web` is a downstream consumer only.

`server/modules/audit/**` and `server/modules/monitor/**` may consume access-log correlation but do not own access-log explorer truth.

## 3. Consumer Analysis

### 3.1 In-scope explorer consumers

| Consumer | Primary use | Belongs to Access Log Explorer | Reason |
| --- | --- | --- | --- |
| Operations | request lookup, failure triage, route traffic spot checks | yes | request-fact timeline is the canonical surface |
| Troubleshooting | correlate one request across request/auth/audit surfaces | yes | `request_id`, route, status, duration, and user correlation are request-fact semantics |
| Security investigation | investigate one suspicious request or authz failure by request facts | partial | request facts belong here; risk classification and incident truth do not |
| Audit correlation | jump from audit record to bounded request facts | partial | correlation entry is allowed; audit evidence truth stays in audit |
| Future observability modules | consume canonical request-fact contract | partial | may use explorer contract as consumer, but not redefine it |

### 3.2 Out-of-scope consumer intent

| Use-case | Belongs elsewhere | Reason |
| --- | --- | --- |
| business action investigation | `Audit` | action/resource/result/risk semantics are audit-owned |
| anomaly detection, trends, incident health | `Monitor` | anomaly/trend/incident semantics are monitor-owned |
| app runtime exception search | future app-log explorer | different authority chain under `server/internal/logger/**` |
| retention, archive, purge policy | future retention authority topic | explorer consumes retained data but does not own lifecycle policy |
| distributed tracing drilldown | future tracing topic | no independent tracing authority exists in MVP |

## 4. Explorer Scope Definition

`Access Log Explorer` is the operator-facing read surface for canonical HTTP request facts.

It owns:

- request-fact filtering on canonical access-log fields
- request-fact sorting on approved fields
- timeline-oriented pagination semantics
- bounded detail inspection for one request row
- correlation jumps using canonical fields only

It must not own:

- business intent interpretation
- audit result taxonomy
- security incident classification
- monitor anomaly/trend semantics
- retention policy
- frontend-origin navigation semantics as backend contract

## 5. Query Authority Matrix

### 5.1 Canonical query fields

| Field | Filterable | Sortable | Searchable | Display-only | Forbidden | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `request_id` | exact | no | no | no | no | highest-confidence correlation lookup |
| `trace_id` | no separate field | no | no | alias-only | yes as separate authority | MVP `trace_id=request_id`; explorer must not create a second backend field |
| `keyword` | yes | no | bounded canonical search | no | no | backend-owned bounded search across `request_id`、`path`、`username` |
| `user_id` | exact | no | no | no | no | stable authenticated correlation |
| `username` | exact | no | bounded contains/prefix not approved | no | no | exact match only in canonical MVP contract |
| `method` | exact or bounded set | no | no | no | no | canonical HTTP transport fact |
| `path` | exact or canonical prefix | no | no | no | no | backend-owned path semantics only |
| `route` | exact | no | no | no | no | route template is more stable than raw path for grouped troubleshooting |
| `status_code` | exact or bounded set | yes | no | no | no | supports operator filtering and secondary sort |
| `status_group` | bounded set | no | no | no | no | approved grouped status filter for operator presets; current values are `4xx` and `5xx` |
| `duration_ms` | range | yes | no | no | no | range filter is canonical; free-form fuzzy search is not |
| `started_at` / time range | bounded range | yes | no | no | no | canonical request timeline authority anchor |
| `occurred_at` | bounded secondary range | yes | no | no | no | completed-time refinement only |
| `client_ip` | not approved in this topic | no | no | yes | no | sensitive/high-cardinality; keep display-only until explicit authz and privacy decision |
| `user_agent` | not approved in this topic | no | no | yes | no | noisy/high-cardinality; display-only |
| `request_size` | not approved in this topic | no | no | yes | no | presentational until real operator need is approved |
| `response_size` | not approved in this topic | no | no | yes | no | presentational until real operator need is approved |

### 5.2 Query contract rules

- `request_id`
  - exact match only
- `user_id`
  - exact match only
- `username`
  - exact match only
- `method`
  - exact match or bounded set match
- `path`
  - exact match or canonical prefix match
- `route`
  - exact match only against backend route template
- `status_code`
  - exact match or bounded set match
- `duration_ms`
  - inclusive numeric range
- `started_from` / `started_to`
  - inclusive canonical time range on `started_at`
- `occurred_from` / `occurred_to`
  - inclusive secondary time range on `occurred_at`

Forbidden query surfaces:

- arbitrary JSON filter blobs
- metadata key search
- audit-only fields such as `action`, `resource_type`, `risk_level`, `result`
- monitor-only fields such as anomaly key, incident seed, trend dimension
- frontend route/origin/preset semantics as backend query fields

Clarification:

- frontend route/query authority must not expose `trace_id` as a canonical explorer filter
- backend may still retain a bounded `trace_id` storage/filter capability for direct API or internal troubleshooting, but that capability is not part of the `web` canonical query truth

## 6. Sort Authority Matrix

| Field | Allowed | Default | Reason |
| --- | --- | --- | --- |
| `started_at` | yes | yes, `desc` | canonical request-start timeline |
| `occurred_at` | yes, as secondary filter | yes | completed-time refinement |
| `duration_ms` | yes | no | useful for slow-request triage |
| `status_code` | yes | no | useful for error clustering within a bounded window |
| `request_id` | no | no | exact lookup field, not meaningful list ordering |
| `method` | no | no | low-value ordering for operator workflow |
| `path` | no | no | unstable/high-cardinality ordering |
| `route` | no | no | grouped filtering is useful; sorting is not canonical |
| `username` | no | no | unstable/high-cardinality and privacy-adjacent |
| `client_ip` | no | no | privacy/high-cardinality concern |
| `user_agent` | no | no | noisy and not operationally stable |

Sort rules:

- default sort is `started_at desc`
- canonical wire shape uses repeated `sort=field:dir` params
- sort param order is the priority order
- secondary stable tie-break should remain backend-owned
- unsupported sort fields must be rejected, not ignored silently

## 7. Pagination Authority

Chosen canonical model:

- `page`
- `page_size`

Recommended bounds:

- `page >= 1`
- `1 <= page_size <= 100`

Rationale:

- matches current admin list/explorer patterns already used by audit
- aligns with the current access-log storage/index shape, which is timeline-first and not yet contract-approved for opaque cursor semantics
- keeps pagination semantics backend-owned and explicit
- avoids forcing `web` to invent cursor state, resume tokens, or cursor invalidation behavior

Rejected for now:

- cursor pagination

Reason:

- current topic does not approve a cursor contract, stable cursor encoding, or cursor invalidation semantics
- storage and retention policy are still evolving, so cursor semantics would over-specify a lower layer too early

## 8. Retention Boundary

Explorer assumptions:

- explorer reads from whatever access-log records currently exist under runtime/storage authority
- explorer may show bounded time filtering on `occurred_at`
- explorer may rely on records disappearing outside the retained dataset without treating that as a contract violation

Explorer does not own:

- retention duration
- purge cadence
- archive semantics
- legal/compliance preservation rules
- “data should still exist” guarantees beyond the currently retained dataset

Truthful current status:

- current retention policy remains intentionally undefined
- explorer must not invent minimum history guarantees such as “7 days”, “30 days”, or “forever”

## 9. Correlation Boundary

### 9.1 Canonical relationships

| Surface | Relationship to Access Log Explorer | Ownership boundary |
| --- | --- | --- |
| Audit Log | may correlate by `request_id`, actor fields, and bounded time window | audit owns action/resource/result/risk/evidence truth |
| Security Event | may originate from the same request and later persist through audit path | access-log explorer may expose correlation entry only; security classification stays outside |
| Monitor Incident | may link into audit and then into request facts | monitor owns anomaly/incident/trend semantics |

### 9.2 EvidenceLink opportunity

Future opportunity only:

- `Access Log Explorer` detail view may later expose canonical drilldowns through a bounded evidence-link contract

Not approved now:

- new `EvidenceLink.target_kind`
- new incident-seed semantics
- direct monitor-owned link authority inside access-log explorer
- access-log-owned inference of audit incident truth

## 10. Frontend Consumption Boundary

Future `web` explorer may own:

- route query sync for approved canonical filters
- visible filter arrangement and grouping
- preset chips or quick-entry shortcuts as UI-only context
- drawer open/close state
- table column visibility preferences

Future `web` explorer must not own:

- backend query semantics
- aliasing `trace_id` into a second backend field
- client-only keyword search presented as canonical dataset search
- unsupported sort fallback behavior
- fabricated correlation to audit/monitor records

## 11. UX Recommendation

Preferred page type:

- `list-form-detail`

Recommended UX pattern:

- management page header
- dedicated filter bar above the table
- table-first explorer body
- right-side detail drawer for one selected request
- optional quick presets as frontend-only shortcuts that compile into canonical backend filters

Recommended filters in the first explorer iteration:

- `request_id`
- `user_id`
- `username`
- `method`
- `route`
- `status_code`
- `duration_ms` range
- `occurred_at` range

Recommended table columns in the first explorer iteration:

- `occurred_at`
- `method`
- `path`
- `route`
- `status_code`
- `duration_ms`
- `user`
- `request_id`

Recommended detail drawer contents:

- canonical request facts only
- correlation jump actions based on `request_id`, user fields, and bounded time windows
- no business/audit interpretation copy inside the access-log-owned detail itself

Rejected UX patterns for canonical v1:

- dashboard-first access-log page
- tag-builder-only query model as the primary surface
- frontend-owned local pagination over a partial dataset
- embedding incident or anomaly read models directly into the access-log table

UX rationale from current repo patterns:

- `Audit Logs` already uses list filters + server pagination + detail drawer
- `User` and `RBAC` list pages use the same `list-form-detail` shell for dense operator work
- overview dashboards act as shortcut entry surfaces, not as the canonical explorer itself

## 12. Ownership Matrix

| Layer | Owner | Responsibility |
| --- | --- | --- |
| runtime logging semantics | `server/internal/httpx/**` | request-fact capture and normalization |
| explorer query/sort/pagination contract | `server/internal/httpx/**` + `openapi/**` | canonical exploration semantics and wire shape |
| shared HTTP wire contract | `openapi/**` | approved API contract |
| generated server artifacts | `server/internal/contract/openapi/**` | derived consumer |
| generated web artifacts | `web/src/contracts/openapi/generated/**` | derived consumer |
| web explorer module | `web/src/modules/access-log/**` | downstream consumer and UI-only navigation context |

## 13. Unresolved Decisions

- whether `client_ip` can ever become filterable under future authz/privacy governance
- whether `username` should remain exact-match-only or later support bounded prefix search
- whether `request_size` and `response_size` deserve filter/sort authority after real operator evidence exists
- whether future quick presets should be standardized in module contract or remain page-local UI context
- what explicit permission code will guard explorer access

## 14. Recommended Next Topic

- `phase-d-access-log-explorer-implementation`

Entry condition for that topic:

- treat this document as the canonical explorer authority
- implement backend and frontend only within these bounded semantics
- do not widen into app-log explorer, retention policy invention, metrics, or tracing
