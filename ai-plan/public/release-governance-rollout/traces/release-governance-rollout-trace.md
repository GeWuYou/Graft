# Release Governance Rollout Trace

## 2026-06-22 Phase 0 topic bootstrap

- 已确认 `ai-plan/public/release-readiness-governance-audit/**` 的 docs-only 审计主题已经 `archive-ready`，不应继续作为新的 loop 承载体。
- 已将 `release-readiness-governance-audit` 从 active topic 索引迁出，并迁入 `ai-plan/public/archive/` 作为上游审计证据。
- 新建 `release-governance-rollout` 作为后续 `v0.1.0 P0` 治理落地的 active topic。
- 固定后续 loop 为 `topic-completion-loop`，并采用串行单 worker round：
  - `phase-1-release-safety-governance`
  - `phase-2-release-identity-and-policy`
  - `phase-3-release-operator-docs-baseline`
- 已固定全局 guardrail：
  - 当前 topic 不直接实施 release workflow、Docker/Compose、Kubernetes、托管平台支持
  - 当前 topic 先收口 authority 和文档口径，再决定是否开实现型 topic

## Suggested Implementation Order

- 第一优先级：Release Safety Governance
  - 先固定 migration/config/upgrade safety governance，避免后续版本与文档治理建立在不稳定的 operator path 上
- 第二优先级：Release Identity And Policy
  - 再固定 `BuildInfo`、`graft version`、release policy 和 support boundary，避免文档引用不存在的 version contract
- 第三优先级：Release Operator Docs Baseline
  - 最后把前两批 authority 真正沉淀为 release governance 设计真值，而不是提前承诺 operator 文档集合

## 2026-06-22 Phase 1 release safety governance

- 已使用 `server/internal/cli/{migrate,serve,validate}.go` 与 `server/internal/config/config.go` 作为 authority
  evidence，确认当前仓库的真实运行边界是：
  - `graft migrate up` / `graft dev` 承担显式迁移入口
  - `graft serve` 保持纯运行时启动，不隐式迁移
- 已把 `v0.1.0` migration safety governance 固定为：
  - forward-only live migration governance
  - 升级前必须先验证 backup/restore readiness
  - rollback 仅承诺文档化的 operator decision points、data risk 与最小验证步骤
  - 不承诺自动数据库回滚、自动配置回滚或 helper tooling
- 已把 `v0.1.0` config compatibility governance 固定为：
  - `additive` / `default-change` / `rename` / `semantic-change` / `removal`
  - patch release 不允许静默 rename/removal/semantic change
  - minor release 的高风险 config 变更必须同时记录 replacement、removal target、operator action、
    release notes 与 upgrade notes
  - startup warning、alias bridge、config rewrite helper 保持 deferred
- 已把 operator upgrade path 的最小治理口径固定为：
  - 版本目标确认
  - backup/restore readiness
  - config diff 与 operator action 检查
  - 显式 migration step
  - post-upgrade verification
- 已修复 topic authority drift：
  - 补齐 `ai-plan/public/release-governance-rollout/startup-prompt.md`
  - 使 topic README 的 recovery 引用重新可用

## 2026-06-22 Phase 2 release identity and policy

- 已使用 `server/internal/cli/root.go`、`.github/workflows/release.yml`、`.github/workflows/publish.yml`、
  `web/package.json` 与 `README.md` 作为 authority evidence，确认当前仓库的真实 release identity 状态是：
  - 发布 tag 与 artifact 名称已经存在 authority
  - `server` 端尚无统一 BuildInfo 注入模型
  - `graft` 根命令当前还没有 `version` subcommand
- 已把 `v0.1.0` release identity baseline 固定为：
  - 官方 release identity 以 Git tag `vMAJOR.MINOR.PATCH` 为准
  - future `BuildInfo` 最小字段集固定为 `version`、`git_commit`、`build_time_utc`、`git_tree_state`
  - `BuildInfo.version` 使用 bare semver；release tag 保持 `v` 前缀
  - 在 BuildInfo / `graft version` 真实实现落地前，tag、artifact filename 与 release notes 共同构成当前
    operator-facing canonical identity
- 已把 future `graft version` 的最小 contract 固定为：
  - 纯 metadata readout
  - 不依赖数据库、Redis、HTTP 启动或 migration 执行
  - release build 至少暴露 `version`、`git_commit`、`build_time_utc`、`git_tree_state`
  - 本批次不把该命令误写成“已存在支持”
- 已把 `v0.1.0` release policy / support boundary 固定为：
  - 一个 active repository release line
  - 不承诺 LTS、多 minor 并行维护或独立 `server` / `web` 官方 release cadence
  - 官方 operator 支持包是同一 tag 下的 `server` artifact、`web` artifact 与 release notes
- 已把 `server` / `web` / migration version coordination 固定为：
  - `server` / `web` / release notes 必须来自同一 release tag
  - 混用不同 tag 的 `server` / `web` artifact 在 `v0.1.0` 下不受支持
  - migration version 仅是内部排序号，不是 product version 或 compatibility label

## 2026-06-22 Phase 3 release authority baseline correction

- 复核最新用户要求后，确认上一版 Phase 3 将 authority 误落到
  `ai-plan/public/release-governance-rollout/operator-docs/**`，不再满足当前 topic 约束。
- 本轮修正保持同一批次边界，不新开 phase，也不扩张实现范围；目标是把同批次错误方向收回到正确
  authority owner。
- 已新建 `ai-plan/design/release/**` 作为 Phase 3 live authority：
  - `build-identity-contract.md`
  - `migration-policy.md`
  - `config-policy.md`
  - `versioning-policy.md`
  - `support-boundary.md`
  - `upgrade-policy.md`
- 已把缺口维度分别固定到新的 design authority：
  - `Upgrade Safety Boundary`
  - `Migration Governance Details`
  - `Configuration Lifecycle`
  - `Build Identity Visibility`
  - `Versioning And Compatibility`
  - `Support Boundary Clarification`
- 已明确当前阶段的 operator-facing install / configuration / upgrade / rollback / versioning docs 为 deferred，
  不再把它们当作 live deliverable。
- 已删除上一版误落的 `operator-docs/**`，原因是这些文件属于同一 Phase 3 批次内错误 authority 落点的修正范围，
  而不是跨批次扩张或回滚他人独立工作。
