# Phase D App Log Retention Authz And Storage Readiness

## Status

- Topic: `phase-d-app-log-retention-authz-and-storage-readiness`
- Status: `archived`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `phase-d-app-log-durable-storage-decision`
  - `phase-d-app-log-operator-workflow-definition`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/logger/**` remains the canonical `App Log` authority
  - `AppLogger` remains the current runtime write entry
  - `App Log` remains `process_output_only`
  - `openapi/**` and `web/**` remain downstream and out of scope until durable storage is explicitly approved
  - `server/internal/httpx/**` remains `Access Log` authority and cannot be reused as an App Log storage shortcut
  - `server/internal/audit/**` + `server/modules/audit/**` remain `Audit Log` / `Security Event` authority

## Goal

Decide whether App Log retention, authz, and storage readiness is strong enough to approve repository-owned durable
`App Log` storage, an App Log API, or an App Log Explorer.

This topic explicitly does not:

- add an `app_logs` schema, migration, repository implementation, or cleanup job
- add OpenAPI paths or generated contracts
- add a `web` App Log Explorer module or UI
- change `AppLogger` runtime sink behavior
- widen `App Log` into `Access Log`, `Audit Log`, `Security Event`, metrics, tracing, OpenTelemetry, or APM

## Parent Evidence

Prior archived topics established:

- durable storage was deferred because operator workflow, authz, retention, cleanup, and data-volume ownership were
  missing
- bounded operator workflows are now explicit:
  - runtime error investigation
  - background job failure investigation
  - module startup / config failure investigation
  - request-correlated troubleshooting by `request_id` / `trace_id`
- the workflow definition did not approve repository-owned durable storage or an explorer

## Readiness Verdict

- Verdict: `deferred-not-approved`
- Runtime readiness: `Partially Ready`
- Schema/API/UI readiness: `Not Approved`

Reason:

- the operator workflow is now concrete enough to define readiness criteria
- repository-owned durable storage still lacks explicit approval of retention duration, cleanup execution, authz contract,
  and repository data-volume acceptance
- deployment-owned external collection remains the recommended durable-search path for MVP operators
- approving API or UI before storage/authz would invert the authority chain

## Retention Decision

Current canonical state:

| Concern | Decision |
| --- | --- |
| storage mode | `process_output_only` |
| repository retention owner | `none` |
| repository default retention window | `0` |
| archive/export owner | `none` |
| cleanup execution owner | `none` until durable storage is approved |

Future durable-store readiness criteria:

- `server/internal/logger/**` must remain the semantic and storage-lifecycle owner
- one canonical retention setting must be approved before schema work starts
- an MVP default may be proposed only in the durable-storage runtime topic, with development/staging/production rationale
- unlimited retention is not an allowed default
- archive/export stays `not-ready` unless a separate archive authority is approved

## Authz And Query Decision

No shared wire contract is approved in this topic.

Future App Log query authz must be approved before OpenAPI or web work starts:

- backend permission owner:
  - `server/internal/logger/**` or a documented core logging registration boundary
- minimum read permission:
  - one read-only permission for App Log troubleshooting data
- forbidden permission shortcuts:
  - do not reuse `access_log.read`
  - do not reuse `audit.read`
  - do not treat menu visibility as backend authorization

Allowed future query dimensions:

- `occurred_at` bounded time window
- `severity`
- `component`
- `operation`
- `request_id`
- `trace_id`
- bounded `message` keyword search

Forbidden query dimensions and surfaces:

- access traffic analytics fields such as `path`, `status_code`, `request_size`, `response_size`, `client_ip`,
  `user_agent`
- audit and compliance fields such as `actor_id`, `actor_type`, `resource_type`, `resource_id`, `action`, `decision`,
  `policy`, `permission`
- standalone security-event timelines, credential timelines, token timelines, metrics dashboards, tracing span views,
  arbitrary JSON filters, regex DSL, and cross-domain joins

## Storage Decision

Repository-owned durable storage remains deferred.

Future storage approval must answer:

- why deployment-owned external collection is insufficient for the approved operator workflows
- expected write volume and row growth bounds for MVP
- which indexes are justified by the allowed query dimensions
- how `message` and `fields` are sanitized before becoming durable repository data
- how cleanup runs without turning scheduler mechanics into policy ownership
- how App Log correlation links to Access Log and Audit without absorbing their fields or evidence semantics

Storage shortcuts that remain forbidden:

- `audit_logs` as App Log storage
- `access_logs` as App Log storage
- Redis TTL storage for App Log truth
- OpenAPI App Log query contract before storage authority approval
- frontend-derived filters or metadata as App Log authority

## Approval Gate For Later Work

Before any later schema/API/UI topic starts, it must record:

- durable-storage approval status
- retention owner and default window
- cleanup lifecycle owner and execution path
- read permission code and authz owner
- approved query dimensions
- approved persisted fields and forbidden fields
- validation scope covering server runtime, OpenAPI generation, and web consumption if those surfaces are included

Until that gate is passed:

- no `app_logs` table
- no App Log repository implementation
- no App Log cleanup job
- no App Log OpenAPI path
- no App Log Explorer UI

## Authority Impact

- no authority owner changes in this topic
- `server/internal/logger/**` remains canonical `App Log` authority
- current `App Log` storage remains `process_output_only`
- `openapi/**` and `web/**` remain downstream future consumers only

## Recommended Next Topic

- `phase-d-app-log-durable-storage-runtime-approval`

Expected scope only if explicitly approved:

- decide whether repository-owned durable storage is finally approved after this readiness gate
- if approved, define schema, retention config, cleanup execution, permission code, OpenAPI contract, and validation scope
- if not approved, keep external collection as the durable operational path

## Final Verdict

- Verdict: `Archive Ready`

Basis:

- retention, authz, and storage readiness criteria are explicit
- the current runtime remains process-output-only without overclaiming repository durability
- no schema, API, or UI work was approved in this topic
