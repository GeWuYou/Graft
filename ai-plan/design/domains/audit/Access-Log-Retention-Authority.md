# Access Log Retention Authority

## 1. Goal

This document defines the canonical retention governance for `Access Log` in `Graft`.

This document is authority-definition only.

It does not approve:

- cleanup jobs
- cron scheduling
- retention UI
- delete endpoints
- archive/export
- metrics, tracing, or app-log expansion

## 2. Authority Summary

- access-log runtime and durable-storage owner
  - `server/internal/httpx/**`
- retention policy owner
  - `server/internal/httpx/**`
- future runtime configuration consumer
  - `server/internal/app/**` and repository configuration wiring
- future shared wire contract consumers
  - `openapi/**`
  - `server/internal/contract/openapi/**`
  - `web/src/contracts/openapi/generated/**`
- future frontend explorer consumer
  - `web/src/modules/<access-log-explorer>/**`

`web` is a downstream consumer only.

`server/modules/audit/**`, `server/modules/monitor/**`, and historical module-registry metadata may consume access-log retention outcomes but do not own retention policy.

## 3. Ownership Decision

Chosen canonical owner:

- `server/internal/httpx/**`

Rationale:

- retention is inseparable from access-log dataset lifecycle
- current durable table, repository contract, and delete primitive already live in `httpx`
- retention semantics must track request-fact storage shape, query guarantees, and future cleanup implementation
- historical module registry owns module lifecycle metadata, not core HTTP request-fact storage policy
- operations configuration may provide values later, but configuration input is not the same thing as semantic authority
- future observability governance may classify policy families, but `Access Log` retention remains dataset-local authority under `httpx`

Rejected as canonical owner:

| Candidate | Verdict | Reason |
| --- | --- | --- |
| historical module registry | reject | registry does not own `access_logs` schema, query contract, or deletion semantics |
| `server/internal/app/**` / operations config | reject as semantic owner | runtime assembly may inject configured values later, but it should not define dataset lifecycle semantics |
| `server/modules/audit/**` | reject | audit owns audit/security evidence retention, not request-fact storage |
| `server/modules/monitor/**` | reject | monitor owns anomaly/trend semantics, not request-fact history |
| `web` | reject | explorer only consumes retained data; it cannot promise history duration |

## 4. Retention Policy Strategy

Chosen direction:

- configurable retention with environment-specific defaults

Rationale:

- fixed retention is too rigid for different deployment sizes and operational constraints
- pure environment-based retention without one canonical config surface hides the real policy behind deployment convention
- configurable retention lets the repository define one semantic policy while allowing safer production vs non-production defaults

Policy rules:

- one canonical retention setting applies to the access-log dataset
- the setting should be expressed as a duration or days-based window in future runtime config
- environment may change the configured value, but environment label itself is not the authority surface
- if configuration is absent, runtime should fall back to owner-defined defaults rather than “retain forever”

Recommended default matrix for future implementation:

| Environment | Recommended default | Rationale |
| --- | --- | --- |
| development | 3 days | enough for local debugging without growing developer databases unnecessarily |
| staging | 7 days | enough for short regression and release validation windows |
| production | 30 days | reasonable operator troubleshooting window for MVP without implying archive/compliance retention |

Guardrails:

- these defaults are operational recommendations, not legal/compliance guarantees
- retention shorter than 24 hours is discouraged because it weakens request troubleshooting value
- retention longer than 90 days should require explicit operator intent because row growth is linear and no archive path exists yet
- “forever retention” is not an allowed default for MVP

## 5. Cleanup Trigger Strategy

Chosen canonical direction:

- future maintenance job under `httpx` authority, expected to run on a schedule once implementation is approved

Evaluation:

| Trigger | Verdict | Reason |
| --- | --- | --- |
| startup cleanup | reject as canonical path | couples deletion cost to boot, creates unpredictable startup latency, and misses long-running runtime cleanup needs |
| scheduled cleanup | chosen direction | aligns with time-window retention, keeps runtime lifecycle explicit, and decouples deletion from operator requests |
| manual cleanup only | reject as canonical path | unsafe for unattended growth control and turns retention into an operational habit instead of platform behavior |
| future maintenance job | chosen expression | matches repository architecture for explicit bounded background work without approving scheduler implementation in this topic |

Current status:

- no cleanup executor is approved yet
- no scheduler wiring is approved yet
- the existing repository delete primitive is a storage capability, not a policy implementation

## 6. Storage Growth Analysis

### 6.1 Growth drivers

Primary growth drivers:

- request volume
- retention duration
- average row width
- secondary index count and cardinality

Current row shape characteristics:

- fixed-size / bounded fields:
  - `id`
  - `request_id`
  - `method`
  - `status_code`
  - `duration_ms`
  - `user_id`
  - `occurred_at`
- variable-size fields:
  - `path`
  - `route`
  - `client_ip`
  - `user_agent`
  - `username`
- nullable fields reduce average row width but do not remove index costs

### 6.2 Row-size estimate

Pragmatic MVP estimate:

