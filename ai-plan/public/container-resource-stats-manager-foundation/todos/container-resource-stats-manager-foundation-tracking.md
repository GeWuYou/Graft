# Container Resource Stats Manager Foundation Tracking

## Topic

Container Resource Stats Manager Foundation

## Scope

基于当前 `Graft` 容器模块建立统一容器资源状态层，明确 metadata / stats / subscription authority，并让
`List / Detail / Dashboard` 共享同一份容器资源状态。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/容器资源状态与订阅治理设计.md`
- `ai-plan/design/服务端API边界与兼容治理规范.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`

## Current Recovery Point

- Phase 0 已完成：
  - 已完成 Arcane 与 Graft 资源数据流审计
  - 已落盘仓库级设计 authority：`ai-plan/design/容器资源状态与订阅治理设计.md`
  - 已建立 active topic recovery 入口
- 当前确认事实：
  - backend authority 主链为 `statsCollector -> resourceStatsCache -> container.stats:{id}`
  - list/detail HTTP `resource` 当前是 latest-known snapshot 投影，但前端仍未把它收口为共享 state seed
  - detail 页已有 realtime patch 逻辑；list 页仍以 HTTP row.resource 为主
  - dashboard 当前未接入容器资源共享状态
- 当前推荐下一批：
  - `phase-1-resource-ownership-separation`

## Task Checklist

- [x] Phase 0：Arcane / Graft 资源数据流审计
- [x] Phase 0：设计 authority 文档落盘
- [x] Phase 0：public topic / tracking / trace 建立
- [ ] Phase 1：resource ownership separation
- [ ] Phase 2：frontend ContainerStatsManager foundation
- [ ] Phase 3：subscription manager unification
- [ ] Phase 4：dashboard shared resource consumption
- [ ] Phase 5：optional history store

## Batch Boundaries

- `phase-1-resource-ownership-separation`
  - 范围：`server/modules/container/**`、`server/internal/contract/openapi/container/**`、必要的 generated consumer
  - 目标：明确 metadata/stats authority，补齐 `ContainerResourceSummary.collected_at`，将 HTTP `resource` 定义为 seed snapshot
- `phase-2-frontend-stats-manager-foundation`
  - 范围：`web/src/modules/container/**`
  - 目标：建立 `Metadata Store`、`Stats Store`、selectors，停止页面局部直接 patch resource 的长期模式
- `phase-3-subscription-manager-unification`
  - 范围：`web/src/modules/container/**`、必要的 `web/src/shared/realtime/**`
  - 目标：统一 acquire/release/ref-count/idle close
- `phase-4-dashboard-shared-resource-consumption`
  - 范围：`web/src/modules/dashboard/**`、`web/src/modules/container/**`
  - 目标：Dashboard 复用同一 stats state，不新增新 authority
- `phase-5-history-store-optional`
  - 范围：`server/modules/container/**`、`web/src/modules/container/**`、OpenAPI 增量
  - 目标：引入可选 history / trend metrics，latest state 与 history 分离
