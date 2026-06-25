# Container Resource Stats Manager Foundation Trace

## 2026-06-24

- 启动任务并确认 task class 为 `cross-boundary`。
- 对照审计 Arcane 与当前 Graft 容器资源数据流，目标不再是功能盘点，而是 `Container Resource Data Flow` authority 收敛。
- Arcane 审计结论：
  - 容器列表 API 返回 metadata row，不携带 CPU / Memory。
  - stats 来自独立 WebSocket stream；列表页通过 `ContainerStatsManager` 持有容器 stats。
  - `containerstats.Store` 只承担 history ring buffer，未形成跨页面统一 latest-state authority。
  - Detail 与 Dashboard 未共享同一份 stats state。
- Graft 审计结论：
  - backend 已具备 `statsCollector -> resourceStatsCache -> container.stats:{id}` 主链。
  - `List` / `Detail` 都会从 backend cache 投影 `resource`。
  - frontend 详情页已有统一 realtime topic 消费，但状态仍停留在页面局部。
  - frontend 列表页仍直接使用 HTTP `row.resource`，未共享详情页的 realtime 资源状态。
- 形成设计决策：
  - metadata authority 保持在 container HTTP APIs
  - realtime stats authority 保持在 canonical topic `container.stats:{id}`
  - HTTP `resource` 降级为 seed snapshot
  - 前端新增 module-owned `ContainerStatsManager`
  - manager 由 `Metadata Store`、`Stats Store`、`Subscription Manager` 组成
- 新增仓库级设计 authority：
  - `ai-plan/design/容器资源状态与订阅治理设计.md`
- 新增 active topic：
  - `ai-plan/public/container-resource-stats-manager-foundation/README.md`
- Phase 1 authority repair：
  - 将 `ContainerResourceSummary.collected_at` 补回 `server/modules/container.ResourceSummary` 与 OpenAPI authority。
  - 将 HTTP `resource` 明确标注为 seed snapshot / latest-known projection，而不是前端最终 authority。
  - 保持 `statsCollector -> resourceStatsCache -> container.stats:{id}` 主链不变。
  - 修复 `CollectStatsSnapshots` 在 stale last-known snapshot 场景下错误使用新尝试时间作为 freshness 的漂移，改为沿用真实 snapshot `collected_at`。
- Phase 2 frontend stats-manager foundation：
  - 在 `web/src/modules/container/shared/stats-manager.ts` 建立 module-owned metadata/stats foundation。
  - list 页由 `seedContainerList -> selectContainerListViews` 读取 `resource`，不再把 `payload.items[].resource` 直接作为长期 authority。
  - detail 页由 `seedContainerDetail -> selectContainerDetailView -> applyContainerRealtimeStats` 组合 `resource`，不再用页面局部 merge/patch 保留 realtime 值。
  - `resourceStatus` 改为读取 canonical `resource.collected_at`，而不是 `inspect_updated_at`。
  - 定向 container tests 通过；全量 `bun run check` 在非本批 `src/modules/rbac/pages/index.test.ts` 既有失败处被阻塞。
- Phase 3 subscription manager unification：
  - 在 `web/src/modules/container/shared/stats-manager.ts` 收口 acquire / release / ref-count / idle-grace cleanup。
  - detail 页不再直接持有 `openRealtimeTopicSocket` 与 socket controller，realtime toggle 改为模块级订阅 authority 开关。
  - list 页开始为当前可见容器集合 acquire/release 订阅，并通过 selector 响应式读取共享 stats state。
  - list 刷新失败改为只清理 list metadata，不再 `resetContainerStatsManager()` 抹掉 detail 等仍在持有的 canonical stats authority。
  - container scoped vitest 通过。
- Phase 4 dashboard shared resource consumption：
  - `ContainerStatsManager` 增加 collection-key 隔离，允许 dashboard 拥有独立 metadata projection，同时继续共享同一份 stats authority。
  - 新增 `web/src/modules/container/contract/dashboard-stats.ts` 作为最小跨模块稳定消费面，dashboard 不再需要 ad-hoc 导入 container 私有实现。
  - dashboard 首页新增容器资源概览卡片，seed 来自 container list HTTP latest-known projection，realtime 仍由 canonical `container.stats:{id}` 驱动。
  - dashboard 释放订阅与清理 projection 时不影响 list/detail 仍持有的 container module authority。
- Phase 5 optional history store：
  - `ContainerStatsManager` 增加短时 ring buffer，latest snapshot 与 history snapshot 分离存储。
  - detail resources 区开始消费 module-owned history state，为后续趋势图保留 authority-safe 基座。
  - 本批未引入新的 server/OpenAPI/history API，也未把 history 升级成 dashboard/platform authority。
- Outer loop archive-readiness check：
  - confirmed no remaining pending batch in owned scope
  - confirmed phase-1 to phase-5 validation and scoped commits were already recorded
  - marked topic `archive-ready` without widening authority to new backend/shared history surfaces
- Post-archive incremental repair:
  - container module 新增 `ops.container.resource_stats.collect_interval_seconds`，默认值下调为 1 秒，collector 发布 cadence 仍从 module-owned runtime-hot config 读取
  - list/detail/dashboard 通过 `stats-manager` 派生 change state，对 CPU / 内存上涨/下降增加视觉强调
  - visual emphasis 仅消费 canonical topic 驱动的 latest stats，不新增 dashboard/shared/platform authority

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-audit-and-design-anchor",
    "phase-1-resource-ownership-separation",
    "phase-2-frontend-stats-manager-foundation",
    "phase-3-subscription-manager-unification",
    "phase-4-dashboard-shared-resource-consumption",
    "phase-5-history-store-optional"
  ],
  "pending_batches": [],
  "current_batch": "archive-ready",
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
