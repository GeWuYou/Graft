# Phase D Access Log Retention Governance

## Status

- Topic: `phase-d-access-log-retention-governance`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `phase-d-access-log-explorer-contract`
  - archive-ready evidence from:
    - `phase-d-access-log-runtime-storage`
    - `phase-d-access-log-explorer-contract`
    - `phase-d-access-log-explorer-implementation`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/httpx/**` owns access-log field semantics, durable table lifecycle, and retention policy
  - `web` remains a downstream explorer consumer only
  - no audit, monitor, or plugin-registry authority supersedes `httpx` for access-log retention

## Goal

Define canonical retention governance for `Access Log` without implementing cleanup runtime.

This topic explicitly does not implement:

- cleanup jobs
- scheduler wiring
- retention UI
- delete endpoints
- archive/export
- metrics, tracing, app-log retention, or generic log platform work

## Authority Decision

Chosen owner:

- `server/internal/httpx/**`

Reason:

- retention is part of the access-log dataset lifecycle, and the current table, repository, migration, explorer query contract, and delete primitive already live in `httpx`
- operations config may later provide values, but it is a consumer/input surface rather than semantic authority
- explorer consumers must not define data-history guarantees

## Retention Policy Matrix

| Environment | Recommended default | Policy style |
| --- | --- | --- |
| development | 3 days | configurable |
| staging | 7 days | configurable |
| production | 30 days | configurable |

Additional rules:

- no unlimited default
- environment label alone is not the authority surface
- future runtime config should expose one canonical retention setting with owner-defined defaults

## Cleanup Trigger Decision

Chosen direction:

- future `httpx` maintenance job running on a schedule after explicit runtime-topic approval

Rejected as canonical path:

- startup cleanup
- manual cleanup only

Current truth:

- no cleanup executor is implemented
- the repository delete primitive is only a lower-level storage capability

## Storage Lifecycle Decision

| Stage | Decision |
| --- | --- |
| ingest | middleware writes request facts into `access_logs` |
| retained | queryable while inside retention window |
| expired | future hard-delete target |
| archived/exported | not approved |

Explorer effect after expiry:

- expired records simply disappear from the retained dataset
- exact lookup may return not found
- time-window queries may return empty results
- pagination reflects the current retained snapshot only

## Storage Growth Summary

Primary growth drivers:

- request volume
- retention duration
- row width from `path` / `user_agent` / optional correlation fields
- secondary index count

Required current indexes:

- `idx_access_logs_occurred_at_id`
- `idx_access_logs_request_id`
- `idx_access_logs_route_occurred_at`
- `idx_access_logs_user_id_occurred_at`

Current decision:

- keep existing indexes
- do not add path/status/duration/username indexes in this governance slice
- use retention as the primary MVP growth-control mechanism

## Audit And Security Boundary

- `Access Log` remains request-fact storage only
- retention expiry does not redefine audit evidence or security-event semantics
- access-log retention must not be described as audit-retention policy

## Recommended Next Topic

- `phase-d-access-log-retention-runtime`

Expected runtime-topic scope:

- add config surface for retention
- wire environment defaults
- implement bounded cleanup execution path
- validate behavior against current storage/query model
