# Observability Development Governance Tracking

## Topic

- Topic: `observability-development-governance`
- Status: `active`
- Goal:
  - define the backend development standard for app log / audit / security event / metric placeholder
  - inventory and roll out bounded compliance fixes against current code
  - expose audit/security governance capability to frontend audit and access-control pages
- Recovery source:
  - `ai-plan/public/README.md`
  - archived `request-correlation-access-logging`
  - archived `logging-unification-rollout`
  - archived `plugin-audit-correlation-governance`
- Worktree: `feat/wt-audit-plugin-mvp`
- Branch: `feat/wt-audit-plugin-mvp`
- Task class: `cross-boundary`
- Loop mode: `topic-completion-loop`

## Startup Receipt

- Governance source: `root AGENTS.md`
- Task class: `cross-boundary governance loop`
- Recovery source: `archive-ready evidence`
- Authority summary:
  - `server/internal/logger/**` is the backend app/error logger authority
  - `server/internal/httpx/**` is the request correlation, access log, and HTTP security-event authority
  - `server/internal/audit/**` plus `server/plugins/audit/**` own audit persistence and audit field normalization
  - audit/security frontend consumption must follow backend authority instead of inventing client-side semantics

## Scope

- Owned scope:
  - `ai-plan/design/日志治理开发规范.md`
  - `ai-plan/public/observability-development-governance/**`
  - `ai-plan/public/README.md`
  - later Phase B/Phase C bounded files only after inventory
- Forbidden scope:
  - OpenTelemetry, Prometheus, Grafana, or full metrics rollout
  - generic log abstraction that erases `App Log / Access Log / Audit Event / Security Event` boundaries
  - repo-wide cleanup unrelated to governance findings

## Batch State

- Completed batches:
  - `phase-a-logging-development-standard`
- Pending batches:
  - `phase-b-logging-compliance-rollout`
  - `phase-c-audit-console-governance-ux`
- Current batch:
  - none
- Next batch:
  - `phase-b-logging-compliance-rollout`

## Phase A Notes

- Completed on `2026-05-29`.
- Established canonical classification:
  - `App Log`
  - `Access Log`
  - `Error Log`
  - `Audit Event`
  - `Security Event`
  - `Metric Candidate / Metric Placeholder`
- Confirmed correlation authority remains:
  - `requestId` from unified `httpx` middleware
  - `traceId == requestId` in MVP
  - actor and request metadata via canonical request context / audit normalization path
- Confirmed asynchronous rules are differentiated:
  - `App Log` and `Error Log` default sync
  - `Audit Event` async allowed with fallback error log
  - `Security Event` async bridge allowed with fallback error log

## Next Phase Entry Criteria

- Phase B must start with inventory only
- no code changes before inventory table exists
- every Phase B fix must map back to a clause in `ai-plan/design/日志治理开发规范.md`
