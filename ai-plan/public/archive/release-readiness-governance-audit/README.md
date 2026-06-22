# Release Readiness Governance Audit

## 当前状态摘要

- 当前主题目标是评估 `Graft` 距离“可长期维护的 `v0.1.0` 正式发布版”还缺哪些治理基础设施。
- 状态：`archive-ready`。
- 任务分类为 `cross-boundary`，涉及 version/build/release/migration/config/deployment/upgrade/documentation governance。
- 当前主题只做调研、审计、治理设计和 topic 持久化，不做 release 流水线改造。
- 本主题已完成 docs-only 审计、治理边界收口与路线分层；尚未执行 topic 归档迁移或 `ai-plan/public/README.md` active-topic 清理。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：root `AGENTS.md` + `README.md` + `server/internal/config/**` + `server/internal/cli/{serve,migrate,validate}.go` + `web/package.json` + `.github/workflows/release-related` + `ai-plan/design/AI任务追踪与恢复设计.md`

## Owned Scope

允许修改：

- `ai-plan/public/release-readiness-governance-audit/**`
- `ai-plan/public/README.md`
- 后续如进入治理落地阶段，再按切片决定是否扩展到 release docs / design docs

禁止误触：

- 本主题当前阶段不得修改 `server/**`、`web/**`、`.github/workflows/**`、Docker/Compose 资产、构建脚本或发布流水线实现。
- 本主题当前阶段不得创建“假装已经支持”的部署资产或 release policy 实现代码。
- 本主题当前阶段不得提交与治理审计无关的 opportunistic fixes。

## Audit Scope

- Version Governance
- Build Governance
- Release Governance
- Migration Governance
- Config Governance
- Deployment Governance
- Upgrade Governance
- Documentation Governance

## Current Recovery Point

- 已完成一轮仓库级非实现审计，确认当前仓库已具备部分 release 相关机制，但尚不具备“可长期维护的正式发布治理闭环”。
- 已确认现状更接近：
  - 独立 `server`
  - 独立 `web`
  - PostgreSQL + Redis
  - Atlas 显式迁移
  - 环境变量 / `.env` 配置
  - 手动 semantic-release 打 tag + tag 后 publish 雏形
- 已确认当前重点缺口不是 GitHub Actions 细节，而是治理基础设施：
  - BuildInfo / version identity
  - release policy
  - upgrade / rollback governance
  - config compatibility governance
  - deployment support boundary
  - release documentation baseline
- 本轮 loop worker 已完成 Batch 4 的 docs-only 治理设计，并把当前 loop 的有效批次重新对齐为 Batch 4-6：
  - Batch 4：Config Compatibility / Deprecation Governance
  - Batch 5：Deployment Support Boundary + Documentation Inventory
  - Batch 6：Consolidated P0/P1/P2 roadmap and archive-readiness check
- Batch 5 docs-only worker round 已完成：
  - 已明确 `v0.1.0` 只承诺文档支持的自管部署边界，不承诺 Docker/Compose、Kubernetes 或托管平台工作流
  - 已明确 `v0.1.0` 的 release/install/upgrade/rollback 最小文档清单
  - 已把 deployment support boundary 与文档缺口保留给 Batch 6 统一并入 P0/P1/P2 路线判断
- Batch 6 docs-only worker round 已完成：
  - 已把本主题审计与 Batch 4/5 结论收口为显式 `P0/P1/P2` 路线，并区分 `v0.1.0` 与 `v0.2.x`
  - 已完成 topic 级 archive-readiness check，结论为 `archive-ready`
  - 本主题后续如需继续推进，应新建治理落地 topic，而不是继续在本审计 topic 中追加实现工作

## Working Conclusions

- 当前不建议先重构 release 流水线。
- 应先补齐 version/build/release/migration/config/deployment/upgrade/doc 的治理规则与文档边界，再让流水线去实现这些规则。
- 当前官方部署模型的建议方向应保持为：
  - `server` 与 `web` 独立分发
  - 数据库迁移显式执行
  - Docker / Compose 作为后续阶段治理目标，而不是当前已承诺能力

