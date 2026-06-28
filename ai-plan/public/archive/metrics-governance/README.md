# Metrics Governance

## Status

- Topic: `metrics-governance`
- Status: `archived`
- Loop mode: `topic-completion-loop`
- Worktree: `feat/wt-audit-plugin-mvp`
- Task class: `cross-boundary`
- Started: `2026-05-29`
- Closed: `2026-05-29`
- Parent evidence:
  - `ai-plan/public/archive/observability-development-governance/README.md`

## Goal

在不误把日志、审计或前端展示当成 metrics authority 的前提下，单独完成 metrics 的治理开题：

- 分类 metrics authority 与边界
- 确认 `server` / `web` 的 canonical metric surface
- 盘点当前仓库里的 metrics placeholder、trend 数据和 authority gap
- 只在 authority 清晰时推进最小 MVP slice

## Recovery Inputs

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/public/README.md`
- `ai-plan/public/archive/observability-development-governance/README.md`
- `ai-plan/design/日志治理开发规范.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`

## Scope

- Owned scope:
  - `server/internal/**`
  - `server/plugins/**`
  - `web/src/modules/**`
  - `web/src/shared/**`
  - `openapi/**`
  - `ai-plan/public/README.md`
  - `ai-plan/design/日志治理开发规范.md`
  - `ai-plan/public/observability-development-governance/**`
  - `ai-plan/public/archive/metrics-governance/**`
- Forbidden scope:
  - parsing logs and calling that metrics
  - OpenTelemetry rollout
  - Prometheus / Grafana / exporter rollout
  - fake dashboards or frontend-derived metrics contracts
  - widening monitor into a multi-capability observability platform without new authority proof

## Authority Summary

- Repository governance source:
  - root `AGENTS.md`
- Topic authority baseline:
  - `ai-plan/design/日志治理开发规范.md` defines the current canonical boundary for `Metric Candidate / Metric Placeholder`
- Current backend runtime authority:
  - `server/plugins/monitor/**` owns the existing `monitor/server-status` runtime read model, including current trend data
  - `server/plugins/monitor/contract/**` owns monitor-specific stable route, menu, permission, and trend-range semantics
- Shared HTTP contract authority:
  - `openapi/paths/monitor.server-status.yaml` and related `openapi/components/**` own the canonical cross-boundary HTTP shape
  - generated `server/internal/contract/openapi/monitor/**` and frontend schema types are derived artifacts only
- Current frontend authority boundary:
  - `web/src/modules/monitor/**` owns monitor UI composition only
  - `web` is a downstream consumer of backend/OpenAPI metric-like fields and does not own their meaning

## Batch 1 Result

- `metrics-governance` is justified as a new active topic because archived observability governance explicitly deferred this work.
- Current authority is clear enough to proceed to inventory:
  - the only real runtime metric-like surface today is the backend-owned `monitor/server-status` payload
  - current `trend` data is a process-local monitor read model, not a repository-wide metrics platform
  - existing generated types and monitor UI are consumers, not independent authority
- No runtime or contract edit is justified in Batch 1 because the gap is governance classification, not a proven implementation drift.

## Immediate Next Step

- Batch 2 should inventory:
  - monitor plugin trend fields and any other metric placeholders in `server/internal/**` and `server/plugins/**`
  - OpenAPI fields that already expose metric-like semantics
  - frontend monitor cards or shared governance cards that might imply stronger metric authority than the backend actually provides
- Batch 3 may implement a bounded MVP slice only if that inventory proves one missing authority repair or one minimal canonical abstraction is needed.

## Batch 2 Inventory Result

- Runtime metric-like authority remains singular and bounded:
  - `server/plugins/monitor/plugin.go` owns the only real sampled metric-like runtime path in current repo scope
  - the monitor plugin starts a Redis-backed trend sampler during `Boot`, stores monitor-owned trend points under the monitor-specific storage key prefix, and serves them only through `monitor/server-status`
  - the sampled fields are process-local runtime and host snapshots such as CPU percent, memory percent, load average, goroutines, and Go runtime memory bytes
- Shared contract authority remains singular and bounded:
  - `server/plugins/monitor/contract/trend.go` owns the stable trend-window query semantics
  - `openapi/components/parameters/trend-range-query.yaml` and `openapi/components/schemas/server-status-trend*.yaml` own the cross-boundary wire shape for the monitor trend payload
  - generated `server/internal/contract/openapi/monitor/**` and frontend schema types remain derived artifacts only
- Frontend metric wording is presentation-only:
  - `web/src/modules/monitor/pages/overview/index.vue`
  - `web/src/modules/monitor/shared/server-status-snapshot.ts`
  - `web/src/modules/monitor/locales/*.json`
  - these files render cards, labels, focus modes, and refresh behavior, but they do not define runtime collection, sampling, retention, or label authority
- Other trend-like fields were inspected and are not accidental metrics authority:
  - `server/plugins/audit/**` plus `openapi/components/schemas/audit-overview-response.yaml` expose audit analytics fields such as `risk_groups`, `trend`, and `security_timeline`
  - those fields are audit-domain read models for event aggregation, not repo-wide metrics authority and not reusable monitor metrics contracts
- Repo-wide metric placeholder governance still lives only in `ai-plan/design/日志治理开发规范.md`
  - no canonical metrics emitter exists
  - no label policy exists
  - no retention or export strategy exists outside the bounded monitor trend payload
  - no web-owned fallback or dashboard contract was found

## Current Design Gap

- The repo currently has one backend-owned monitor trend surface, but no platform-level decision about whether future metrics should stay plugin-local, move into a shared backend metrics service, or remain documentation-only placeholders.
- The missing authority is not a broken consumer contract; it is the absence of a canonical metrics lifecycle definition for:
  - emitter ownership
  - label and cardinality rules
  - retention and aggregation policy
  - operator-facing consumers beyond the existing monitor page
- Because that gap is upstream governance rather than downstream drift, Batch 2 does not justify new runtime code, OpenAPI expansion, or frontend dashboard rollout.

## Batch 3 MVP Decision

- Batch 3 confirms there is no safe runtime, OpenAPI, or frontend MVP to implement inside current authority boundaries.
- The smallest justified MVP for this topic is doc-only closure:
  - keep `server/plugins/monitor/**` plus `openapi/paths/monitor.server-status.yaml` as the sole current metric-like authority
  - treat the monitor trend payload as a plugin-owned operational read model, not a repository-wide metrics platform
  - keep all other metric demand as `Metric Candidate / Metric Placeholder` only until a future authority owner is explicitly defined
- No current consumer drift was found that would justify an authority repair:
  - generated server/frontend contracts are already aligned with the monitor OpenAPI source
  - `web/src/modules/monitor/**` is only rendering backend-owned fields
  - audit overview trend fields remain audit analytics, not shared metrics semantics
- Batch 3 intentionally does not introduce:
  - a generalized `MetricsEmitter`
  - log-parsing-derived metrics
  - Prometheus, OpenTelemetry, Grafana, or exporter rollout
  - new dashboard or shared metrics DTO contracts outside the existing monitor payload

## Current Canonical Decision

- Current canonical metric-like runtime surface:
  - `server/plugins/monitor/**`
- Current canonical shared wire surface:
  - `openapi/paths/monitor.server-status.yaml`
- Current downstream-only consumer:
  - `web/src/modules/monitor/**`
- Current repo-wide metrics governance stance:
  - outside the bounded monitor plugin read model, metrics remain placeholder-only governance under `ai-plan/design/日志治理开发规范.md`
- A future non-placeholder metrics topic must first define:
  - canonical owner
  - allowed metric/label taxonomy
  - retention and aggregation semantics
  - operator-facing consumer contract
  - validation path for both `server` and `web` if shared surfaces are introduced

## Batch 4 Closeout

- Final archive-readiness check passed:
  - the doc-only MVP decision is fully recorded in this topic archive
  - the bounded monitor/OpenAPI authority remains explicit
  - narrow doc validation passed
  - broader backend/frontend validation remains honestly deferred because Batches 1-3 changed no runtime or shared contract surfaces
  - no further bounded implementation work is justified inside this topic
- Validation evidence:
  - `git diff --check`
  - `rg -n "metrics-governance" ai-plan/public`
- Deferred validation:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./plugins/user/... ./plugins/rbac/... ./plugins/audit/...`
  - `cd web && bun run check`
  - Deferred reason: this topic closed as doc-only governance work and did not modify runtime, OpenAPI source inputs, or frontend implementation surfaces.

## Closeout

- Topic status: `archived`
- Archive reason:
  - the topic completed its authority discovery, inventory, and final MVP decision without finding any justified runtime, OpenAPI, or frontend repair
  - the truthful outcome is a bounded doc-only closure: current metric-like authority stays limited to `server/plugins/monitor/**` plus `openapi/paths/monitor.server-status.yaml`, and broader metrics remain placeholder-only governance
- Follow-up status: `new-topic-only`
- Archive notes:
  - future non-placeholder metrics work must open a new bounded topic instead of reopening this archive line
  - any future topic must first define emitter ownership, metric/label taxonomy, retention and aggregation semantics, and operator-facing consumer contracts before implementation starts
  - logs, audit analytics, and frontend presentation remain non-authority evidence for metrics unless a future authority owner changes that contract
