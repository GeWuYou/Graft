# Graft 响应式架构迁移路线

## 目标

在不改变业务模型、API、权限、菜单或路由语义的前提下，将 Graft 收敛为 Desktop 优先、Container-first、Mobile Friendly 的单一响应式架构。设计 authority 见 [Graft 响应式架构治理规范](../design/governance/frontend/Graft响应式架构治理规范.md)。

## Phase 1：基础设施

- 盘点并保留 `web/src/style/variables.less` 的 `768/992/1200/1400` 断点和 `layout.less` 的页面/密度 token authority。
- 在 shared 层建立容器尺寸、语义 variant 和 token 消费边界；修复壳层 `min-width: 760px` 与直接窗口宽度耦合。
- 建立治理路径下的例外/债务 manifest 结构，但不把迁移状态打入运行时包。

验收：无第二套 breakpoint/page-surface authority；壳层可在窄容器保留可用内容区；基础设施满足 `bun run check`。

## Phase 2：公共组件

- 从既有 `PageContainer`、管理列表、表格宽度观察、工具栏和表单模式收敛 `ResponsivePage`、`ResponsiveToolbar`、`ResponsiveTable`、`ResponsiveForm`、`ResponsiveDialog` 等共享 primitive。
- 固化 Table 的 `data/entity` presentation、列优先级、横向滚动、CardList 资格与 Dialog 的 `purpose/size` surface 策略。
- 建立 Monaco、ECharts、xterm、Overlay 的受控例外登记与共享尺寸策略。

验收：业务页不再自行决定设备布局；共享组件 API 只接受业务语义与 variant；受控例外有 owner 和清理条件。

## Phase 3：页面迁移

- 按壳层、管理列表、表单/Dialog、Container/Project/Monitor 和复杂 workspace 的风险顺序迁移。
- 保留 `paged-table`、`overview-dashboard`、`form-detail` 等 route surface authority；删除对应模块中的局部 media query、固定像素 overlay 和重复布局。
- 将复杂 workspace 在 Mobile 降级为只读，而不是开发第二套编辑器。

验收：每个迁移页面在 `375/768/992/1200px` 以及窄 Drawer/Dialog/Split 容器下无非预期溢出，并保留主操作与可访问关闭路径。

## Phase 4：全面检查

- 清偿 Responsive Debt，消除无批准例外的 viewport/device 检测和页面级 breakpoint。
- 实现并接入 `responsive:governance:check` 到现有 `bun run check`，让 manifest、禁用 API、例外和债务成为合并门禁。
- 对页面母版与高风险现有页面执行浏览器验收、键盘/焦点检查、主题与 locale 文案长度检查。

验收：CI 能阻止新增分散响应式实现；运行时与治理数据分离；Desktop 完整能力与 Mobile Friendly 边界均有可复现证据。
