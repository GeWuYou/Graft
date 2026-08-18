# 首页工作台与 Dashboard 贡献设计

## 1. 目标

首页是全局运维工作台，不是模块卡片目录。它应让管理员先回答“现在是否需要处理什么”，再进入健康确认、近期活动、资源状态和快捷操作。

设计目标：

- 模块数量增长时，首页高度和几何结构保持有界。
- 模块继续拥有事实、权限和 drill-down；首页拥有信息预算、排序、折叠和视觉层级。
- 缺少证据、局部数据源失败和已确认业务失败必须保持不同语义。
- 正常状态提供确认但不与异常争夺视觉注意力。
- 先在现有 Dashboard Registry 与 wire contract 内扩充模块事实并完整消费 summary、widgets 和 container 数据面，
  再按独立阶段演进 typed attention 契约。

## 2. 当前 Authority

- `server/internal/dashboard` 定义结构化 widget registry、权限过滤和聚合。
- Core 与业务模块注册 health、alert-list、stat-group 等事实贡献。
- `openapi/**` 是 Dashboard HTTP wire contract authority。
- `web/src/modules/dashboard` 消费 summary，并独立编排容器资源摘要和菜单派生快捷入口。
- 当前模块不注册任意 Vue 组件；问题在于 Web 将贡献 metadata 直接投影为独立卡片和页面几何。

生产数据链保持：

```text
module/core contribution facts
  -> dashboard registry and authorization
  -> OpenAPI dashboard summary
  -> dashboard presentation policy
  -> fixed workbench regions
```

容器资源和快捷入口在当前版本仍是独立数据面。容器资源动作只能进入 canonical
`/infrastructure/docker/containers/**` 列表或对象详情；隐藏的 `/infrastructure/docker/containers/resources`
没有稳定管理对象或导航身份，必须删除且不保留 alias、redirect 或兼容菜单。生产收敛前不得由页面按文案或 widget
ID 推断新的服务端事实。

## 3. Presentation Ownership

首页长期拥有以下 presentation 责任区域：

1. operational status
2. attention
3. system health
4. module coverage
5. operational metrics and contextual links
6. recent activity
7. resources
8. quick actions

这些责任区域不是要求始终渲染八张卡片。贡献者不得决定首页列宽、卡片数量或首屏位置。`size`、`category` 和
`priority` 可继续服务现有 renderer，但不是新工作台几何 authority。

首屏最多四个主要 surface，并按 operational status、attention、system health、module coverage 排列。桌面采用
12 列 8/4 结构；窄屏保持相同优先级转为单列。operational metrics、contextual links、resources、recent activity
和 quick actions 下移，快捷入口不得遮挡异常。

列表默认预算保持有界：Attention 展示前 5 条，Health 展示前 3 条，contextual links 展示前 6 条；超出预算的
内容进入可展开详情，而不是继续拉长首屏。没有真实 timeline contribution 时不渲染空 Recent Activity 卡片，也
不得用 summary、日志或任务状态伪造活动记录。

## 4. 状态与证据语义

页面 presentation status 为闭合集合：

- `error`: 已确认且正在影响用户或运维流程的失败。
- `warning`: 已确认的降级、局部能力失败或观测源失败。
- `unknown`: 没有可信检测结果、尚未检测或证据过期。
- `info`: 正常事件、禁用、不适用或中性说明。
- `healthy`: 已有可信证据证明正常。

Evidence state 独立描述证据质量：

- `confirmed`
- `source-failed`
- `missing`
- `not-applicable`

禁止从 widget loader status、display state 或 ordering priority 直接推导 presentation status。具体映射：

- 页面 summary 整体请求失败：页面级 `error`。
- 单个可选贡献 loader 失败：默认 `warning + source-failed`，保留重试。
- 5xx：`error`。
- 4xx 聚合和慢请求：`warning`。
- 尚未执行的能力检测：`unknown + missing`。
- disabled：默认 `info`，只有 owner 证明其影响必需能力时才升级。
- PostgreSQL/Redis 的健康证据：`healthy + confirmed`，低视觉权重呈现。

排序为 `error > warning > unknown > info > healthy`。Attention 通常只接收前三类；健康项折叠进安静摘要。

## 5. 预览边界

Phase 1 通过 development-only `/mock/dashboard-preview` 验证信息架构。预览使用直接声明的固定 presentation scenario，不伪造生产 widget，也不调用生产 API。

该路由：

- 不进入服务端 menu、permission 或 OpenAPI。
- 不替代 `/`。
- 不成为第二套长期 shell 或 UI baseline。
- 在正式首页采用获批设计并完成同等验收后删除。

## 6. 生产演进

### 6.1 现有契约内的权威扩充

第一轮内容扩充是 cross-boundary authority repair：业务模块通过现有 Dashboard Registry 增强或新增结构化事实，
Dashboard service 负责权限过滤与有界装载，`web/src/modules/dashboard` 只负责 presentation projection。该阶段继续
复用当前 Dashboard summary、既有 widget payload types、容器摘要/实时状态和 menu-derived quick actions，不新增
HTTP、OpenAPI、permission、cache 或依赖。