- typical row: roughly `200B` to `600B` before PostgreSQL page/index overhead
- rows with long `path` or `user_agent` values can exceed that range
- effective storage footprint should be treated as “base row + multiple indexes”, so real disk use will commonly be materially higher than raw payload bytes

### 6.3 Growth examples

Illustrative order-of-magnitude estimates:

| Request volume | Retention | Approx rows retained |
| --- | --- | --- |
| 10k / day | 30 days | 300k |
| 100k / day | 30 days | 3M |
| 1M / day | 30 days | 30M |

Implication:

- even moderate production traffic can move this table into multi-million-row territory quickly
- retention duration is therefore the primary MVP growth control, ahead of archive/export features

### 6.4 Index assessment

Current required indexes remain:

| Index | Keep | Reason |
| --- | --- | --- |
| `idx_access_logs_occurred_at_id` | yes | supports default timeline sort and stable pagination |
| `idx_access_logs_request_id` | yes | supports highest-confidence exact lookup |
| `idx_access_logs_route_occurred_at` | yes | supports route troubleshooting with time ordering |
| `idx_access_logs_user_id_occurred_at` | yes | supports authenticated-user correlation with time ordering |

Current non-required indexes:

- no separate index on `status_code`
- no separate index on `duration_ms`
- no separate index on `path`
- no separate index on `username`

Rationale:

- current explorer contract allows those filters, but they should first rely on bounded windows and the timeline index before more write-amplifying indexes are approved
- additional indexes should be justified by real operator workload evidence in a later runtime/perf topic

### 6.5 Growth risks

Expected risks:

- high-volume production deployments can grow `access_logs` rapidly
- delete operations on large retained datasets may create vacuum/maintenance pressure
- high-cardinality text fields make extra indexing expensive
- offset pagination cost rises with deep page traversal, especially when retention windows are wide

This topic does not approve:

- partitioning
- archive tables
- cold storage
- schema redesign

Those remain future authority topics if growth evidence requires them.

## 7. Explorer Interaction After Expiry

Retention effect:

- expired access-log rows are deleted from the canonical retained dataset

User-facing behavior:

- explorer should simply reflect the retained dataset truth
- there is no promise that an older request still exists once outside retention
- missing historical rows after expiry are not an error condition

Query behavior:

- filters and list endpoints operate only on currently retained rows
- exact lookup by `request_id` may return not found when the row has expired
- bounded time filters outside the retained window may return empty results

Pagination behavior:

- page-based pagination remains valid only for the current retained snapshot
- total counts and page contents may shrink as cleanup removes old rows
- requesting a page beyond the new total after cleanup should return an empty item set or the backend’s standard bounded pagination response, not resurrect expired history

Recommended UX implication for future explorer work:

- present empty-state messaging that truthfully indicates no retained records matched the query
- do not imply archive recovery, export recovery, or soft-delete recovery

## 8. Audit And Security Boundary

Retention policy must not create new audit or security semantics.

Rules:

- `Access Log` remains request-fact storage only
- retention expiry of `Access Log` does not delete or redefine audit evidence semantics
- security-event persistence rules remain owned by audit/security paths, not by access-log retention
- no compliance-preservation guarantee should be inferred from access-log retention defaults
- access-log retention must not be described as an audit-retention policy

## 9. Policy Matrix

### 9.1 Ownership matrix

| Concern | Canonical owner | Notes |
| --- | --- | --- |
| access-log field semantics | `server/internal/httpx/**` | already defined by access-log authority |
| access-log durable storage lifecycle | `server/internal/httpx/**` | includes retention semantics |
| retention config values at deploy time | future operations config consumed by runtime | input source only, not semantic owner |
| cleanup execution | future `httpx` maintenance job | not implemented yet |
| explorer history guarantees | none beyond retained dataset | `web` consumes retained truth only |

### 9.2 Retention matrix

| Dimension | Decision |
| --- | --- |
| policy style | configurable retention |
| defaulting style | environment-specific defaults under one canonical config surface |
| canonical production recommendation | 30 days |
| canonical staging recommendation | 7 days |
| canonical development recommendation | 3 days |
| unlimited default | forbidden |
| archive/export dependency | none for MVP; not approved |

### 9.3 Storage lifecycle matrix

| Stage | Meaning | Owner |
| --- | --- | --- |
| ingest | request facts written by middleware into `access_logs` | `server/internal/httpx/**` |
| retained | queryable inside current retention window | `server/internal/httpx/**` |
| expired | eligible for hard delete by future maintenance job | `server/internal/httpx/**` |
| archived | not approved | none |
| restored/exported | not approved | none |

## 10. Recommended Next Topic

Recommended implementation topic:

- `phase-d-access-log-retention-runtime`

Expected scope:

- add bounded runtime configuration for access-log retention
- wire owner-defined defaults
- implement one explicit maintenance execution path for retention cleanup
- validate delete behavior against current `access_logs` indexes and pagination semantics

Still out of scope for that next topic unless separately approved:

- archive/export
- retention UI
- scheduler platform expansion beyond the minimal required maintenance path
- metrics/tracing/log-product broadening
