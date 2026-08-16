# 首页工作台与 Dashboard 贡献设计

## 1. 目标

首页是全局运维工作台，不是模块卡片目录。它应让管理员先回答“现在是否需要处理什么”，再进入健康确认、近期活动、资源状态和快捷操作。

设计目标：

- 模块数量增长时，首页高度和几何结构保持有界。
- 模块继续拥有事实、权限和 drill-down；首页拥有信息预算、排序、折叠和视觉层级。
- 缺少证据、局部数据源失败和已确认业务失败必须保持不同语义。
- 正常状态提供确认但不与异常争夺视觉注意力。

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

容器资源和快捷入口在当前版本仍是独立数据面。生产收敛前不得由页面按文案或 widget ID 推断新的服务端事实。

## 3. Presentation Ownership

首页固定拥有六个 presentation 区域：

1. operational status
2. attention
3. system health
4. recent activity
5. resources
6. quick actions

贡献者不得决定首页列宽、卡片数量或首屏位置。`size`、`category` 和 `priority` 可继续服务现有 renderer，但不是新工作台几何 authority。

首屏采用 12 列 8/4 结构；窄屏按 Attention、Health、Activity、Resources 顺序单列。快捷入口保持低优先级，不遮挡异常。

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

正式演进优先采用 additive contract：

- summary 增加服务端生成时间、coverage 和 typed attention summary。
- attention item 携带稳定 ID、来源、presentation status、evidence state、时间、影响和服务端授权 action。
- loader 的 payload、展示控制、汇总指标和 attention facts 分离，停止解析 payload 魔法字段。
- 聚合在权限过滤后完成，采用有整体预算的受限并发和确定性排序。
- 容器模块自行贡献 attention facts；Web 不从容器摘要反推跨模块严重度。

个性化只有在产生真实持久化需求后才引入 `server/modules/dashboard`。在此之前，`server/internal/dashboard` 保持运行时贡献聚合边界。

## 7. 验收约束

- 固定场景不为视觉效果虚构 error。
- unknown 不使用 warning/error 文案或色彩。
- healthy 不使用高权重绿色卡片、边框或填充 Tag。
- 首屏最多四个主要 surface，内部优先使用行分隔。
- 主题颜色来自 TDesign/Graft token；light、dark、brand 与 acrylic 设置均保持可读。
- 浏览器截图是 inspection evidence，视觉接受仍由人完成。
