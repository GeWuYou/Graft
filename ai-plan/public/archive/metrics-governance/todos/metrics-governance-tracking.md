# Metrics Governance Tracking

## Topic

- Topic: `metrics-governance`
- Parent topic evidence: `observability-development-governance` (archived)
- Scope: authority discovery first; no automatic runtime rollout

## Goal

- classify the canonical metric authority chain before inventory or MVP work
- keep metrics separate from logs, audit events, and frontend-only analytics display
- decide whether the repo currently has only placeholders, one bounded runtime surface, or a broader shared contract need

## Current Recovery Point

- Batch 1 completed startup preflight under root `AGENTS.md`, read both subdomain `AGENTS.md` files, and used the archived observability closeout as parent evidence only.
- Current repository authority for metric-like runtime data is limited:
  - `server/plugins/monitor/**` owns the current `server-status` read model
  - `openapi/**` owns the cross-boundary HTTP schema for that read model
  - `web/src/modules/monitor/**` consumes those fields and does not define their semantics
- `ai-plan/design/日志治理开发规范.md` remains the only repository-wide normative source that mentions `Metric Candidate / Metric Placeholder`.
- No repo-wide metrics emitter, scrape contract, retention model, label policy, or aggregation authority exists yet.
- Batch 2 inventory confirms the current bounded surface in more detail:
  - `server/plugins/monitor/plugin.go` is the only owned runtime path that samples and stores metric-like points
  - `server/plugins/monitor/contract/trend.go` plus `openapi/components/parameters/trend-range-query.yaml` own the stable trend-window contract
  - `openapi/components/schemas/server-status-trend.yaml` and `server-status-trend-point.yaml` own the shared payload shape
  - `web/src/modules/monitor/**` uses metric wording and charts only as downstream presentation
  - `server/plugins/audit/**` overview `trend` data is audit analytics, not metrics authority

## Authority Discovery

- Canonical owner for current runtime metric-like payload:
  - `server/plugins/monitor/**`
- Canonical owner for shared wire shape:
  - `openapi/paths/monitor.server-status.yaml`
- Derived consumers only:
  - `server/internal/contract/openapi/monitor/**`
  - `web/src/modules/monitor/**`
- Not authority:
  - archived observability docs
  - generated schema/types
  - frontend chart composition
  - audit overview trend analytics
  - monitor page metric labels or copy

## Active Risks

- The current `trend` payload can be mistaken for a general metrics platform even though it is process-local monitor state.
- Existing UI wording such as `metric` in the monitor module is presentation vocabulary, not proof of a canonical shared metric abstraction.
- Introducing a platform-wide `MetricsEmitter` now would violate current design governance because label rules, retention, export strategy, and operator consumers are still undefined.
- Audit overview `trend` naming can be misread as metrics evidence even though its authority is the audit-domain overview read model, not a shared metrics surface.

## Design Gap

- No canonical authority yet answers whether future metrics should remain plugin-owned read models, converge into one backend metrics service, or stop at documented placeholders.
- No shared rule exists yet for:
  - who may emit canonical metrics
  - which dimensions may become labels
  - what retention and aggregation window belongs to a canonical metric instead of a page-local trend payload
  - how `web` may consume metrics without becoming a fallback authority
- Batch 3 should treat this as an authority decision first. Runtime rollout is justified only if one minimal canonical repair path becomes explicit.

## Batch 3 Decision

- Batch 3 determined that no smallest safe runtime/OpenAPI/web implementation exists yet.
- The truthful MVP decision is doc-only:
  - accept `monitor/server-status` as the only current canonical metric-like runtime and wire surface
  - keep repo-wide metrics outside that boundary as placeholders only
  - defer any shared metrics service, label taxonomy, export path, or dashboard contract to a future authority-defining topic
- No authority repair is pending inside owned implementation scope because:
  - the monitor plugin and monitor OpenAPI source are already aligned
  - generated consumers are derived artifacts only
  - frontend monitor wording is presentation, not authority drift
  - audit analytics trend fields are domain-local analytics, not metrics contracts

## Final Recovery Point

- Batch 4 completed the final archive-readiness check.
- Archive-ready evidence:
  - the doc-only MVP decision is fully recorded
  - `server/plugins/monitor/**` remains the only current runtime metric-like authority
  - `openapi/paths/monitor.server-status.yaml` remains the only shared wire authority for that surface
  - broader backend/frontend validation stayed deferred for honest scope reasons because no runtime or shared contract files changed in this topic
- No active continuation remains for this topic.
- Any future metrics implementation must start as a new bounded topic with a fresh startup preflight and explicit authority owner.
