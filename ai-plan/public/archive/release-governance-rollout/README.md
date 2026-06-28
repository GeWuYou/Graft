# Release Governance Rollout

本主题已完成 `v0.1.0 P0` release governance 落地顺序的 docs-only 收口，并已 `archive-ready`。

## 当前状态摘要

- 状态：`archive-ready`
- 任务分类：`cross-boundary`
- 本主题不实现 release workflow、`server/**`、`web/**` 或部署资产。
- 本主题的 live authority 已固定到：
  - `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`
  - `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
  - `ai-plan/design/release/build-identity-contract.md`
  - `ai-plan/design/release/migration-policy.md`
  - `ai-plan/design/release/config-policy.md`
  - `ai-plan/design/release/versioning-policy.md`
  - `ai-plan/design/release/support-boundary.md`
  - `ai-plan/design/release/upgrade-policy.md`

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：root `AGENTS.md` + `ai-plan/design/release/**` + `README.md` + `server/internal/cli/{serve,migrate,validate}.go` + `server/internal/config/**` + `web/package.json` + `ai-plan/public/archive/release-readiness-governance-audit/README.md`

## Phase Outcome

- Phase 1：Release Safety Governance
  - 固定 migration forward-only / backup / rollback governance
  - 固定 config compatibility / deprecation / rename governance
  - 固定最小 upgrade safety baseline
- Phase 2：Release Identity And Policy
  - 固定 `BuildInfo` / future `graft version` 最小 contract
  - 固定 release policy / support boundary
  - 固定 `server` / `web` / migration 的版本协同口径
- Phase 3：Release Authority Baseline Alignment
  - 将 release governance 开发约束 authority 收口到 `ai-plan/design/release/**`
  - 明确 operator-facing install / configuration / upgrade / rollback / versioning docs 当前阶段 `deferred`

## Archive-Readiness Check

- Phase 1/2/3 authority 已内部一致。
- `ai-plan/design/release/**` 已覆盖：
  - upgrade safety boundary
  - migration governance details
  - configuration lifecycle
  - build identity visibility contract
  - versioning and compatibility
  - support boundary clarification
- 当前不存在清晰且安全的新增 bounded batch。
- 结论：
  - `release-governance-rollout` 已完成当前 topic 目标
  - 后续如继续推进，应新开实现型或 operator-docs 型 topic，而不是继续在本主题叠加

## Deferred Follow-Up

- operator-facing install / configuration / upgrade / rollback / versioning docs
- BuildInfo 注入与 `graft version` 真实实现
- release workflow / deployment automation / helper tooling

