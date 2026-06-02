# Phase D Access Log Explorer Contract

## Status

- Topic: `phase-d-access-log-explorer-contract`
- Status: `archived`
- Task class: `cross-boundary`
- Recovery source: `subtopic`
  - `ai-plan/public/phase-d-access-log-runtime-storage/README.md`
  - `ai-plan/design/Access-Log-Authority-Contract.md`
  - `server/internal/httpx/**`
  - current observability governance docs
  - current audit/monitor/admin explorer UX patterns

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `subtopic`
- authority summary:
  - `server/internal/httpx/**` owns access-log runtime semantics and current storage baseline
  - `openapi/**` is future shared explorer wire-contract authority only after implementation approval
  - `web` is a downstream explorer consumer only

## Goal

Define the canonical `Access Log Explorer` authority before runtime API or UI implementation.

This round is authority-first exploration and contract design only.

## Deliverables

### Authority Summary

- `Access Log Explorer` is the operator read surface for canonical HTTP request facts
- canonical field/query/sort/pagination authority is defined in [Access-Log-Explorer-Authority.md](../../design/Access-Log-Explorer-Authority.md)
- `Audit`, `Security Event`, and `Monitor` may correlate with access logs but do not own access-log explorer truth

### Ownership Matrix

| Surface | Canonical owner | Responsibility | Must not own |
| --- | --- | --- | --- |
| Access Log runtime/storage baseline | `server/internal/httpx/**` | request-fact capture, normalization, current durable storage | audit result taxonomy, monitor anomaly semantics, frontend query semantics |
| Access Log Explorer contract | `ai-plan/design/Access-Log-Explorer-Authority.md` -> future backend implementation topic | canonical filter/sort/pagination/detail semantics | app-log explorer, retention policy, tracing, metrics |
| Shared HTTP explorer contract | `openapi/**` in next topic only | wire schema after implementation approval | authority discovery for field meaning |
| Web explorer module | future `web/src/modules/<access-log-explorer>/**` | UI consumption, route-query sync, drawer state, presets as UI-only context | backend filter/sort authority |

### Query Matrix

| Field | Decision |
| --- | --- |
| `request_id` | exact-match filter |
| `trace_id` | no separate backend field; alias-only MVP concept |
| `keyword` | forbidden in canonical MVP contract |
| `user_id` | exact-match filter |
| `username` | exact-match filter |
| `method` | exact/bounded-set filter |
| `path` | exact or canonical-prefix filter |
| `route` | exact-match filter |
| `status_code` | exact/bounded-set filter and allowed sort |
| `duration_ms` | inclusive range filter and allowed sort |
| `occurred_at` | inclusive range filter and default sort field |
| `client_ip` | display-only for now |
| `user_agent` | display-only for now |
| `request_size` | display-only for now |
| `response_size` | display-only for now |

### Sort Matrix

| Field | Decision |
| --- | --- |
| `occurred_at` | allowed, default `desc` |
| `duration_ms` | allowed |
| `status_code` | allowed |
| all other fields | unsupported and should be rejected |

### Pagination Decision

- chosen model: `page` + `page_size`
- rationale:
  - matches current explorer/list UX in this repo
  - aligns with current timeline-first access-log storage shape
  - avoids premature cursor semantics before explicit cursor authority exists

### Correlation Boundary Decision

- `Audit Log`
  - may correlate by `request_id`, actor, and bounded time window
  - retains audit evidence truth
- `Security Event`
  - may share request correlation
  - retains security classification outside access-log explorer
- `Monitor Incident`
  - may reach request facts only through bounded correlation flows
  - retains incident/anomaly/trend truth

### Retention Boundary Decision

- explorer assumes only that some retained access-log dataset exists
- explorer does not own retention duration, purge, archive, or minimum-history guarantees
- current retention remains intentionally undefined and must stay undefined in this topic

### UX Recommendation

- use `list-form-detail`
- use dedicated filter bar plus table-first explorer layout
- use server-owned pagination
- use right-side detail drawer for one request
- allow quick presets only as frontend navigation/context helpers, not backend authority
- avoid dashboard-first explorer, free-form keyword authority, tag-builder-only primary UX, or embedded audit/incident models

## Unresolved Decisions

- future permission code for explorer access
- whether `client_ip` can ever become filterable under explicit privacy/authz governance
- whether `username` should remain exact-only or later support bounded prefix search
- whether request/response size become operator-significant query fields

## Recommended Next Topic

- `phase-d-access-log-explorer-implementation`

## Final Verdict

- Verdict: `Archive Ready`

Basis:

- explorer ownership is now separated from runtime capture, audit truth, and monitor truth
- canonical query/sort/pagination boundaries are explicit before any API or UI work
- retention remains truthfully unowned instead of being invented by the explorer topic