模块贡献边界固定为：

- monitor 与 audit 在各自现有 health / alert-list contribution 内补足真实摘要、有限明细和 canonical drill-down；
  不把趋势存储、审计日志或页面推导结果复制进 Dashboard authority。
- announcement 通过现有 timeline widget 表达真实公告记录；没有符合权限和查询条件的真实记录时返回空 timeline，
  页面不得生成占位活动。
- backup 通过现有 health widget 表达 module-owned 备份健康事实，并把动作指向 canonical 备份管理入口。
- runtime-target 通过现有 stat-group widget 表达目标总数与 owner-defined 运行状态计数，并把动作指向 canonical
  `/infrastructure/runtime-targets` 或对应对象详情。
- 每个 contribution 继续声明既有 permission requirement；Dashboard service 必须先完成权限过滤，再执行 loader 和
  汇总 source coverage。

同一 summary 请求中的 widget loaders 采用固定最多 4 个并发 worker，共享请求 deadline；单源失败保持隔离并保留
既有 `priority -> order -> id` 确定性结果排序，不得因并发完成顺序改变页面顺序。Dashboard registry definition 属于
startup-immutable 数据，loader runtime fact 属于 `no cache`；不得新增 Redis、本地 TTL 或第二套 snapshot authority。

- module coverage 只读取 `system_summary` 的已注册、已启用和降级模块计数；页面不得扫描 menu、widget ID 或模块
  文案重新推断模块运行状态。
- source coverage 只统计服务端权限过滤后返回的 widgets 及其 loader status；未授权贡献源不能进入分母、失败数或
  可见明细。该 coverage 表示首页数据源装载情况，不等同于业务健康严重度。
- `stat-group` 投影为运行指标组，用于展示失败、运行中、禁用等 owner-defined metrics；不得把每个 metric 伪造成
  Attention 事件。
- `link-list` 投影为当前事实附近的 contextual links，默认展示 6 条后折叠；它与 menu-derived quick actions 的
  展示、点击计数和排序完全隔离。
- resources 继续使用现有 container dashboard summary，并把首页投影限制为运行/异常容器、CPU/内存、Top 3 CPU、
  Top 3 memory 和最多 5 条 anomaly。请求失败、无权限、无采样和已确认异常必须保持独立证据语义；所有列表或对象
  动作进入 canonical container 列表/详情，不得扩展或复活隐藏 resources 聚合页。

页面局部 presentation 类型可以为这些区域提供闭合集合和有界列表，但必须从生成的 OpenAPI payload 类型收窄，
不得复制 wire DTO 或创建兼容映射。交互优先使用 TDesign Vue Next 的 Card、List、Collapse、Tag 等既有组件，所有
可见文案进入 Dashboard 中英文 locale catalog，颜色、间距和响应式状态使用 TDesign/Graft token。

### 6.2 后续 typed attention 演进

正式演进优先采用 additive contract：

- summary 增加服务端生成时间、coverage 和 typed attention summary。
- attention item 携带稳定 ID、来源、presentation status、evidence state、时间、影响和服务端授权 action。
- loader 的 payload、展示控制、汇总指标和 attention facts 分离，停止解析 payload 魔法字段。
- 聚合在权限过滤后完成，采用有整体预算的受限并发和确定性排序。
- 容器模块自行贡献 attention facts；Web 不从容器摘要反推跨模块严重度。

这一演进属于后续独立阶段。现有契约内容扩充不得提前引入局部 attention DTO、第二个 summary endpoint 或以 Web
推断代替服务端事实 authority。后续显式 source coverage 字段也必须保持“权限过滤后统计”的同一语义。

个性化只有在产生真实持久化需求后才引入 `server/modules/dashboard`。在此之前，`server/internal/dashboard` 保持运行时贡献聚合边界。

## 7. 验收约束

- 固定场景不为视觉效果虚构 error。
- unknown 不使用 warning/error 文案或色彩。
- healthy 不使用高权重绿色卡片、边框或填充 Tag。
- 首屏最多四个主要 surface，内部优先使用行分隔。
- Attention、Health 和 contextual links 的默认预算分别为 5、3、6，超额内容可在有界 surface 内展开；resources
  固定为 Top 3 CPU、Top 3 memory 和最多 5 条 anomaly，超额资源只进入 canonical 列表/详情，不在首页无限展开。
- 桌面 8/4 与窄屏单列使用同一信息优先级，不能在窄屏把 Module Coverage 或异常详情移到快捷入口之后。
- 无真实 timeline 时不显示 Recent Activity 空卡，不为填充页面伪造记录。
- announcement timeline 只呈现服务端真实记录；空结果隐藏 Recent Activity，loader 失败呈现 source-failed 证据而
  不是虚构公告。
- 主题颜色来自 TDesign/Graft token；light、dark、brand 与 acrylic 设置均保持可读。
- development preview 与正式首页使用同一 presentation surface；内容扩充完成并通过正式页人工视觉接受后，才删除
  preview 路由、固定 scenario 和专用 locale namespace。
- 浏览器截图是 inspection evidence，视觉接受仍由人完成。
