# Access Log Authority Contract

## 1. Goal

This document defines the canonical authority contract for future `Access Log` runtime work in `Graft`.

This document is governance-only.

It does not approve:

- new log pages
- `Access Log Explorer` UI work
- durable access-log storage
- access-log query APIs
- metrics, tracing, OpenTelemetry, or monitor-scope expansion

## 2. Authority Summary

- runtime semantics owner
  - `server/internal/httpx/**`
- shared wire-contract owner for future approved APIs
  - `openapi/**`
- generated artifact consumers
  - `server/internal/contract/openapi/**`
  - `web/src/contracts/openapi/generated/**`
- future frontend consumer
  - future `web/src/modules/<access-log-explorer>/**`

`web` is a downstream consumer only.

`server/plugins/audit/**` is not the canonical owner of `Access Log`.

## 3. Access Log Definition

`Access Log` is the canonical request-fact record for one HTTP request lifecycle at the transport/runtime boundary.

It records:

- request correlation
- transport facts
- route identity
- response status
- request duration
- bounded client context

It does not record:

- business action truth
- compliance evidence truth
- audit policy result truth
- arbitrary application metadata
- security finding taxonomy
- frontend-owned semantics

## 4. Boundary Matrix

| Surface | Canonical owner | Responsible for | Must not own |
| --- | --- | --- | --- |
| Access Log | `server/internal/httpx/**` | request lifecycle, transport facts, latency, status, canonical request correlation | audit action truth, business event truth, security finding truth, frontend query semantics |
| Audit Log / Audit Event | `server/internal/audit/**` + `server/plugins/audit/**` | user action, business event, permission decision, compliance evidence, audit policy outcome | raw request traffic truth, generic request observability, access-log storage authority |
| Security Event | publish in `server/internal/httpx/**`, persist via audit path | threat signal, abuse indicator, auth/authz failure, security finding candidate | generic request traffic truth, access-log contract ownership, standalone log-explorer authority |

## 5. Canonical Schema

### 5.1 Canonical item shape

`Access Log` item authority:

| Field | Status | Reason |
| --- | --- | --- |
| `request_id` | required | canonical per-request correlation seed; already owned by `httpx` request-id middleware |
| `method` | required | canonical HTTP request method fact |
| `path` | required | canonical request URL path fact as received by server runtime |
| `route` | optional | route template may be empty when request does not resolve to a registered route |
| `status_code` | required | canonical HTTP response status fact |
| `duration_ms` | required, derived | derived from request start/end timestamps inside middleware; still part of canonical access-log item |
| `client_ip` | optional | transport/client context may be absent or deployment-dependent |
| `user_agent` | optional | request header may be absent or empty |
| `user_id` | optional | consumer-friendly correlation field for authenticated requests only; must not be fabricated for anonymous requests |
| `username` | optional | bounded operator-friendly correlation field for authenticated requests only; must not become display-name authority |
| `request_size` | optional, derived | may be derived from `Content-Length` or future bounded runtime measurement; absent until runtime owner explicitly captures it |
| `response_size` | optional, derived | may be derived from response writer accounting only after runtime owner explicitly captures it |
| `occurred_at` | required | canonical event timestamp for the completed request record |

### 5.2 Required exclusions

Forbidden fields in canonical `Access Log` authority:

| Field | Status | Reason |
| --- | --- | --- |
| `action` | forbidden | belongs to audit/business semantics |
| `resource_type` | forbidden | belongs to audit/business semantics |
| `resource_id` | forbidden | belongs to audit/business semantics |
| `success` | forbidden | transport truth is `status_code`; audit/security may derive their own result semantics |
| `result` | forbidden | belongs to audit/security outcome taxonomy |
| `risk_level` | forbidden | belongs to audit/security risk semantics |
| `reason` | forbidden | belongs to audit/security/business explanation semantics |
| `metadata` | forbidden | prevents arbitrary unowned semantics from entering the contract |
| `frontend_origin` | forbidden | frontend-owned navigation context is not backend access-log authority |
| `page_route` | forbidden | frontend route is not server access-log truth |
| `permission_code` | forbidden | belongs to authz/audit/security authority, not generic request fact authority |
| `trace_id` as separate authority | forbidden-now | in MVP `traceId=requestId`; no separate tracing authority exists |

### 5.3 Current runtime alignment

Current runtime already emits these canonical fields from `server/internal/httpx/accesslog.go`:

- `requestId`
- `traceId` as `requestId` alias only
- `method`
- `path`
- `route`
- `status`
- `latency`
- `clientIp`
- `userAgent`

Current runtime does not yet emit canonical access-log authority for:

