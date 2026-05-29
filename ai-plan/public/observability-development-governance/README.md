# Observability Development Governance

## Status

- Topic: `observability-development-governance`
- Status: `active`
- Loop mode: `topic-completion-loop`
- Worktree: `feat/wt-audit-plugin-mvp`
- Branch: `feat/wt-audit-plugin-mvp`
- Task class: `cross-boundary`
- Started: `2026-05-29`

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
- `ai-plan/design/契约治理与魔法值治理规范.md`

## Scope

- Owned scope:
  - `ai-plan/design/日志治理开发规范.md`
  - `ai-plan/public/observability-development-governance/**`
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
- Phase B: `pending`
- Phase C: `pending`

## Phase A Acceptance

- `ai-plan/design/日志治理开发规范.md` completed
- topic tracking updated to mark Phase A done
- no runtime code changes required in this phase
