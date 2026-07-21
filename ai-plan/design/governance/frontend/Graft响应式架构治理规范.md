# Graft 响应式架构治理规范

## 1. 定位与目标

Graft 是 Desktop 优先的 Self-hosted Application Platform。响应式的目标是让同一套 Vue 组件在不同可用容器宽度下保持可浏览、可监控和可完成简单应急操作的能力；不是为手机另建产品或业务流程。

- Desktop 保持完整管理与复杂编辑能力。
- Tablet 维持高密度操作，并在空间不足时自动采用紧凑布局。
- Mobile Friendly 支持浏览、监控、筛选、确认和简单操作。
- 响应式属于共享组件与样式基础设施，不属于业务页面的条件分支。

本规范与 [前端视觉设计规范](前端视觉设计规范.md)、[TDesign MCP 辅助开发规范](TDesign-MCP-辅助开发规范.md)、`web/AGENTS.md` 一起约束前端实现；其中 `web/AGENTS.md` 的目录所有权、TDesign、UnoCSS、i18n、验证及 Authority-first 规则仍为执行真值。

## 2. Non-goals

本规范不解决或不引入以下内容：

- 独立 Mobile UI、移动 App、移动路由或 Mobile API。
- Mobile First 视觉和业务流程。
- Desktop/Mobile 两套页面、组件或数据模型。
- 所有复杂功能都可在手机上完成。
- 为 Monaco、YAML/JSON 编辑、Diff、终端或复杂 Compose 工作流开发触控版编辑器。
- 因响应式迁移修改业务语义、权限、菜单、路由或后端契约。

## 3. Authority 与运行时边界

响应式能力的 canonical owner 是 `web/src/shared/**` 与 `web/src/style/**`。当前断点权威在 `web/src/style/variables.less`，全局壳层/页面 token 在 `web/src/style/layout.less`，页面宽度 surface 由 `web/src/layouts/components/PageContainer.vue` 和 `web/src/utils/route/meta.ts` 共同拥有；响应式迁移不得创建第二份页面宽度或 route surface authority。

```text
web/src/                       # B2 以现有目录为基础的目标形态
  shared/responsive/
    breakpoints.ts
    container.ts
    responsive.ts
    tokens.ts
  shared/components/responsive/
    ResponsivePage ... ResponsiveEmpty
  shared/composables/
    useContainerSize.ts
    useResponsiveVariant.ts
  style/responsive.less
```

- `shared/responsive/**` 只承载跨模块、无业务语义的运行时能力：断点语义、容器观测、variant 策略与 token 接口。
- `shared/components/responsive/**` 是响应式组件唯一实现层；`layouts/**` 只组合壳层组件，`modules/**` 只声明业务语义和组合页面。
- 后续 `style/responsive.less` 只能作为既有 `variables.less` / `layout.less` 的受控收敛入口，不能复制或替代其断点、页面宽度和 token 真值。UnoCSS 仍只用于辅助布局和少量原子样式，不构成第二套设计系统。
- 页面迁移状态、债务、例外登记是治理资产，不能打入生产运行包。机器可读 manifest 将位于 `scripts/responsive/` 或 `docs/responsive/`，而不是 `shared/responsive/`；本阶段不创建脚本或运行时清单。
- 共享组件只能包含无业务语义能力；业务字段、权限码、路由名、模块文案、DTO 与 API path 仍由原有 owner 持有。

## 4. 断点、容器与生命周期

### 4.1 规范断点

| 语义层 | 可用宽度 |
| --- | --- |
| Mobile | `< 768px` |
| Tablet | `768px - 991px` |
| Desktop | `992px - 1199px` |
| Wide | `>= 1200px` |

这些值必须继续从 `web/src/style/variables.less` 消费，不能由组件或页面重复定义。`1400px` 只可用于 Wide 内部的额外密度增强，不能创造第五种业务页面。组件优先根据自身容器而不是 window 宽度作出布局决定；窗口媒体查询只保留给壳层级别的全局布局。

### 4.2 生命周期

```text
Container Resize
  -> ResponsiveLayout
  -> ResponsiveVariant
  -> Responsive Component
  -> Business Component
```

- `ResizeObserver` 是容器变化的默认信号。业务组件不得获得或派生 `isMobile`、`isTablet`、`isDesktop`。
- 业务组件可消费语义化 `variant`，例如 `compact`、`comfortable`、`spacious`，但不按设备名分支。
- CSS Grid、Flex、Container Query 和媒体查询优先；只有共享基础设施无法表达的尺寸测量才允许 JS。
- Drawer、Split、Dialog 内的内容必须以容器为准，因为它们的可用宽度不等于 viewport。

