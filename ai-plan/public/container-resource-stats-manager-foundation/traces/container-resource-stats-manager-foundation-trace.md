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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-audit-and-design-anchor",
    "phase-1-resource-ownership-separation",
    "phase-2-frontend-stats-manager-foundation"
  ],
  "pending_batches": [
    "phase-3-subscription-manager-unification",
    "phase-4-dashboard-shared-resource-consumption",
    "phase-5-history-store-optional"
  ],
  "current_batch": "phase-2-frontend-stats-manager-foundation",
  "next_batch": "phase-3-subscription-manager-unification",
  "closeout_status": "active"
}
```
