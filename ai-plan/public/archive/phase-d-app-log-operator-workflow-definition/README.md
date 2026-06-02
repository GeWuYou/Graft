# Phase D App Log Operator Workflow Definition

## Status

- Topic: `phase-d-app-log-operator-workflow-definition`
- Status: `archived`
- Task class: `server`
- Recovery source: `parent topic`
  - `phase-d-app-log-durable-storage-decision`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/logger/**` remains the canonical `App Log` authority
  - `AppLogger` remains the runtime write entry
  - `App Log` remains `process_output_only`
  - `server/internal/httpx/**` remains `Access Log` / request-correlation authority only
  - `server/internal/audit/**` + `server/modules/audit/**` remain `Audit Log` / `Security Event` authority

## Goal

Define the minimum operator workflow that must exist before approving repository-owned durable `App Log` storage or a future `App Log Explorer`.

This topic explicitly does not:

- add tables, migrations, repositories, cleanup jobs, APIs, or UI
- change `AppLogger` runtime wiring or sink behavior
- widen `App Log` into `Access Log`, `Audit Log`, `Security Event`, metrics, tracing, OpenTelemetry, or APM

## Expected Users

| User | Canonical need | Current truthful path | Durable-store pressure |
| --- | --- | --- | --- |
| developer | startup/runtime branch debugging, local and test failure diagnosis | `process_output_only` | low |
| operator / admin | production or staging runtime troubleshooting for bounded incidents | external collector over process output | medium |
| support / debugging role | correlation-based lookup for one failing request or one bounded background execution | external collector over process output | medium |

## Minimum Operator Workflows

### Runtime Error Investigation

Seed:

- time window
- severity `error` / `warn`
- component
- optional keyword from message/error text

Workflow:

1. narrow to the incident time window
2. isolate the component or named logger branch
3. review `error` first, then bounded related `warn` / `info`
4. identify the operation, retry state, or dependency failure summary
5. if request-correlated, jump by `request_id` / `trace_id` into bounded access/audit evidence

### Background Job Failure Investigation

Seed:

- job failure time
- component
- operation / job name

Workflow:

1. locate failure entries in the job window
2. review same-component start / skip / retry / terminal failure summaries
3. determine whether the failure is config, dependency, or data-state related
4. escalate to audit or monitor only when those domains already own matching evidence

### Module Startup / Config Failure Investigation

Seed:

- module component
- boot time window
- config validation error keyword

Workflow:

1. inspect `Register / Boot / Shutdown` summaries for the module/component
2. find dependency-missing, config-invalid, or external-service-unreachable entries
3. use the result to repair config or startup ordering
4. do not turn module startup diagnosis into access/audit/security query requirements

### Request-Correlated Troubleshooting

Seed:

- `request_id`
- `trace_id`

Workflow:

1. query bounded `App Log` context by correlation id
2. inspect same-request runtime messages only
3. jump to `Access Log` for transport facts when needed
4. jump to `Audit Log` / `Security Event` only when those authorities confirm related evidence

## Query Contract

### Required Query Dimensions

| Dimension | Required | Why |
| --- | --- | --- |
| `time window` | yes | every approved workflow is bounded in time |
| `severity` | yes | error-first and warn-first investigation |
| `component` | yes | canonical runtime ownership / module branch narrowing |
| `operation` | yes | job, startup step, integration action narrowing |
| `request_id` | yes | request-correlated troubleshooting |
| `trace_id` | yes | MVP correlation alias and future-proof correlation field |
| `keyword/message search` | yes, bounded | operator often starts from known error text or dependency name |

### Forbidden Query Dimensions

| Surface | Forbidden examples | Why |
| --- | --- | --- |
| access traffic analytics | `path`, `status_code`, `request_size`, `response_size`, `client_ip`, `user_agent` | owned by `Access Log` |
| audit / compliance review | `actor_id`, `resource_id`, `action`, `decision`, `policy`, `permission` | owned by `Audit Log` / `Security Event` |
| security incident timeline | session / token / threat timeline semantics | owned by security-event and audit investigation paths |
| metrics dashboard | trend, percentile, aggregate KPI, dashboard cards | not `App Log` authority |
| tracing span view | span tree, waterfall, service graph | not available in MVP |
| generic query platform | arbitrary JSON filters, regex DSL, cross-domain joins | unjustified for bounded operator workflow |

## Relationship With Other Surfaces

| Surface | Relationship |
| --- | --- |
| `Access Log Explorer` | future sibling surface that owns request traffic facts; `App Log` may jump there by correlation only |
| `Audit Log` / `Security Event` | authoritative evidence path for compliance, actor, permission, and security review; `App Log` is never the evidence root |
| `Monitor` | separate observability authority for anomaly/trend evidence; no monitor dashboard semantics in `App Log` |

## Decision

- Keep `process_output_only`: `yes`
- Recommend deployment-owned external collector: `yes`
- Approve repository-owned durable `App Log` store now: `no`
- Approve repository-owned `App Log Explorer` now: `no`
- Durable-store status after workflow analysis: `remain deferred`

Reason:

- the workflows are real but still bounded troubleshooting workflows, not proof that the repository must own a second durable operational dataset now
- developers are already fully served by process output
- operators and support roles can truthfully rely on external collection until retention owner, authz contract, cleanup lifecycle, and data-volume acceptance are explicitly approved
- approving explorer before storage/authz would invert the authority chain

## Authority Impact

- no authority owner changes
- `server/internal/logger/**` remains canonical `App Log` authority
- `server/internal/httpx/**` remains the request-correlation and `Access Log` authority
- `server/internal/audit/**` + `server/modules/audit/**` remain audit/security evidence authority
- this topic strengthens workflow constraints but does not approve runtime storage, API, or UI

## Recommended Next Topic

- `phase-d-app-log-retention-authz-and-storage-readiness`

Expected scope:

- decide whether repository-owned durable `App Log` storage is still needed after comparing the approved workflows against deployment-owned external collection
- if still needed, define retention owner, authz/query permission contract, cleanup lifecycle, and readiness criteria before any schema/API/UI topic

## Final Verdict

- Verdict: `Archive Ready`

Basis:

- minimum operator workflows are now explicit
- required and forbidden query dimensions are explicit
- `App Log` vs `Access Log` / `Audit` / `Monitor` boundaries remain explicit
- durable `App Log` storage still truthfully remains deferred after workflow analysis