## 5. Variant 体系

响应式组件只接受稳定语义，不接收像素宽度或设备布尔值。

| 维度 | 值 | 用途 |
| --- | --- | --- |
| Density | `compact` / `comfortable` / `spacious` | 信息密度、间距和工具栏排布 |
| Layout | `stack` / `flow` / `split` / `grid` | 内容关系与栅格行为 |
| Surface | `page` / `dialog` / `drawer` / `sheet` | 容器与交互表面 |
| Presentation | `data` / `entity` | 表格优先或实体卡片可选 |
| Interaction | `readonly` / `interactive` / `workspace` | 窄容器下允许的操作等级 |

业务页面只传递业务语义，例如 `presentation="data"`、`purpose="form"` 或 `interaction="workspace"`；shared 层依据容器与策略选择具体布局。禁止以 `width=900`、`mobile` 或 `isMobile` 作为组件公共接口。

## 6. Responsive Decision Tree

```text
页面
  -> ResponsivePage
     -> 列表或查询结果 -> ResponsiveTable
        -> presentation=data   -> 表格，窄容器横向滚动与列优先级
        -> presentation=entity -> shared 决定表格或 ResponsiveCardList
     -> 详情或概览 -> ResponsiveContent
     -> 表单或编辑 -> ResponsiveForm + ResponsiveDialog
     -> 代码、日志、终端、Diff -> Desktop Preferred 内容容器
```

新页面默认以 `ResponsivePage` 作为外层，并选择 `ResponsiveHeader`、`ResponsiveToolbar`、`ResponsiveContent`、`ResponsiveEmpty` 中适用的组件。列表页不得绕过 `ResponsiveTable`；页面不自行决定窄容器采用 Card、Table、Dialog 或 Drawer。

## 7. 组件职责与契约

| 组件 | 业务输入 | shared 层职责 |
| --- | --- | --- |
| `ResponsivePage` | 页面类型、密度 | 页面宽度、滚动边界、安全区与标准留白 |
| `ResponsiveHeader` | 标题、主/次操作 slots | 标题与操作区堆叠、动作溢出 |
| `ResponsiveToolbar` | 筛选、主操作、次操作、批量操作、溢出 slots | 换行、全宽筛选、保留主操作、次操作收进 Overflow Menu |
| `ResponsiveContent` | 内容与布局语义 | 标准节奏、信息栅格与窄容器内边距 |
| `ResponsiveTable` | `rows`、语义 columns、行操作、`presentation` | 容器测量、固定主列/操作列、列优先级、横向滚动、分页与 CardList 切换 |
| `ResponsiveCardList` | 同一行数据与实体摘要映射 | 仅用于 `entity` presentation 的移动呈现，不能形成第二个业务页 |
| `ResponsiveForm` | 字段组、语义列跨度、操作区 | 标签位置、Grid 到单列、控件宽度与底部操作区安全性 |
| `ResponsiveDialog` | `purpose`、`size`、slots、操作语义 | 在 Dialog、Drawer、Fullscreen、Bottom Sheet 间选择表面 |
| `ResponsiveLayout` / `ResponsiveSidebar` | 导航树与壳层 slots | Desktop 固定侧栏、Tablet 紧凑导航、Mobile Drawer、面包屑截断和 Tabs 溢出 |
| `ResponsiveEmpty` | 空、加载、错误态语义 | 紧凑且可访问的反馈布局，不重复页面级间距规则 |

`ResponsiveTable` 的 `data` presentation 永远保留表格语义，Mobile 允许横向滚动和辅助列隐藏；只有 `entity` presentation 可以切换为 `ResponsiveCardList`。每列必须声明可见性优先级，并确保状态、主标识和可操作入口不会同时消失。

`ResponsiveDialog` 的默认映射为：短确认/动作流在 Mobile 使用 Bottom Sheet；详情或短表单按可用空间使用 Dialog/Drawer；复杂表单在 Mobile 使用 Fullscreen；`workspace` 不承诺编辑能力，改为只读、复制、检索或引导至 Desktop。

## 8. 页面策略

