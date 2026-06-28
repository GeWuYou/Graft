# Release Readiness Governance Audit Tracking

## Topic

Release Readiness Governance Audit

## Scope

评估 `Graft` 距离“可长期维护的 `v0.1.0` 正式发布版”还缺哪些治理基础设施，并把后续治理落地工作拆成可执行批次。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `README.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- `ai-plan/design/architecture/项目设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `server/internal/config/config.go`
- `server/internal/cli/serve.go`
- `server/internal/cli/dev.go`
- `server/internal/cli/migrate.go`
- `server/internal/app/openapi_docs.go`
- `web/package.json`
- `web/src/utils/request.ts`
- `.releaserc.json`
- `.github/cliff.toml`
- `.github/workflows/release.yml`
- `.github/workflows/publish.yml`
- `.github/workflows/license-compliance.yml`

## Current Recovery Point

- 已完成非修改式仓库审计，覆盖：
  - 当前构建链路
  - 当前发布链路
  - 部署拓扑现状
  - 前端分发模型
  - Atlas migration 现状
  - 配置加载与校验机制
  - 版本注入缺口
- 已确认当前整体 readiness 更接近 `4/10`，不建议先动 release 流水线。
- 已确认需要先补的治理 P0：
  - BuildInfo / `graft version`
  - Release policy / checklist / upgrade notes 模板
  - Migration forward-only / backup / rollback policy
  - Config compatibility / deprecation / rename policy
  - Deployment support boundary
  - 最小安装 / 升级 / 回滚 / release docs
- 下一阶段建议通过多代理方式拆分治理落地设计，而不是直接实现流水线。
- Batch 4 docs-only closeout 已完成：
  - 已把当前 loop 的有效批次对齐为 Batch 4-6
  - 已明确 `v0.1.0` P0 只承诺 config compatibility / deprecation 文档治理，不承诺自动兼容桥接
  - 已把 deployment support boundary 与 documentation inventory 留给 Batch 5
- Batch 5 docs-only closeout 已完成：
  - 已明确 `v0.1.0` 的 deployment support boundary 只覆盖 documentation-first 的自管部署支持
  - 已明确 `v0.1.0` 不承诺 Docker/Compose、Kubernetes、托管平台矩阵或自动部署/升级/回滚工作流
  - 已形成 release/install/upgrade/rollback 所需的最小文档清单
  - 已把自动化 deployment support、部署资产与更强支持矩阵推迟到 `v0.2.x`
- Batch 6 docs-only closeout 已完成：
  - 已把本主题收口为显式 `P0/P1/P2` 路线，并固定 `v0.1.0` vs `v0.2.x` 边界
  - 已完成 archive-readiness check，当前主题状态为 `archive-ready`
  - 余下动作仅剩 topic 归档迁移与 active-topic index 清理，不再属于本主题内部继续分析的阻塞项

## Current Loop State

- current_batch：`Batch 6: Consolidated P0/P1/P2 roadmap and archive-readiness check`
- status：`archive-ready`
- next_batch：`none`
- remaining_after_current：`none`

## Consolidated Roadmap

### `P0` for `v0.1.0`

- BuildInfo / `graft version` governance baseline
- Release policy / support boundary
- Migration forward-only / backup / rollback policy
- Config compatibility / deprecation / rename policy
- Deployment support boundary
- 最小 install / upgrade / rollback / release documentation baseline

### `P1` for early `v0.2.x`

- 受控的 release checklist / doc consistency / deployment validation automation
- 更强的 structured build metadata 与 operator-facing version introspection
- config inventory / deprecation warning / controlled compatibility assistance
- upgrade / rollback helper tooling

### `P2` for later `v0.2.x+`

- Docker image / Compose / orchestration asset 官方支持
- Kubernetes / hosted-platform support matrix
- multi-node / zero-downtime rollout governance
- 更强的一键式 operator experience tooling

## Task Checklist

- [x] 建立 public topic 恢复入口
- [x] 完成第一轮 release readiness governance 审计结论
- [x] 形成 P0 / P1 / P2 治理缺口清单
- [x] 给出下一轮 `$graft-multi-agent-batch` / `$graft-multi-agent-loop` 提示词

## Historical Planning Batches

- [ ] Batch 1：Version + Build Governance 落地设计
- [ ] Batch 2：Release + Documentation Governance 落地设计
- [ ] Batch 3：Migration + Upgrade Governance 落地设计

## Current Loop Checklist

- [x] Batch 4：Config Compatibility / Deprecation Governance
- [x] Batch 5：Deployment Support Boundary + Documentation Inventory
- [x] Batch 6：Consolidated P0/P1/P2 roadmap and archive-readiness check
