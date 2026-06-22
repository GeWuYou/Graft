# Release Governance Rollout Tracking

## Topic

Release Governance Rollout

## Scope

把 `release-readiness-governance-audit` 的 `v0.1.0 P0` 审计结论拆成可执行的 `$graft-multi-agent-loop` 治理落地批次，先固化 authority、文档边界和实施顺序，再决定是否开启实现型 topic。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `README.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/design/数据库表设计与迁移规范.md`
- `ai-plan/design/服务端API边界与兼容治理规范.md`
- `ai-plan/public/archive/release-readiness-governance-audit/README.md`
- `ai-plan/public/archive/release-readiness-governance-audit/todos/release-readiness-governance-audit-tracking.md`
- `ai-plan/public/archive/release-readiness-governance-audit/traces/release-readiness-governance-audit-trace.md`

## Current Recovery Point

- 已完成上游审计 topic archive handoff。
- 当前 active topic 只承接 `v0.1.0 P0` 治理落地顺序，不直接实现 release workflow。
- 当前批次 `phase-1-release-safety-governance` 已完成。
- 下一批固定为 `phase-2-release-identity-and-policy`。
- 剩余串行计划：
  - Phase 2：BuildInfo / version / release policy governance
  - Phase 3：operator docs baseline

## Task Checklist

- [x] 归档 `release-readiness-governance-audit`
- [x] 建立新的 active topic recovery 入口
- [x] 固定 loop mode、预算和 stop conditions
- [x] Phase 1：Release Safety Governance
- [ ] Phase 2：Release Identity And Policy
- [ ] Phase 3：Release Operator Docs Baseline
- [ ] Final archive-readiness check

## Current Loop State

- `loop_mode`: `topic-completion-loop`
- `current_batch`: `none`
- `next_batch`: `phase-2-release-identity-and-policy`
- `remaining_after_current`:
  - `phase-3-release-operator-docs-baseline`

## Phase 1 Decisions

- 已固定 migration safety baseline：
  - live migration governance is forward-only
  - `graft serve` 不隐式迁移
  - 升级前必须验证数据库 backup/restore 能力
  - rollback 只承诺文档化 decision points，不承诺自动回滚
- 已固定 config compatibility baseline：
  - change class: `additive` / `default-change` / `rename` / `semantic-change` / `removal`
  - patch release 不允许静默 rename/removal/semantic change
  - minor release 的 rename/removal/semantic change 必须携带 release notes 和 upgrade notes
- 已固定 operator upgrade path 的最小口径：
  - backup readiness
  - config diff check
  - explicit migration step
  - post-upgrade verification
- 已修复 authority drift：
  - 补齐 `ai-plan/public/release-governance-rollout/startup-prompt.md`

## Batch Boundaries

- `phase-1-release-safety-governance`
  - 聚焦 migration/config/upgrade safety rules
  - 不进入 workflow、CLI helper 或 runtime compatibility bridge
- `phase-2-release-identity-and-policy`
  - 聚焦 `BuildInfo`、`graft version`、release policy、support boundary
  - 不进入 workflow 改造
- `phase-3-release-operator-docs-baseline`
  - 聚焦 operator-facing 文档最小集合
  - 不进入 docs-site 或 hosted docs 建设
