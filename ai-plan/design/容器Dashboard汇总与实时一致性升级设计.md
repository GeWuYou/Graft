# 容器 Dashboard 汇总与实时一致性升级设计

本文档定义 `Graft` 容器 Dashboard 升级的 canonical design，覆盖 Dashboard 汇总 authority、shared realtime
manager 接入方式、布局模型、资源状态语义和异常容器信息层级。

本文档是本主题的设计 authority，不替代：

- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/容器资源状态与订阅治理设计.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `ai-plan/design/architecture/前端架构设计.md`

## 1. 背景与问题

当前 `Graft` 容器模块已经具备：

- `GET /api/ops/containers/dashboard-summary`
- list 级 realtime topic：`container.stats.list`
- detail 级 realtime topic：`container.stats:{id}`
- 前端 `ContainerStatsManager`

但 Dashboard 容器概览仍存在两类问题：

1. 布局与信息层级问题
   - CPU TOP3 与 Memory TOP3 为两个独立栏目
   - 卡片高度不一致，导致对齐差、空白大、扫描路径长
2. 数据流与实时一致性问题
   - Dashboard 当前只消费一次 `dashboard-summary` HTTP 结果
   - 没有进入 shared realtime manager
   - 首次进入后 CPU 总量、内存总量、Top3、异常列表不会随 realtime 自动更新

本轮升级目标不是把 Dashboard 降级成前端 collection 聚合页，而是：

- 保留 Dashboard summary API 的 authority
- 通过 shared stats manager 接入 Dashboard summary realtime
- 不新增第二套 websocket owner
- 不新增第二套 page-local cache

## 2. Authority 决策

## 2.1 Dashboard Summary Authority

`Dashboard` 的容器概览 authority 保持在后端 summary 边界：

- `GET /api/ops/containers/dashboard-summary`
- 对应 realtime summary topic

Dashboard 不以前端全量 `ContainerSummary[]` 聚合结果作为 canonical authority。

原因：

- `Dashboard` 是 overview，不是 management list
- Top3、总 CPU、总内存、异常列表本质上都是聚合视图
- 容器规模增大时，让前端为 Dashboard 传全量 summary 再做 TopN 计算不合理

## 2.2 Shared Realtime Manager Authority

虽然 summary authority 保留在后端，但 `web` 运行时仍只允许一个 shared realtime owner：

- `web/src/modules/container/shared/stats-manager.ts`

新增或扩展后的 `ContainerStatsManager` 负责：

- Dashboard summary seed snapshot
- Dashboard summary realtime subscription lifecycle
- Dashboard summary selector
- Dashboard summary socket state

Dashboard 页面本身不得：

- 直接 new websocket
- 直接持有 topic controller
- 维护第二份长期 summary cache

## 2.3 不采用 Dashboard Collection Seed

本轮明确不采用：

- `seedContainerList(..., 'dashboard:...')`
- Dashboard 通过 collection 前端重算 totals / TopN / anomalies

原因：

- 这会让 Dashboard 越来越像 Container Management List
- 会把 summary authority 从 backend 下沉到 consumer
- 与 Dashboard 的 overview page type 不匹配

## 3. 目标数据流

### 3.1 HTTP Seed

首次进入 Dashboard：

```text
Dashboard page
  -> getContainerDashboardSummary()
  -> ContainerStatsManager.seedContainerDashboardSummary(summary)
  -> Dashboard selector
  -> UI
```

### 3.2 Realtime Update

首次 seed 之后：

```text
Dashboard page
  -> ContainerStatsManager.acquireContainerDashboardSummarySubscription()
  -> canonical realtime topic
  -> ContainerStatsManager.applyContainerDashboardSummaryRealtime(payload)
  -> Dashboard selector
  -> UI 自动更新
```

### 3.3 Realtime Topic

新增 canonical summary realtime topic：

```text
container.dashboard.summary
```

该 topic 是与 `GET /api/ops/containers/dashboard-summary` 对齐的 canonical summary companion channel。

topic payload 与 `dashboard-summary` API shape 对齐。

禁止：

- Dashboard 页面自己定义第二套 runtime-only payload
- 让 `web` 在 manager 外部再维护同义 summary adapter

## 4. 前端架构要求

## 4.1 Dashboard Page Ownership

Dashboard 页面仍位于：

- `web/src/modules/dashboard/pages/index.vue`

但其容器概览子区块必须通过 container module stable facade 消费：

- 页面只负责 seed / acquire / release
- 页面只读 manager selector
- 页面不解释 summary authority

## 4.2 Container Stats Manager 扩展

`ContainerStatsManager` 新增 Dashboard summary 子域：

- `seedContainerDashboardSummary(summary)`
- `clearContainerDashboardSummary()`
- `acquireContainerDashboardSummarySubscription()`
- `releaseContainerDashboardSummarySubscription()`
- `selectContainerDashboardSummaryView()`
- `selectContainerDashboardRealtimeState()`

必要时补充：

- summary `collected_at`
- summary freshness
- summary first-load state

