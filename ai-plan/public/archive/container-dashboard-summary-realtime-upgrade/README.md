# Container Dashboard Summary Realtime Upgrade

本 README 只承载 topic recovery、阶段边界和 archive-ready 判定，不是仓库规范正文。

稳定设计 authority 以 `ai-plan/design/容器Dashboard汇总与实时一致性升级设计.md` 为准。

## 当前状态摘要

- 当前主题目标是升级 `Dashboard` 容器概览的布局、realtime 一致性、资源语义和异常信息层级。
- 当前状态：`archive-ready`。
- 任务分类为 `cross-boundary`，涉及 backend summary authority、realtime contract、OpenAPI consumer 和 frontend dashboard/container module state architecture。
- Canonical design：`ai-plan/design/容器Dashboard汇总与实时一致性升级设计.md`。
- 推荐执行技能：`$graft-multi-agent-loop`，loop mode 默认 `topic-completion-loop`。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：`ai-plan/design/容器Dashboard汇总与实时一致性升级设计.md` + `server/modules/container/**` + `openapi/**` + `web/src/modules/container/**` + `web/src/modules/dashboard/**`

## Owned Scope

允许修改：

- `ai-plan/design/容器Dashboard汇总与实时一致性升级设计.md`
- `ai-plan/public/container-dashboard-summary-realtime-upgrade/**`
- `ai-plan/public/README.md`
- `openapi/**`
- `server/modules/container/**`
- `server/internal/contract/openapi/container/**`
- `web/src/modules/container/**`
- `web/src/modules/dashboard/**`

禁止误触：

- 不得废弃 `GET /api/ops/containers/dashboard-summary` authority。
- 不得把 Dashboard 长期改成基于前端 collection seed 的 TopN / totals 聚合页。
- 不得让 Dashboard 页面自己 new websocket。
- 不得在页面中维护第二套长期 summary cache。
- 不得把 stop/exited/dead 的资源值继续展示为 `Unavailable`。

## Phase Plan

- Phase 0：设计锚定、topic 持久化、loop recovery 资产建立。
- Phase 1：backend / OpenAPI summary realtime authority。
- Phase 2：frontend `ContainerStatsManager` summary 子域与 Dashboard 接入。
- Phase 3：Dashboard 容器区块重构、Skeleton / Empty State、异常原因信息层级。
- Phase 4：验证、closeout、archive-readiness check。

## Current Recovery Point

- 已完成设计决策：
  - 保留 `dashboard-summary` 作为 Dashboard summary authority。
  - 不采用 Dashboard collection seed。
  - Dashboard realtime 必须复用 shared stats manager，而不是 page-local websocket。
  - 热点榜单改为 unified `Top Resource Consumers`，不再沿用 CPU/Memory 双栏。
- 已完成 Phase 1：
  - backend/OpenAPI summary realtime authority 已就绪
  - `container.dashboard.summary` 已接入统一 realtime topic/ticket/gateway 体系
  - summary payload 已补充 `collected_at` 与 anomaly reason fields
- 已完成 Phase 2：
  - frontend `ContainerStatsManager` summary 子域与 Dashboard 页面接入
- 已完成 Phase 3：
  - Dashboard 容器概览已切到 unified `Top Resource Consumers`，并完成 skeleton / empty state / anomaly cause 层级升级
- 已完成 Phase 4：
  - completion-state validation 已基于当前工作树复跑通过
  - topic recovery docs 已与 live implementation 和验证事实对齐
- archive-ready 判定：
  - `dashboard-summary` authority 保持在 backend summary + canonical realtime topic
  - Dashboard 已通过 shared `ContainerStatsManager` 消费 summary，不存在 page-local websocket 或第二套长期 summary cache
  - 当前 owned scope 内未发现新的 cross-boundary drift
  - 当前主题无剩余 pending batch；当前 owned scope 已完成验证并可独立提交

## Validation Targets

```bash
cd server && go run ./cmd/graft validate backend
cd web && bun run check
git diff --check
```
