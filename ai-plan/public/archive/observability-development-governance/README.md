# Observability Development Governance

## Status

- Topic: `observability-development-governance`
- Status: `archived`
- Loop mode: `topic-completion-loop`
- Worktree: `feat/wt-audit-plugin-mvp`
- Branch: `feat/observability-development-governance`
- Task class: `cross-boundary`
- Started: `2026-05-29`
- Closed: `2026-05-29`

## Goal

一次性完成三段式治理闭环：

- Phase A: `logging-development-standard`
- Phase B: `logging-compliance-rollout`
- Phase C: `audit-console-governance-ux`

Hard order：

- 必须先完成 Phase A，再进入 Phase B
- 必须先完成 Phase B，再进入 Phase C

## Recovery Inputs

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/public/README.md`
- archived `ai-plan/public/archive/request-correlation-access-logging/**`
- archived `ai-plan/public/archive/logging-unification-rollout/**`
- archived `ai-plan/public/archive/plugin-audit-correlation-governance/**`
- `ai-plan/design/日志治理开发规范.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`

## Scope

- Owned scope:
  - `ai-plan/design/日志治理开发规范.md`
  - `ai-plan/public/archive/observability-development-governance/**`
  - `ai-plan/public/README.md`
  - Phase B inventory and bounded fixes under approved server/web authority paths
- Forbidden scope:
  - OpenTelemetry
  - Prometheus / Grafana / exporter rollout
  - fake metrics backend
  - repo-wide unrelated refactor
  - 把 audit log 当普通 app log

## Phase Status

- Phase A: `done`
- Phase B: `done`
- Phase C: `done`

## Phase A Acceptance

- `ai-plan/design/日志治理开发规范.md` completed
- topic tracking updated to mark Phase A done
- no runtime code changes required in this phase

## Phase B Acceptance

- inventory completed before any bounded code changes
- fix-now rollout stayed inside approved `server` authority paths
- bounded backend validation passed:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./plugins/user/... ./plugins/rbac/... ./plugins/audit/...`

## Phase C Acceptance

- frontend audit filters now support URL-driven governance queries including `requestId`, `traceId`, and `resourceId`
- audit logs now expose `requestId` / `traceId` visibility, copy actions, and canonical troubleshooting-id filtering
- audit logs now surface `actor`, `action`, `resource`, `result`, `reason`, and event-source distinction (`security event` vs `audit event`)
- RBAC role list, RBAC permission list, and user list now expose related-audit navigation entrypoints for operators with audit read permission
- frontend write-success and write-failure prompts in owned access-control scope now preserve correlation hints for operator troubleshooting
- fake overview risk-watch runtime content was removed; P2 analytics remain future scope
- bounded frontend validation passed:
  - `cd web && bun run check`

## Closeout

- Topic status: `archived`
- No additional batch is required for this three-phase loop.
- Future work, if any, should open a new bounded topic instead of reopening this governance loop.

## Post-Phase-C Follow-Up

- Follow-up: `audit-console-analytics-p2`
- Status: `archive-ready`
- Relationship:
  - this is a bounded cross-boundary follow-up opened after the original Phase C closeout
  - it reused this recovery directory as parent-topic evidence instead of reopening the original three-phase loop
- Canonical owner:
  - `server/plugins/audit/**` and `server/internal/audit/**` own the backend read model
  - `openapi/**` owns the shared HTTP contract shape
  - `web/src/modules/audit/**` is a downstream consumer only
- Accepted contract direction:
  - keep existing `summary.high_risk_events` as the canonical high-risk summary count
  - extend `/audit/overview` with a bounded grouped-risk analytics field
  - extend `/audit/overview` with a bounded server-computed trend field aligned to the existing `window` query (`24h` / `7d` / `30d`)
  - extend `/audit/overview` with a bounded `security_timeline` field for recent security-relevant events
  - extend `/audit/logs` with first-class `source` query semantics based on the existing backend `AuditSource` authority
- Explicit non-goals:
  - no general observability or metrics rollout
  - no frontend-derived fake analytics
  - no shell/layout redesign
  - no SOC-style detection platform or geo/IP profiling expansion
- Implemented result:
  - `/audit/overview` now exposes backend-owned `risk_groups`, `trend`, and `security_timeline`
  - `/audit/logs` now supports first-class `source` query semantics backed by the existing `AuditSource` authority
  - the audit overview and logs pages now consume those canonical backend fields inside the existing shell/layout
- Validation evidence:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./plugins/user/... ./plugins/rbac/... ./plugins/audit/...`
  - `cd web && bun run check`
  - `git diff --check`
- Closeout:
  - the bounded P2 follow-up is complete and now becomes part of this topic's archived evidence
  - further audit analytics expansion must open a new bounded topic instead of extending this recovered follow-up in place
