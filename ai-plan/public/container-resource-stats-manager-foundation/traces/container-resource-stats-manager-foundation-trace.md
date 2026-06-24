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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-audit-and-design-anchor"
  ],
  "pending_batches": [
    "phase-1-resource-ownership-separation",
    "phase-2-frontend-stats-manager-foundation",
    "phase-3-subscription-manager-unification",
    "phase-4-dashboard-shared-resource-consumption",
    "phase-5-history-store-optional"
  ],
  "current_batch": "phase-0-audit-and-design-anchor",
  "next_batch": "phase-1-resource-ownership-separation",
  "closeout_status": "active"
}
```
