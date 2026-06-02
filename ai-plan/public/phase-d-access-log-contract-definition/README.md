# Phase D Access Log Contract Definition

## Status

- Topic: `phase-d-access-log-contract-definition`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `phase-d-log-retention-and-storage-authority`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/httpx/**` owns `Access Log` runtime semantics
  - `openapi/**` is the future shared wire-contract authority after runtime approval
  - `web` is future downstream consumer only

## Deliverables

### Access Log Boundary Matrix

| Surface | Canonical owner | Responsible for | Must not own |
| --- | --- | --- | --- |
| Access Log | `server/internal/httpx/**` | request lifecycle, transport facts, latency, status, request correlation | audit truth, business-event truth, security-finding truth, frontend semantics |
| Audit Log / Audit Event | `server/internal/audit/**` + `server/modules/audit/**` | user action, business event, permission decision, compliance evidence | request-traffic truth, access-log storage authority |
| Security Event | publish in `server/internal/httpx/**`, persist through audit path | threat signal, abuse indicator, auth/authz failure, security finding candidate | generic request-traffic truth, standalone access-log authority |

### Canonical Access Log Schema

| Field | Status | Notes |
| --- | --- | --- |
| `request_id` | required | canonical per-request correlation seed |
| `method` | required | canonical transport fact |
| `path` | required | canonical request path fact |
| `route` | optional | empty when no canonical route template resolves |
| `status_code` | required | canonical response status fact |
| `duration_ms` | required, derived | middleware-derived request duration |
| `client_ip` | optional | bounded client context |
| `user_agent` | optional | bounded client context |
| `user_id` | optional | authenticated-request correlation only |
| `username` | optional | authenticated-request operator correlation only |
| `request_size` | optional, derived | future bounded runtime measurement only |
| `response_size` | optional, derived | future bounded runtime measurement only |
| `occurred_at` | required | canonical completed-request timestamp |

Forbidden:

- `action`
- `resource_type`
- `resource_id`
- `success`
- `result`
- `risk_level`
- `reason`
- `metadata`
- frontend-owned route/origin fields
- monitor-owned anomaly/trend fields

### Query Contract

Allowed filters:

- `request_id`
- `path`
- `route`
- `status_code`
- `occurred_from`
- `occurred_to`
- `user_id`
- `username`

Forbidden filters:

- arbitrary metadata
- frontend-owned semantics
- audit-only business fields
- monitor-only semantics
- free-form JSON filter payloads

### Sort Contract

Allowed sort:

- `occurred_at`
- `duration_ms`
- `status_code`

Default:

- `occurred_at desc`

Forbidden:

- no-owner fields
- arbitrary metadata-path sort
- unstable high-cardinality client fields as canonical sort

### Pagination Contract

Chosen contract:

- `page`
- `page_size`

Reason:

- matches existing audit list style
- enough for bounded troubleshooting workflows
- avoids inventing cursor/storage semantics before storage authority exists

### Operator Workflow

Canonical path:

`request_id`
-> `Access Log`
-> related `Audit Log`
-> related `Security Event`

Allowed jumps:

- `user` -> `Access Log` -> bounded time-window `Audit Log`
- `Audit Incident` -> `Audit Log` -> correlation jump to `Access Log`
- `Access Log` -> `Audit Log` only when audit authority confirms a matching record

### Ownership Matrix

| Layer | Owner | Responsibility |
| --- | --- | --- |
| runtime semantics | `server/internal/httpx/**` | canonical access-log field semantics |
| future durable store | future runtime topic only | not approved in this topic |
| future wire contract | `openapi/**` | canonical API schema after runtime approval |
| generated server artifact | `server/internal/contract/openapi/**` | derived consumer |
| generated web artifact | `web/src/contracts/openapi/generated/**` | derived consumer |
| future explorer UI | future `web/src/modules/<access-log-explorer>/**` | downstream consumer only |

### Governance Gap List

- no durable storage authority exists for access logs
- no approved query API exists
- current runtime does not yet normalize `duration_ms`, `status_code`, `occurred_at`
- current runtime does not yet capture `user_id`, `username`, `request_size`, `response_size`
- no explicit authz/query-permission contract exists for future explorer/API
- no OpenAPI source exists yet for access-log explorer consumption

### Recommended Runtime Topic

- `phase-d-access-log-runtime-storage`

## Final Verdict

- Verdict: `Archive Ready`

Basis:

- future explorer consumer contract is now explicit
- schema, boundary, owner, and operator workflow are explicit
- runtime gaps remain recorded as governance gaps instead of being implemented prematurely