## Consolidated P0 / P1 / P2 Roadmap

### `P0` for `v0.1.0`

以下内容属于 `v0.1.0` 正式发布前必须先明确的最小治理边界；本 topic 负责给出设计与文档 authority，不负责直接实现。

1. Version / Build identity baseline
   - 需要统一 `server` build identity 的治理口径，例如 `BuildInfo`、version string、commit/build time/dirty state 注入边界，以及 `graft version` 的最小输出目标。
2. Release policy / support boundary
   - 需要固定 `v0.1.0` 支持的发布关系、支持的部署形态、`server` / `web` / migration 的版本协同口径，以及明确不支持的自动化承诺。
3. Migration / upgrade / rollback governance
   - 需要明确 forward-only migration、升级顺序、备份前提、rollback 风险与 operator 决策点，避免把未落地的自动回滚能力误写成承诺。
4. Config compatibility / deprecation governance
   - 需要固定配置变更分类、patch/minor 的兼容约束、deprecation record 字段，以及 release notes / upgrade notes 的联动要求。
5. Deployment support boundary
   - 需要固定 `v0.1.0` 只支持 documentation-first 的自管部署，不承诺 Docker/Compose、Kubernetes、托管平台矩阵或自动化部署校验。
6. Minimum release documentation baseline
   - 需要补齐 release policy、install guide、config reference + compatibility notes、upgrade guide、rollback / recovery guide、release notes template。

### `P1` for early `v0.2.x`

以下内容应在 `v0.1.0` 治理边界稳定后进入下一阶段治理或实现 topic：

1. Controlled automation around release and deployment docs
   - release checklist automation
   - doc-to-implementation consistency checks
   - deployment validation automation
2. Stronger version/build ergonomics
   - structured build metadata injection
   - operator-facing version introspection beyond the minimal `v0.1.0` identity baseline
3. Controlled config compatibility assistance
   - startup deprecation warnings
   - machine-readable config inventory / schema export
   - controlled legacy key alias / rename bridge
4. Upgrade / rollback helper tooling
   - preflight helpers
   - operator-facing migration helpers
   - bounded recovery helpers that still preserve explicit operator control

### `P2` for later `v0.2.x+`

以下内容不应回流为 `v0.1.0` blocking item，且应在后续独立 topic 中按真实产品承诺重新评估：

1. Official Docker image / Compose / orchestration assets
2. Kubernetes or hosted-platform support matrix
3. More advanced rollout governance
   - multi-node rollout semantics
   - zero-downtime or blue-green expectations
4. Broader operator experience tooling
   - one-command install / upgrade / rollback
   - deeper deployment environment compatibility matrix

## Archive-Readiness Check

- `Batch 4` 与 `Batch 5` 的 docs-only 治理结论已被保留为 active topic 真值。
- 本主题的核心目标已经完成：
  - 已完成 release-readiness 审计
  - 已完成 `v0.1.0` 必需治理口径与 `v0.2.x` 推迟项收口
  - 已形成显式 `P0/P1/P2` 路线
  - 已明确本 topic 不进入 release workflow、Docker/Compose、`server` 或 `web` 实现
- 剩余工作属于新的治理落地或实现 topic，不再属于本审计 topic 的 archive 阻塞项。
- 结论：
  - 本 topic 作为 docs-only design/closeout topic 已 `archive-ready`
  - 仍待后续会话执行 topic 归档迁移与 active-topic index 清理

## Batch Numbering Reconciliation

- 当前 loop 状态以下列批次为准：
  - Batch 4：Config Compatibility / Deprecation Governance
  - Batch 5：Deployment Support Boundary + Documentation Inventory
  - Batch 6：Consolidated P0/P1/P2 roadmap and archive-readiness check
