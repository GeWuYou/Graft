# Graft Responsive Manifest

本文件是开发者使用的响应式接入清单，运行时设计真值见 [Graft 响应式架构治理规范](../../ai-plan/design/governance/frontend/Graft响应式架构治理规范.md)。它不替代 `web/AGENTS.md`、设计规范或主题恢复材料。

## 新页面必做

- 使用 `ResponsivePage` 作为页面入口，并组合适用的 `ResponsiveHeader`、`ResponsiveToolbar`、`ResponsiveContent`、`ResponsiveEmpty`。
- 列表或查询页使用 `ResponsiveTable`；表格声明 `presentation="data"`、`presentation="entity"` 或受控的 `presentation="log"`。`log` 在 compact density 使用同一表格的卡片槽，业务页面不得自行切换第二份 Mobile 卡片布局。
- 表单使用 `ResponsiveForm`；短确认、详情、表单与 workspace 通过 `ResponsiveDialog` 的 `purpose` 和 `size` 表达语义，不传像素宽度。
- 在 `375`、`768`、`992`、`1200px` 以及窄 Drawer/Dialog/Split 容器验收。
- 在页面 profile 中记录页面类型、响应式组件、复杂 workspace 的 Mobile 降级和已批准例外。机器可读 manifest 位于 `docs/responsive/manifest.json`，由 `bun run responsive:governance:check` 消费，不进入生产 bundle。

## 运行时边界

```text
web/src/shared/responsive/
  breakpoints.ts
  container.ts
  responsive.ts
  tokens.ts
```

运行时 shared 仅包含以上类别的响应式基础设施及其共享组件/组合式函数。迁移状态、债务和例外登记属于治理数据，不能放进 `shared` 运行时包。

业务页面只声明 `density`、`layout`、`surface`、`presentation`、`interaction` 等语义 variant；组件根据自身容器选择布局。不要以设备名称或窗口像素驱动业务条件分支。

## 禁止项

`modules/**`、`app/**` 和页面实现中禁止新增：

- `window.innerWidth`、`document.body.clientWidth`、`screen.width`、`matchMedia`、`useMediaQuery`、`resize` 驱动的设备判断。
- `isMobile`、`mobile`、`phone`、`tablet`、user-agent 或 orientation 的业务布局分支。
- `if (isMobile)`、`v-if="mobile"`、`MobilePage.vue`、Desktop/Mobile 双页面。
- 页面级 `@media`、`@container`、任意本地 breakpoint 或固定像素 Dialog/Drawer 宽度。

唯一例外是 shared/平台基础设施中的 Monaco、ECharts、xterm 尺寸同步、Overlay 定位、动画几何和浏览器能力检测。例外必须进入治理清单并有 owner、原因、影响范围、替代项与清理触发条件。

## 验收口径

Desktop 是完整能力基线。Tablet 保持紧凑高密度操作。Mobile Friendly 至少保证浏览、监控、筛选、确认和简单操作；Monaco、YAML/JSON、Diff、终端与复杂 Compose 编辑在 Mobile 应安全降级为只读，不开发独立移动编辑器。

PR 必须通过响应式 manifest 校验和四宽度证据；作者仍须在 closeout 中报告例外或债务。浏览器证据的执行时机由任务验证范围决定。