- 表格：审计、日志、监控、RBAC、任务和资源运行数据属于 `data`；模板、目录、镜像、网络、卷和短实体选择列表可声明 `entity`。业务页面不能用 CSS 或 `v-if` 自行实现第二份卡片布局。分页列表的页面 surface、工具栏和滚动细则继续以 [分页列表页统一规范与收敛计划](分页列表页统一规范与收敛计划.md) 为准，本规范不复制或取代其列表页 authority。
- 表单：标签、表单列、Footer 和提交按钮由 `ResponsiveForm` 管理。复杂编辑保持 Desktop Preferred，Mobile 不强行压缩为可编辑代码工作台。
- 导航：Sidebar、Breadcrumb、Tabs、TopBar 与 Drawer 由壳层组件集中治理，页面不得自行侦测窗口来折叠导航。
- Monaco/YAML/JSON/Code Editor/Terminal/Diff：Desktop Preferred；Mobile 默认只读，保留查看、复制、搜索、下载或转到 Desktop 的安全路径。
- 图表：ECharts 等继续使用 `ResizeObserver` 更新实例尺寸；组件必须有最小高度、可访问 tooltip、窄容器图例收敛和轴标签降采样策略。

## 9. Design Token

在现有 TDesign/Graft token 和主题体系上补足以下语义 token，不新增平行色彩或组件库：

- `breakpoint`、`container-width`、`page-max-width`。
- `space`、`radius`、`typography`、`icon-size`、`touch-target`、`safe-area`。
- `header-height`、`toolbar-height`、`sidebar-width`、`dialog-size`、`content-gutter`。
- Density、Surface 和 Layout variant 对应的 token 映射。

禁止在新增页面中引入裸 `px` 宽度/高度、任意断点、任意边距/内边距或字号；确有固定格式需求时必须由 token 解释其语义。

## 10. Exception Policy

业务目录禁止新增 `window.innerWidth`、`document.body.clientWidth`、`matchMedia`、`useMediaQuery`、`screen.width`、`orientation`、user-agent/mobile/phone/tablet 检测、页面级 `@media` 或 `@container`。

允许的受控例外仅限 shared 或平台基础设施：Monaco、ECharts、xterm 的实例尺寸同步；Overlay/Popover 定位；动画几何；浏览器 API 能力检测。每一例外必须在未来的治理清单登记：owner、原因、影响范围、替代方案、清理触发条件；未登记不得合入。例外不得向模块页面暴露设备布尔值。

## 11. Responsive Debt

历史响应式债务必须进入治理清单并按迁移批次销账。清单字段固定为 `Issue`、`Owner`、`Deadline`、`Migration Phase`、`Reason`、`Replacement`。

| Issue | Owner | Deadline | Migration Phase | Reason | Replacement |
| --- | --- | --- | --- | --- | --- |
| 壳层 `min-width: 760px` | shell/shared | Phase 1 | 基础设施 | 阻断窄容器 | `ResponsiveLayout` 容器策略 |
| 固定像素 Dialog/Drawer | shared + consumers | Phase 2 | 公共组件 | 无 Mobile surface 策略 | `ResponsiveDialog` purpose/size |
| 页面本地 media query | 各模块 | Phase 3 | 页面迁移 | 断点分散 | shared variant 与容器规则 |
| 页面 overflow / 固定尺寸 | 各模块 | Phase 3 | 页面迁移 | 窄容器不可用 | ResponsivePage/Content token |

任何新增债务都必须有明确 owner、到期批次与替代项；“历史代码很多”不是豁免理由。

## 12. 编码、验收与 CI

- 新页面必须通过 `ResponsivePage` 进入，共享组件优先于页面级布局；禁止 `if (isMobile)`、`v-if="mobile"`、`MobilePage.vue` 或 Desktop/Mobile 双组件。
- 业务页面可选择语义 variant，不可自行读取 viewport/device，亦不可新增页面级断点 CSS。
- 未来 CI 的 `responsive:governance:check` 必须读取治理路径下的 manifest，而非运行时 bundle，并检查页面 profile、禁用 API、受控例外和未清偿债务。该门禁将在后续批次实现并接入 `bun run check`。
- 每个新页面及迁移页面必须在 `375`、`768`、`992`、`1200px` 验收；还要覆盖窄 Drawer、Dialog 与 Split 容器。
- 验收检查：无非预期横向溢出、工具栏与分页可操作、表格主信息可见、表单可填写、Dialog 有可用关闭/提交路径、图表不截断、workspace 在 Mobile 降级为安全只读。

## 13. 迁移路线

1. Phase 1：响应式 token、容器策略、壳层导航契约、规范、manifest 治理入口与高风险壳层债务。
2. Phase 2：以既有管理页模式收敛 `Responsive*` 公共组件、表格列策略、Dialog 语义和表单/工具栏规则。
3. Phase 3：迁移壳层、管理列表、表单/Dialog、Container/Project/Monitor 等复杂页面，删除对应局部响应式实现。
4. Phase 4：全量债务收口、CI 门禁、四宽度与容器级浏览器验收。

迁移必须最小化业务代码改动：保留数据、权限、路由和操作语义，只将布局决策上移到 shared 层。
