# Phase D App Log Durable Storage Decision

## Status

- Topic: `phase-d-app-log-durable-storage-decision`
- Status: `archived`
- Task class: `server`
- Recovery source: `parent topic`
  - `phase-d-log-retention-and-storage-authority`
  - current `phase-d-app-log-storage-authority` result

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/logger/**` remains the canonical `App Log` authority
  - `AppLogger` remains the runtime write entry
  - `server/internal/httpx/**` remains the `Access Log` authority and only a boundary reference for this topic
  - `server/internal/audit/**` + `server/modules/audit/**` remain the durable `Audit Log` / `Security Event` authority

## Goal

Make one authority-first decision on whether `Graft` should approve repository-owned durable `App Log` storage now.

This topic explicitly does not:

- add tables, migrations, repositories, cleanup jobs, APIs, or UI
- widen `App Log` into `Access Log`, `Audit Log`, `Security Event`, metrics, tracing, OpenTelemetry, or APM
- change `AppLogger` write semantics

## Consumer Evaluation

| Consumer | Current need | Current authority fit | Durable-store readiness |
| --- | --- | --- | --- |
| developer debugging | runtime failure and branch diagnostics | `process_output_only` already fits | no repository durable store needed now |
| operator troubleshooting | bounded operational investigation | partially fit through deployment sink collection | blocked by missing operator workflow |
| admin UI explorer | searchable in-product log browsing | no approved authority yet | blocked by missing query/authz/workflow contract |
| external log collector | sink ingestion from stdout/stderr or process logger | already fits current authority | repository durable store not required |

## Durable Storage Risk Review

| Risk | Current assessment | Why it blocks approval now |
| --- | --- | --- |
| high write volume | high | `AppLogger` covers core runtime, modules, jobs, adapters, and failure summaries; write rate is less bounded than audit evidence and less predictable than request-only access logs |
| DB bloat | high | no approved operator workflow exists to justify how much runtime noise should become repository-kept data |
| index cost | unknown-high | query dimensions for operator/UI use are not defined, so durable schema would either under-index real needs or over-index speculative ones |
| retention / cleanup ownership | missing | current authority remains `none`; approving storage without lifecycle owner would violate the retention rules already recorded in logger governance |
| sensitive data leakage | elevated | `message` plus free-form `fields` are more likely than access logs to carry module-specific secrets or user identifiers despite sanitization |
| overlap with access / audit / security | elevated | without an explicit operator workflow, teams will tend to backfill request facts, actor facts, and security facts into app-log storage, which would erode existing authority boundaries |

## Authority Decision

- Decision: `defer until operator workflow is defined`

Reason:

- `process_output_only` is still the truthful MVP authority for developer debugging and for deployment-level external collection.
- Repository-owned durable `App Log` storage is not justified by current consumers alone because the operator troubleshooting and admin-explorer paths still lack a canonical workflow.
- Access-log durable storage is not a reusable precedent here: `httpx` first established request-fact schema, retention owner, cleanup direction, and explorer workflow. `App Log` does not yet have the equivalent operator contract.
- Approving schema now would force premature decisions on retention, indexing, authz, query shape, and data hygiene while the canonical user journey is still undefined.
- Rejecting durable storage permanently would be too strong because a future operator workflow may justify a bounded repository-owned dataset. The correct decision in this topic is therefore defer, not reject.

## Current Canonical Path

While this decision is deferred:

- `AppLogger` writes to the current process logger output only
- repository runtime owns no `App Log` retention, archive, or purge lifecycle
- operators who need durable search should use deployment-owned external collection from stdout/stderr or the configured process sink
- future repository work must not reuse `audit_logs`, `access_logs`, or monitor storage as an `App Log` shortcut

## Missing Workflow Questions Before Any Schema Topic

1. Who is the canonical operator for repository-owned `App Log` investigation, and what concrete incidents require in-product search instead of external collection?
2. What is the minimum bounded workflow: `request_id` lookup, component-time-window troubleshooting, error-only review, or another narrower path?
3. What authz model should guard future `App Log` queries, and is that model backend-only or shared through `openapi/**` and `web`?
4. What retention window is justified for this dataset, and who owns cleanup execution and config defaults?
5. Which canonical indexes are actually needed for the approved workflow?
6. What additional redaction or field-deny rules are required before module-supplied `message` / `fields` can become durable repository data?
7. How will future `App Log` views link to `Access Log` and `Audit Log` without collapsing those authorities into one mixed dataset?

## Authority Impact

- no authority owner changes in this topic
- `server/internal/logger/**` remains the only `App Log` authority
- `process_output_only` remains canonical current storage mode
- future durable-store work is blocked until an operator workflow topic answers the questions above

## Recommended Next Topic

- `phase-d-app-log-operator-workflow-definition`

Expected scope:

- define the canonical operator troubleshooting workflow for future repository-owned `App Log` search
- define minimal query/authz/correlation needs
- decide whether repository-owned durable storage is still justified after that workflow is explicit

## Final Verdict

- Verdict: `Archive Ready`

Basis:

- the decision is explicit: durable `App Log` storage is deferred, not approved and not permanently rejected
- the current external-collection path and process-output authority stay truthful
- the blocking workflow and lifecycle questions are now concrete enough to constrain the next bounded topic