- 旧的 Batch 1-3 命名保留为早期规划上下文，不作为本轮 worker 的 pending batch 列表。
- 本主题当前 closeout、tracking 与 trace 一律以 Batch 4-6 的 loop 状态作为恢复真值。

## Batch 4 Governance Design

### Goal

- 为 `v0.1.0` 正式发布前的配置兼容、弃用、重命名和升级说明建立最小治理边界。
- 明确哪些要求必须先落成文档与发布口径，哪些能力推迟到 `v0.2.x`。
- 把部署支持边界和文档资产盘点留给 Batch 5，避免本批次重新扩张到部署实现或 release workflow 细节。

### v0.1.0 P0 Boundary

`v0.1.0` 必须先承诺文档级治理，不承诺自动兼容桥接实现。

- 必须为会影响安装、启动、迁移、认证、数据库、缓存、外部地址或构建行为的配置项建立 canonical owner 记录。
- 必须把每类配置变化标记为以下之一：
  - additive
  - default-change
  - rename
  - semantic-change
  - removal
- patch release 不允许静默删除、静默重命名或静默改变稳定配置项语义。
- minor release 若必须发生 `rename`、`semantic-change` 或 `removal`，必须同时提供：
  - release notes 条目
  - upgrade notes 条目
  - replacement 或迁移动作
  - 生效版本与最早移除目标版本
- 兼容 alias / fallback / 双读配置不是 `v0.1.0` 的默认承诺；只有在 authority 不能直接修复时才允许作为例外记录。
- 若确需例外兼容，至少记录：
  - canonical key
  - legacy key
  - why direct repair is deferred
  - affected consumers
  - expiry trigger
  - validation expectation

### Required Deprecation Record

`v0.1.0` 的 topic 级治理要求后续正式文档至少能承载这组字段：

- config key / config group
- canonical owner
- change class
- deprecated_in
- removal_target
- replacement
- operator action required
- release-notes required
- upgrade-notes required

这些字段当前只定义治理口径，不要求本批次落地到代码生成、CLI 输出或 CI 检查。

### v0.1.0 Non-Goals

- 不在本批次承诺自动 config rewrite tooling。
- 不在本批次承诺启动时 legacy key alias bridge。
- 不在本批次承诺 machine-readable config schema export。
- 不在本批次承诺 release workflow 自动校验弃用清单。

### v0.2.x Follow-Up Scope

以下能力推迟到 `v0.2.x` 讨论或落地：

- machine-readable config inventory / schema export
- config diff automation 或文档漂移检查
- startup deprecation warnings with structured metadata
- 受控的 legacy key alias / rename bridge
- 面向 operator 的 config migration helper

### Batch 5 Handoff Boundary

- Batch 5 只聚焦官方 deployment support boundary 与 release/install/upgrade 文档资产盘点。
- Batch 5 不应重写本批次已经确定的 config compatibility / deprecation 口径。
- 若 Batch 5 发现 deployment 文档要求额外配置分类，只能补充 inventory 视角，不能回退到“默认允许静默配置漂移”。

## Batch 5 Governance Design

### Goal

- 为 `v0.1.0` 定义官方 deployment support boundary，只承诺当前仓库能够真实支撑的 operator 支持面。
- 盘点 release/install/upgrade/rollback 所需的最小文档集合，但不实现 workflow、脚本、CI gate 或部署资产。
- 为 Batch 6 提供可直接归类为 P0 与 `v0.2.x` 的 release support 输入。

### `v0.1.0` P0 Deployment Support Boundary

`v0.1.0` 只承诺 documentation-first 的自管部署支持，不承诺自动化或平台分发体验。

- 官方支持的部署形态是：
  - `server` 与 `web` 独立构建、独立分发、独立部署
  - operator 自行准备 PostgreSQL、Redis、进程管理、TLS 与反向代理
  - 数据库迁移通过显式步骤执行，而不是依赖应用启动时隐式迁移
