# Container Dashboard Summary Realtime Upgrade Tracking

## Topic

Container Dashboard Summary Realtime Upgrade

## Scope

保留 `dashboard-summary` authority，增加 summary realtime contract，并通过 shared `ContainerStatsManager`
接入 Dashboard，完成容器概览布局升级、Skeleton/Empty State、资源语义统一和异常信息层级增强。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/容器Dashboard汇总与实时一致性升级设计.md`
- `ai-plan/design/容器资源状态与订阅治理设计.md`
- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/服务端API边界与兼容治理规范.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`

## Current Recovery Point

- Phase 0 已完成：
  - 当前设计 authority 已锚定到 `容器Dashboard汇总与实时一致性升级设计.md`
  - active topic / tracking / trace 已建立
- Phase 1 已完成：
  - backend 新增 canonical realtime topic：`container.dashboard.summary`
  - `GET /api/ops/containers/dashboard-summary` 保持为 Dashboard summary authority
  - summary realtime payload 已与 `dashboard-summary` 响应 shape 对齐，并补充 `collected_at`
  - anomaly items 已补充轻量原因字段：`status`、`reason_code`、`reason_label`
- Phase 2 已完成：
  - frontend 已新增 summary-domain realtime contract/parser
  - `ContainerStatsManager` 已持有 dashboard summary seed / subscribe / release / selector
  - Dashboard page 已改为通过 shared manager 消费 summary，而不是 page-local summary cache
- Phase 3 已完成：
  - Dashboard 容器区块已改为 unified `Top Resource Consumers`
  - 首次进入已使用 skeleton，不再闪 `0 / 0 / 0%`
  - 无运行容器时会显示 empty state 并隐藏无意义 progress 面
  - Dashboard 资源展示已改为 `N/A` / `Not Collected` 语义，并增强 anomaly cause 层级
- 当前确认事实：
  - Dashboard 容器概览已切到 shared stats manager + canonical summary topic
  - list/detail/dashboard 三类容器 stats 消费现在都走同一 frontend realtime owner
  - `dashboard-summary` authority 不应被前端 collection 聚合替代
- 当前状态：
  - topic 已完成 archive-readiness check，结论为 `archive-ready`
  - Phase 4 closeout 已完成；当前无剩余 pending batch

## Task Checklist

- [x] Phase 0：设计 authority 落盘
- [x] Phase 0：public topic / tracking / trace 建立
- [x] Phase 1：backend summary realtime topic 与 OpenAPI/contract 同步
- [x] Phase 2：frontend `ContainerStatsManager` summary 子域
- [x] Phase 2：Dashboard 页面改为 seed/acquire/select/release manager
- [x] Phase 3：`Top Resource Consumers` unified layout
- [x] Phase 3：Skeleton / Empty State
- [x] Phase 3：`N/A` / `Not Collected` / `Unavailable` 语义统一
- [x] Phase 3：异常卡片原因增强
- [x] Phase 4：validation / closeout / archive-readiness

## Batch Boundaries

- `phase-1-summary-authority-and-realtime-contract`
  - 范围：`openapi/**`、`server/modules/container/**`、`server/internal/contract/openapi/container/**`
  - 目标：保留 HTTP summary API，新增 canonical summary realtime topic，并补齐需要的 summary/anomaly reason fields
- `phase-2-frontend-summary-manager-integration`
  - 范围：`web/src/modules/container/**`、`web/src/modules/dashboard/pages/index.vue`
  - 目标：summary seed/acquire/select/release 全部进入 shared `ContainerStatsManager`
- `phase-3-dashboard-ux-upgrade`
  - 范围：`web/src/modules/dashboard/**`、必要的 `web/src/modules/container/locales/**`
  - 目标：重构 unified consumers 布局、Skeleton、Empty State、语义与异常卡片
- `phase-4-validation-and-closeout`
  - 范围：本主题 touched files
  - 目标：完成验证、主题 closeout 与 archive-readiness check

## Final Status

- 主题已完成全部计划批次并达到 `archive-ready`。
- 当前 live implementation 与 authority 结论：
  - `GET /api/ops/containers/dashboard-summary` 继续作为 Dashboard summary authority
  - canonical realtime topic `container.dashboard.summary` 已成为 summary 的唯一 realtime companion
  - Dashboard summary seed / acquire / release / selector 已全部进入 shared `ContainerStatsManager`
  - Dashboard unified `Top Resource Consumers`、skeleton、empty state、`N/A` / `Not Collected` 语义和 anomaly cause 层级均已落地
- Final validation evidence：
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
  - `git diff --check`
- 当前 owned scope 已完成验证，可按单主题切片独立提交。