- `user_id`
- `username`
- `request_size`
- `response_size`
- `occurred_at` as an explicit field
- `duration_ms` as normalized millisecond field

Those are contract-defined now, but runtime implementation remains future work.

## 6. Query Contract

Future `Access Log Explorer` may only consume a query contract bounded to canonical owner fields.

Allowed filters:

- `request_id`
- `path`
- `route`
- `status_code`
- `occurred_from`
- `occurred_to`
- `user_id`
- `username`

Allowed query rules:

- `request_id`
  - exact match only
- `path`
  - exact or canonical prefix match owned by backend contract
- `route`
  - exact match only against backend route template
- `status_code`
  - exact match or bounded set match
- `occurred_from` / `occurred_to`
  - inclusive time range on `occurred_at`
- `user_id`
  - exact match only
- `username`
  - exact match or bounded canonical contains/prefix match only if backend owner explicitly supports it in a future runtime topic

Forbidden filters:

- arbitrary metadata keys
- frontend page semantics
- arbitrary headers as a search surface
- audit-only fields such as `action`, `risk_level`, `resource_type`
- monitor-only fields such as anomaly/trend dimensions
- free-form JSON filter blobs

## 7. Sort Contract

Allowed sort fields:

- `occurred_at`
- `duration_ms`
- `status_code`

Default sort:

- `occurred_at desc`

Forbidden sort fields:

- any field without canonical owner
- `username`
- `client_ip`
- `user_agent`
- arbitrary metadata paths

Rationale:

- operator workflows are timeline-first
- unowned or high-cardinality free-form sorts are unstable and not contract-safe

## 8. Pagination Contract

Future contract should use page-based pagination:

- `page`
- `page_size`

Recommended bounds:

- `page >= 1`
- `1 <= page_size <= 200`

Reason:

- consistent with current audit list contract style
- simpler for bounded operator troubleshooting flows
- does not require inventing a cursor contract before durable storage/query authority exists

Cursor pagination is `not-ready` for this topic because no approved durable access-log storage and ordering implementation exists yet.

## 9. Operator Workflow

Canonical troubleshooting path:

`request_id`
-> `Access Log`
-> related `Audit Log`
-> related `Security Event`

Additional allowed paths:

- `user`
  -> `Access Log`
  -> bounded time-window `Audit Log`
- `Audit Incident`
  -> related `Audit Log`
  -> bounded correlation jump by `request_id` into `Access Log`
- `Access Log`
  -> `Audit Log` only when audit authority confirms a matching record

Forbidden workflow assumptions:

- `Access Log` directly explains business intent
- `Access Log` substitutes for audit evidence
- `Security Event` becomes access-log-owned truth
- frontend heuristics create correlation without canonical fields

## 10. Ownership Matrix

| Layer | Owner | Responsibility |
| --- | --- | --- |
| runtime logging semantics | `server/internal/httpx/**` | define request-fact field semantics, request lifecycle capture, normalization rules |
| future durable access-log store | future runtime topic only | not approved in this contract topic |
| future shared API contract | `openapi/**` | own canonical HTTP wire schema after runtime topic approval |
| generated server artifacts | `server/internal/contract/openapi/**` | consume OpenAPI source as derived artifact |
| generated web artifacts | `web/src/contracts/openapi/generated/**` | consume OpenAPI source as derived artifact |
| future web explorer module | future `web/src/modules/<access-log-explorer>/**` | consume canonical contract only; no authority ownership |

## 11. Governance Gaps

- no durable storage authority exists for access logs
- no approved access-log query API exists
- no explicit runtime field capture exists yet for `user_id`, `username`, `request_size`, `response_size`, `occurred_at`, `duration_ms`
- current runtime emits `status` and `latency`, not yet normalized `status_code` and `duration_ms`
- current runtime emits `traceId` only as `requestId` alias; no independent tracing authority exists
- no explicit access-log authz/query permission contract exists
- no OpenAPI source exists yet for access-log explorer consumption

## 12. Recommended Next Runtime Topic

Recommended runtime topic:

- `phase-d-access-log-runtime-storage`

This future topic may start only after it keeps the current contract truthful:

- no access-log UI work before runtime storage/query authority exists
- no reuse of `audit_logs` as access-log store
- no new contract fields from frontend semantics

## 13. Final Verdict

- verdict: `Archive Ready`

Reason:

- future `Access Log Explorer` consumer contract can now be answered
- schema boundary is explicit
- audit/access/security ownership is explicit
- operator workflow is explicit
- runtime/storage gaps remain honestly recorded instead of being backfilled by fake compatibility
