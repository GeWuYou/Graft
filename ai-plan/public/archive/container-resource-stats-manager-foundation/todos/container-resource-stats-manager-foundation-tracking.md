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
- `ai-plan/design/domains/container/容器管理设计.md`
- `ai-plan/design/domains/container/容器资源状态与订阅治理设计.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`

## Current Recovery Point

- Phase 0 已完成：
  - 已完成 Arcane 与 Graft 资源数据流审计
  - 已落盘仓库级设计 authority：`ai-plan/design/domains/container/容器资源状态与订阅治理设计.md`
  - 已建立 active topic recovery 入口
- 当前确认事实：
  - backend authority 主链为 `statsCollector -> resourceStatsCache -> container.stats:{id}`
  - list/detail HTTP `resource` 是 latest-known snapshot 投影，phase-2 已将其收口为共享 stats seed
  - detail 页 realtime snapshot 已不再直接 patch 页面局部 `resource`，而是写入 module-owned stats store
  - list 页不再直接以 HTTP row.resource 作为长期 authority
  - dashboard 已通过 container-owned contract facade 接入共享资源状态
- 当前状态：
  - topic 已完成 archive-readiness check，结论为 `archive-ready`
  - 当前增量修复补齐了 1 秒采样默认值与 UI 变化强调，未引入新 authority owner

## Task Checklist

- [x] Phase 0：Arcane / Graft 资源数据流审计
- [x] Phase 0：设计 authority 文档落盘
- [x] Phase 0：public topic / tracking / trace 建立
- [x] Phase 1：resource ownership separation
- [x] Phase 2：frontend ContainerStatsManager foundation
- [x] Phase 3：subscription manager unification
- [x] Phase 4：dashboard shared resource consumption
- [x] Phase 5：optional history store

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

## Phase 1 Closeout

- 已完成 authority repair：
  - `server/modules/container/runtime.go` 为 `ResourceSummary` 增加 `CollectedAt`
  - `resourceStatsCache` 在 latest-known snapshot 入缓存时补齐 `collected_at`
  - `CollectStatsSnapshots` 在复用 stale last-known snapshot 时保留真实 freshness，而不是用新采样尝试时间伪装更新
  - OpenAPI `ContainerResourceSummary` 与 `ContainerSummary.resource` 注释明确 HTTP `resource` 是 seed snapshot / latest-known projection
- 本批未进入：
  - `ContainerStatsManager`
  - 订阅引用计数
  - Dashboard 共享消费

## Phase 2 Closeout

- 已完成 module-owned `ContainerStatsManager` foundation：
  - 新增 `shared/stats-manager.ts`
  - list 页通过 manager seed/selectors 读取 `resource`
  - detail 页通过 manager seed/selectors + realtime apply 读取 `resource`
  - `resourceStatus` 的 freshness 显示改为读取 canonical `resource.collected_at`
- 已删除 phase-1 前的页面局部长期 authority 逻辑：
  - 移除 `mergeDetailStructurePreservingRealtimeResource`
  - 移除 `applyRealtimeResourceToDetail`
- 本批未进入：
  - `Subscription Manager` acquire/release/ref-count
  - Dashboard 共享消费
  - 平台级 shared stats authority 上提

## Phase 3 Closeout

- 已完成 module-owned `Subscription Manager`：
  - `web/src/modules/container/shared/stats-manager.ts` 新增 `acquire/release/selectContainerStatsRealtimeState`
  - list/detail 共享 `container.stats:{id}` socket lifecycle、ref-count 与 idle-grace close
  - detail 页 realtime toggle 改为消费模块级 subscription authority，而不是直接持有 socket controller
  - list 页刷新失败只清理当前 list metadata，不再 `resetContainerStatsManager()` 抹掉其它页面仍持有的 stats authority
- 已完成验证：
  - container scoped vitest 通过
- 本批未进入：
  - Dashboard 共享消费
  - history / trend metrics

## Phase 4 Closeout

- 已完成 dashboard shared resource consumption：
  - `web/src/modules/container/contract/dashboard-stats.ts` 暴露 container-owned dashboard consumption facade
  - `web/src/modules/container/shared/stats-manager.ts` 引入 collection-key 隔离，让 dashboard metadata projection 与 list projection 分离
  - `web/src/modules/dashboard/pages/index.vue` 通过 facade seed/acquire/select/release 容器资源概览，不新增 dashboard 自有 stats authority
- 已完成边界修复：
  - dashboard 不再需要直接导入容器页面或容器模块私有类型实现细节
  - container module 继续拥有 canonical stats authority，dashboard 仅消费 facade
- 本批未进入：
  - history / trend metrics

## Phase 5 Closeout

- 已完成 optional history store：
  - `web/src/modules/container/shared/stats-manager.ts` 增加 module-owned history ring buffer
  - detail resources 区新增 history 消费区块，latest 与 history 分离读取
- 已保持 authority 边界：
  - 未新增 server/OpenAPI authority
  - 未将 history 提升为 dashboard/platform shared authority

## Terminal Closeout

- archive-ready 判定通过：
  - 已完成 Phase 1 到 Phase 5 的 authority / frontend / dashboard / history 收口
  - phase-1 至 phase-5 均已完成对应 scoped commit
  - 当前无剩余 pending batch，也无新的 in-scope authority gap
  - 当前增量修复保持 archive-ready，不重开新的 batch