这些都属于 container module-owned shared state，不提升到平台级 `web/src/shared/**`。

## 4.3 允许的 page-local state

Dashboard 页面允许保留的 page-local state 仅限：

- 是否已触发首次容器 summary 加载
- 当前 UI loading / skeleton 呈现
- 排名变化动画的 presentation state
- 更新时间相对文本的 display state

这些不是数据 authority。

## 5. 布局与信息层级

## 5.1 页面类型

该页面继续归类为：

- `overview-dashboard`

容器概览子区块要符合：

- console-first
- 高信息密度但不拥挤
- 状态清晰
- 扫描路径短

## 5.2 热点榜单布局

本轮不采用双独立栏：

- CPU TOP3
- Memory TOP3

推荐改成统一榜单：

- `Top Resource Consumers`

每个容器一张卡，卡片内同时展示：

- 容器名
- 生命周期状态 / 健康状态
- CPU 百分比 + progress
- Memory 百分比 + progress
- 可选 rank delta / 更新时间

原因：

- 用户无需左右来回对照
- 更符合 overview-dashboard 的单卡片扫描路径
- 后续扩展 Top5 / Top10 更稳定

## 5.3 Summary 区块

Summary 保留四项：

- running
- abnormal
- CPU total
- memory total

这些值都来自 backend summary authority，不在前端重算。

## 5.4 更新时间位置

容器概览区块必须有独立更新时间：

- `Updated 2s ago`
- `Collected at 2026-06-25 08:40:49`

页面顶部的总刷新时间不能替代容器 summary authority 时间。

## 6. Skeleton 与 Empty State

## 6.1 Skeleton

首次进入 Dashboard 时：

- summary cards 使用 skeleton
- Top Resource Consumers 使用 skeleton cards
- anomaly list 使用 skeleton rows

禁止先渲染假 `0 / 0 / 0%` 再跳真实值。

## 6.2 Empty State

当 `running_containers == 0`：

- 显示 `No running containers.`
- 隐藏热点 progress 区
- 不渲染空的 CPU/Memory 榜单骨架

若异常容器仍存在：

- anomaly 区块继续显示

当整个 summary 无任何容器数据：

- 显示区块级 empty state

## 7. 资源数值语义统一

资源显示必须统一为以下 taxonomy：

- `N/A`
  - 非运行中容器，资源统计不适用
- `Not Collected`
  - 运行中容器，但当前还没有可用采样快照
- `Unavailable`
  - 采集失败、权限失败、Docker API error、明确 stats error
- `Unknown`
  - authority 返回未知状态或无法识别值

禁止把 stopped/exited/dead 场景展示为 `Unavailable`。

## 8. 热点容器变化反馈

## 8.1 阈值色

资源热点阈值：

- `>= 70%`：warning
- `>= 90%`：danger

不仅使用文字或数值颜色，还应影响：

- progress status
- card accent

## 8.2 变化高亮

当 CPU 或 Memory 跨过 warning / danger 阈值时：

- 卡片边框或背景做 500-800ms 高亮
- warning 用黄色 glow
- danger 用红色 glow

普通小幅变化：

- 数值平滑过渡
- progress 缓动

## 8.3 排名变化

若榜单位置变化：

- 显示 `↑1` / `↓1` / `new`

这是 presentation state，不是 shared authority。

## 9. 异常容器信息层级

当前异常容器不应只显示：

- `Exited`

首层至少展示：

- 容器名
- 生命周期状态
- 异常原因标签
- 相对时间

推荐首层可见字段：

- `Exited`
- `Exit Code: 137`
- `OOMKilled`
- `3 minutes ago`

或：

- `Restarting`
- `Back-off`
- `5 retries`

折叠层再展示：

- restart count
- last exit code
- latest error summary
- image/runtime/source summary

如 summary DTO 当前缺少这些字段，应扩展 backend summary contract，而不是前端再调 detail API 拼装。

## 10. 后端契约要求

## 10.1 保留 API

保留：

- `GET /api/ops/containers/dashboard-summary`

它继续是 Dashboard summary 的 seed authority。

## 10.2 新增 Realtime Contract

新增 summary realtime topic：

- topic 名在 container contract 中统一定义
- payload shape 与 HTTP summary 对齐

## 10.3 DTO 扩展原则

如果需要支持更好的异常卡片原因表达，可以扩展 summary item，但必须满足：

- authority 先落 OpenAPI source
- generated artifact 同步更新
- 前端只消费 canonical fields

## 11. 验收标准

本主题完成后，至少满足：

- Dashboard 不再只停留在一次性 HTTP summary
- Dashboard 通过 shared stats manager 自动更新
- 页面不新增第二套 websocket owner
- 页面不新增第二套长期 summary cache
- Summary 四项、热点榜单、异常列表都能随 realtime 自动更新
- stopped/exited/dead 的资源值显示为 `N/A`
- 首次加载使用 skeleton，不再先渲染假 0 值
- 无运行容器时显示明确 empty state
- Top Resource Consumers 的信息层级和扫描路径优于原双栏布局