- 官方支持的发布支持语义是：
  - 仓库给出 release/install/upgrade/rollback 所需的文档口径
  - operator 依照文档手动执行安装、升级、回滚前检查与回滚后验证
  - 不把当前仓库尚未实现的自动流程伪装成“已支持能力”
- `v0.1.0` 不承诺：
  - Docker image、Docker Compose bundle、Kubernetes manifests
  - one-command install、one-command upgrade、one-command rollback
  - 零停机升级承诺
  - 托管云平台适配矩阵
  - 发布流水线自动验证部署可用性

### Deployment Boundary Interpretation

- deployment support boundary 的 canonical owner 仍是本 topic 的治理文档，而不是 `.github/workflows/**`、Docker/Compose 资产或未来脚本实现。
- Batch 4 的 config compatibility / deprecation 口径是 Batch 5 的前置输入：
  - deployment/install/upgrade 文档必须引用 canonical config key、change class 与 operator action required
  - Batch 5 不得重新定义“静默配置漂移是否允许”的规则
- `v0.1.0` 的 rollback support 只承诺：
  - 文档化的回滚前提
  - 文档化的 operator decision points
  - 文档化的数据风险说明
  - 文档化的最小验证步骤
- `v0.1.0` 不承诺自动数据库回滚、自动配置回滚或自动恢复脚本。

### Minimum Documentation Inventory For `v0.1.0`

在声称 `v0.1.0` 具备正式 release/install/upgrade/rollback 支持前，至少应存在以下文档资产：

1. Release Policy / Support Boundary
   - 说明支持的部署形态、明确不支持的形态、版本承诺边界，以及 `server` / `web` / database migration 的发布关系。
2. Install Guide
   - 说明先决条件、必需外部依赖、基本安装顺序、首次启动前检查，以及最小可用性验证。
3. Config Reference + Compatibility Notes
   - 以 Batch 4 的 config compatibility / deprecation 口径为 authority，覆盖稳定配置项、默认值风险、rename/removal 说明与 operator action。
4. Upgrade Guide
   - 说明支持的升级路径、升级顺序、迁移执行时机、配置变更检查点，以及升级后验证清单。
5. Rollback / Recovery Guide
   - 说明允许的回滚边界、禁止假设、数据库与配置风险、回滚前备份要求，以及回滚后验证步骤。
6. Release Notes Template
   - 至少固定版本摘要、breaking/deprecation change、operator action、upgrade note、rollback note 与 known issues 栏位。

### `v0.1.0` Non-Goals

- 不在本批次承诺 release checklist automation。
- 不在本批次承诺 deployment smoke environment。
- 不在本批次承诺 install/upgrade/rollback CLI helper。
- 不在本批次承诺 Docker/Compose 或其他封装部署资产。

### Deferred To `v0.2.x`

以下 deployment support 能力明确推迟到 `v0.2.x` 或更晚阶段讨论：

- Docker image / Compose / orchestration asset 的官方支持策略
- 更强的 deployment matrix，例如 process manager、proxy、OS 基线或托管平台细分支持
- 自动化的 upgrade preflight / rollback helper / deployment validation tooling
- 更细粒度的 multi-node、zero-downtime 或 blue-green style rollout governance
- 文档与实现状态的一致性自动检查

### Batch 6 Input

- Batch 6 应把本批次结论并入统一 P0/P1/P2 路线：
  - `v0.1.0` P0：文档化 deployment support boundary + 最小 release/install/upgrade/rollback 文档集合
  - `v0.2.x`：自动化部署支持、部署资产与更强支持矩阵
- 若 Batch 6 发现某项文档缺口阻塞 archive-readiness，应优先归类为文档治理缺口，而不是跳回实现流水线。

## Validation Targets

本主题当前阶段以仓库审计和文档一致性检查为主，不要求运行代码验证。

建议后续治理落地切片按变更边界选择验证：

```bash
cd server && go run ./cmd/graft validate backend
cd web && bun run check
git diff --check
```
