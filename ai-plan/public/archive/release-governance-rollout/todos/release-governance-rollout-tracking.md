# Release Governance Rollout Tracking

## Topic

Release Governance Rollout

## Scope

把 `release-readiness-governance-audit` 的 `v0.1.0 P0` 审计结论拆成串行治理落地批次，先固化 authority，再决定是否开启实现型 topic。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `README.md`
- `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `ai-plan/design/release/build-identity-contract.md`
- `ai-plan/design/release/migration-policy.md`
- `ai-plan/design/release/config-policy.md`
- `ai-plan/design/release/versioning-policy.md`
- `ai-plan/design/release/support-boundary.md`
- `ai-plan/design/release/upgrade-policy.md`
- `ai-plan/public/archive/release-readiness-governance-audit/README.md`

## Final Recovery Point

- Phase 1：已固定 release safety baseline。
- Phase 2：已固定 release identity / policy baseline。
- Phase 3：已把 authority 收口到 `ai-plan/design/release/**`。
- Final archive-readiness check：已通过。
- 当前主题已归档，不再作为 active topic 继续推进。

## Task Checklist

- [x] 建立 active topic recovery 入口
- [x] Phase 1：Release Safety Governance
- [x] Phase 2：Release Identity And Policy
- [x] Phase 3：Release Authority Baseline Alignment
- [x] Final archive-readiness check

## Exit Criteria Result

- [x] Phase 1/2/3 authority 一致
- [x] release design authority 已稳定
- [x] operator-facing docs 已明确 deferred
- [x] 无新增 bounded batch
- [x] topic 可转入 archive

