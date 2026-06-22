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
  - 最后把前两批 authority 真正沉淀为最小 operator 文档集合

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
