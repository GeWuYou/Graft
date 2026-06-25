# Container Dashboard Summary Realtime Upgrade Trace

## 2026-06-25

- 启动本主题并确认 task class 为 `cross-boundary`。
- 审计结论：
  - Dashboard 容器概览仍通过 `getContainerDashboardSummary()` 直接落页面本地 `ref`
  - list/detail 已接入 shared `ContainerStatsManager`
  - 当前 Dashboard 不具备 realtime 自动更新能力
- 否决旧方案：
  - 不废弃 `dashboard-summary` authority
  - 不采用 Dashboard collection seed
  - 不让 Dashboard 页面自己维护第二套 realtime/cache
- 形成新的 authority 决策：
  - 保留 `GET /api/ops/containers/dashboard-summary`
  - 新增 summary realtime topic
  - 由 shared `ContainerStatsManager` 持有 summary subscription 与 selector
- 形成新的 UX 决策：
  - CPU/Memory 双栏改为 unified `Top Resource Consumers`
  - 首次加载必须用 skeleton
  - 无运行容器时显示 `No running containers.`
  - stopped/exited/dead 等不适用资源值统一显示 `N/A`，并与 `Not Collected` / `Unavailable` / `Unknown` 的 canonical taxonomy 对齐
- Phase 1 完成：
  - 新增 canonical realtime topic：`container.dashboard.summary`
  - container module realtime issuer 已注册 summary topic，并复用统一 websocket ticket/gateway
  - stats collector 现在会从既有采样链发布 dashboard summary snapshot
  - OpenAPI source 已将 `collected_at` 标记为 required；generated contracts 仍需保持同一契约版本同步
  - anomaly items 已补充 `status`、`reason_code`、`reason_label`
- 验证补充：
  - `cd server && go test ./modules/container -run 'Test(BuildContainerDashboardSummary|ServiceDashboardSummaryUsesRuntimeList|IssueContainerDashboardSummaryRealtimeSubscription|ContainerStatsPublishedUsesOpenAPIResourceJSONShape|ContainerDashboardSummaryPublishedUsesRealtimeSummaryShape)'` 通过
  - `cd server && go run ./cmd/graft validate backend` 通过
  - `cd web && bun run check` 在主工作树复跑通过
  - `git diff --check` 通过
- 语义审计补充：
  - Dashboard 现有 `Unavailable` fallback 与 container module 的 `Not Collected` / `Not Running` / runtime error 语义不一致
  - Phase 3 应复用现有 container module 资源语义，而不是发明第三套 dashboard 词汇
- Phase 2 完成：
  - frontend realtime contract 已新增 `container.dashboard.summary`
  - `ContainerStatsManager` 已扩展 dashboard summary 子域，负责 seed / acquire / release / selector
  - Dashboard page 已改为 HTTP seed -> manager -> selector，并复用 shared realtime owner
  - focused frontend tests 通过，完整 `cd web && bun run check` 通过
- Phase 3 准备结论：
  - `DashboardContainerResources` 当前双栏布局并未被现有测试强绑定
  - 若 Phase 3 先只改展示层，保留 `hotspots.cpu` / `hotspots.memory` contract shape，可显著降低改动成本
  - `selectContainerStatsChangeState()` 可复用为 dashboard 数值变化高亮基础，不需要第二套动画状态系统
- Phase 3 完成：
  - `DashboardContainerResources` 已切到 unified `Top Resource Consumers` cards
  - loading 态改为容器 overview / consumers / anomalies 的 skeleton surfaces
  - no-running 场景已显示 `No running containers.` 并隐藏无意义 consumer progress
  - stopped/paused 等不适用资源值已展示为 `N/A`，未采集场景改为 `Not Collected`
  - anomaly cards 已使用 `reason_code` / `reason_label` / status / restart count 形成更清晰原因层级
  - focused component test 通过，`cd web && bun run check` 通过，`git diff --check` 通过
- Phase 4 final validation and closeout：
  - 重新执行 completion-state validation baseline：
    - `cd server && go run ./cmd/graft validate backend`
    - `cd web && bun run check`
    - `git diff --check`
  - 以上命令均基于当前工作树通过。
  - 复核当前 touched scope，未发现新的 cross-boundary authority drift：
    - summary authority 仍保持在 `GET /api/ops/containers/dashboard-summary` + canonical realtime topic `container.dashboard.summary`
    - frontend 仍通过 shared `ContainerStatsManager` 消费 summary，不存在 page-local websocket 或第二套长期 summary cache
    - unified `Top Resource Consumers`、skeleton / empty state 和 anomaly cause hierarchy 与设计文档保持一致
  - 当前 owned scope 已完成验证，可按单主题切片独立提交。
  - Topic terminal verdict：`archive-ready`

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-design-anchor-and-topic-persistence",
    "phase-1-summary-authority-and-realtime-contract",
    "phase-2-frontend-summary-manager-integration",
    "phase-3-dashboard-ux-upgrade",
    "phase-4-validation-and-closeout"
  ],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
