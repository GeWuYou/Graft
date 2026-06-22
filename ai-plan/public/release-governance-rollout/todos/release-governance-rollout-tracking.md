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
- `ai-plan/design/release/build-identity-contract.md`
- `ai-plan/design/release/migration-policy.md`
- `ai-plan/design/release/config-policy.md`
- `ai-plan/design/release/versioning-policy.md`
- `ai-plan/design/release/support-boundary.md`
- `ai-plan/design/release/upgrade-policy.md`
- `ai-plan/public/archive/release-readiness-governance-audit/README.md`
- `ai-plan/public/archive/release-readiness-governance-audit/todos/release-readiness-governance-audit-tracking.md`
- `ai-plan/public/archive/release-readiness-governance-audit/traces/release-readiness-governance-audit-trace.md`

## Current Recovery Point

- 已完成上游审计 topic archive handoff。
- 当前 active topic 只承接 `v0.1.0 P0` 治理落地顺序，不直接实现 release workflow。
- 当前批次 `phase-3-release-operator-docs-baseline` 已完成修正，authority 已收口到 `ai-plan/design/release/**`。
- 下一批固定为 `final-archive-readiness-check`。
- 剩余串行计划：
  - Final：archive-readiness check

## Task Checklist

- [x] 归档 `release-readiness-governance-audit`
- [x] 建立新的 active topic recovery 入口
- [x] 固定 loop mode、预算和 stop conditions
- [x] Phase 1：Release Safety Governance
- [x] Phase 2：Release Identity And Policy
- [x] Phase 3：Release Authority Baseline Alignment
- [ ] Final archive-readiness check

## Current Loop State

- `loop_mode`: `topic-completion-loop`
- `current_batch`: `none`
- `next_batch`: `final-archive-readiness-check`
- `remaining_after_current`:
  - `archive decision only`

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
  - 聚焦 release governance authority baseline
  - 不进入 docs-site、hosted docs 或 operator-facing 文档集合建设

## Phase 2 Decisions

- 已固定 release identity baseline：
  - official release identity 以仓库 Git tag `vMAJOR.MINOR.PATCH` 为唯一 authority
  - future `BuildInfo` baseline fields 固定为 `version`、`git_commit`、`build_time_utc`、`git_tree_state`
  - `BuildInfo.version` 使用 bare semver；release tag 继续保留 `v` 前缀
- 已固定 `graft version` 最小边界：
  - 当前仓库还没有 `version` subcommand
  - 后续实现必须是纯 metadata readout，不能依赖数据库、Redis、HTTP 启动或 migration 执行
  - release build 至少输出 `version`、`git_commit`、`build_time_utc`、`git_tree_state`
- 已固定 release policy / support boundary：
  - `v0.1.0` 只承诺一个 active repository release line
  - 不承诺 LTS、多 minor 并行维护或独立 `server` / `web` 官方发布节奏
- 已固定 version coordination：
  - 官方 `server` / `web` artifact 与 release notes 必须来自同一 release tag
  - migration version 是内部排序号，不是 product version，也不能用来替代 release compatibility

## Phase 3 Decisions

- 已固定 release governance authority canonical location：
  - `ai-plan/design/release/build-identity-contract.md`
  - `ai-plan/design/release/migration-policy.md`
  - `ai-plan/design/release/config-policy.md`
  - `ai-plan/design/release/versioning-policy.md`
  - `ai-plan/design/release/support-boundary.md`
  - `ai-plan/design/release/upgrade-policy.md`
- 已完成 coverage check，并把原本缺口收口到 release design authority：
  - `Upgrade Safety Boundary`
  - `Migration Governance Details`
  - `Configuration Lifecycle`
  - `Build Identity Visibility`
  - `Versioning And Compatibility`
  - `Support Boundary Clarification`
  - `Operator Documentation Mapping`
- 已固定 operator-facing 文档状态：
  - install / configuration / upgrade / rollback / versioning docs 当前阶段 deferred
  - `ai-plan/public/release-governance-rollout/operator-docs/**` 属于上一版同批次误落产物，已在本轮修正中移除
