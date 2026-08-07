# 主题预设治理审计报告

- 审计日期：2026-08-07
- 范围：`THEME_PRESET_DEFINITIONS`、主题状态、Personalization Workbench、全局样式、相关测试与 locale 展示面
- 结论：**不符合“预设仅为可写初始配置”的治理目标。** 预设 ID 仍参与运行时 token 合成；8 个深色预设还注入了当前用户无法在高级编辑器中写入的图表和 Material token。

## 权威模型

当前实际链路为：

```text
Preset ID
  -> selectedThemePresetId
  -> buildResolvedThemeTokens()
  -> brand tokens -> base tokens -> preset.tokenOverrides
  -> preset.materialTokenOverrides（受 preserveThemePersonalization 与首启特例控制）
  -> 用户排版/密度/圆角/阴影派生 token -> themeTokenOverrides
  -> stylesheet、chartColors、root attributes、CSS -> 实际渲染
```

`selectedThemePresetId` 在 [web/src/store/modules/setting.ts](web/src/store/modules/setting.ts#L301) 被持续解析为预设对象，并在 [web/src/utils/theme-workbench.ts](web/src/utils/theme-workbench.ts#L35) 参与每次 token 合成。因此它不是“最近应用的预设”元数据，而是视觉 authority。

## 手工入口

| 配置字段 | Personalization Workbench 入口 | 状态 |
| --- | --- | --- |
| `mode`、`brandTheme` | 外观：模式、品牌色 | 可写 |
| `fontFamilyPreset`、`fontSizePreset` | 排版：字体、字号 | 可写 |
| `radiusPreset`、`shadowPreset`、`densityPreset` | 风格：圆角、阴影、密度 | 可写 |
| `layout`、菜单分栏、固定侧栏、折叠菜单、页签 | 布局：导航行为 | 可写；受布局前置条件约束 |
| 页头、面包屑、页脚、亚克力 | 布局/外观：元素与视觉效果 | 可写；路由禁用页脚时无可见效果 |
| 已登记语义 token | 高级：Token 编辑器 | 可写 |
| 图表与 Material 缺失 token | 无 | **不可写** |

## 预设矩阵

缩写：`S` 为 `system/standard/standard/standard/standard`；`R` 为 `source-han-sans/standard/rounded/standard/comfortable`；`Side`/`Mix` 为共享布局 patch；`P` 为已定义的常规 token；`C4` 为 4 个未定义图表 token；`M9` 为 9 个未定义 Material token。

| 预设 | authorityPatch / stylePatch | Token 与实际视觉差异 | 手工入口 | 100% 可复现 | 隐藏权威 | 风险 |
| --- | --- | --- | --- | --- | --- | --- |
| `industrial-yellow` | system/small/square/hard-offset/compact；Side、页头/页脚关闭、亚克力关闭 | light/dark `P`，含 neo 色彩与硬阴影 | 风格、布局、高级 Token | 是 | 预设 ID 仍持续供给 token | 中 |
| `tdesign-default` | `S`；Side | 无独立 token | 外观、排版、风格、布局 | 是 | 同上 | 中 |
| `tencent-cloud` | inter/standard/business/standard/standard；Mix | light `P` | 同上及高级 Token | 是 | 同上 | 中 |
| `mountain-green` | `R`；Side | 无独立 token | 外观、排版、风格、布局 | 是 | 同上 | 中 |
| `midnight-blue` | harmonyos/standard/standard/floating/comfortable；Mix | 无独立 token | 外观、排版、风格、布局 | 是 | 同上 | 中 |
| `graphite-slate` | inter/small/business/flat/compact；Side、页签/自动折叠开启 | 无独立 token | 外观、排版、风格、布局 | 是 | 同上 | 中 |
| `sunset-amber` | `R`；Side | 无独立 token | 外观、排版、风格、布局 | 是 | 同上 | 中 |
| `ocean-teal` | harmonyos/standard/standard/floating/standard；Mix | 无独立 token | 外观、排版、风格、布局 | 是 | 同上 | 中 |
| `frost-silver` | system/large/capsule/flat/comfortable；Side | 无独立 token | 外观、排版、风格、布局 | 是 | 同上 | 中 |
| `violet-haze` | `R`；Side | light `P` | 同上及高级 Token | 是 | 同上 | 中 |
| `signal-rose` | inter/small/business/standard/compact；Mix | dark `P` | 同上及高级 Token | 是 | 同上 | 中 |
| `forest-night` | harmonyos/standard/rounded/flat/comfortable；Side | dark `P` | 同上及高级 Token | 是 | 同上 | 中 |
| `amethyst-night` | system/standard/standard/floating/standard；Mix | dark `P` | 同上及高级 Token | 是 | 同上 | 中 |
| `one-dark-pro` | `S` + comfortable/floating；Mix、亚克力开启 | dark `P + C4`，`M9` | 仅 `P`、亚克力开关 | 否 | preset ID、C4、M9、保留个性化开关 | 高 |
| `atom-one-dark` | `S` + floating；Mix、亚克力开启 | dark `P + C4`，`M9` | 仅 `P`、亚克力开关 | 否 | 同上 | 高 |
| `material-oceanic` | `S` + comfortable/floating；Mix、亚克力开启 | dark `P + C4`，`M9` | 仅 `P`、亚克力开关 | 否 | 同上 | 高 |
| `github-dark` | `S` + compact；Side、页签开启、亚克力关闭 | dark `P + C4`，仍携带 `M9` | 仅 `P`、亚克力开关 | 否：完整配置快照不可手写 | 同上 | 高 |
| `dracula` | `S` + rounded/floating；Mix、亚克力开启 | dark `P + C4`，`M9` | 仅 `P`、亚克力开关 | 否 | 同上 | 高 |
| `nord` | `S` + flat；Side、亚克力关闭 | dark `P + C4`，仍携带 `M9` | 仅 `P`、亚克力开关 | 否：完整配置快照不可手写 | 同上 | 高 |
| `tokyo-night` | `S` + floating；Mix、亚克力开启 | dark `P + C4`，`M9` | 仅 `P`、亚克力开关 | 否 | 同上 | 高 |
| `catppuccin-mocha` | `S` + rounded/floating；Mix、亚克力开启 | dark `P + C4`，`M9` | 仅 `P`、亚克力开关 | 否 | 同上 | 高 |

“是”仅表示现有公开字段足以叠加出等价可见效果；所有行仍受“预设 ID 持续参与合成”的总体 authority 违规影响。对于 8 个深色预设，“否”是严格的配置可重建结论：即使 `github-dark` 与 `nord` 当前关闭亚克力，预设仍包含无法手写的 Material state。

## Token 权威审计

`THEME_TOKEN_DEFINITIONS` 共 59 项。直接写在预设对象内的 27 个 token 都已定义，Industrial Yellow 的 `--graft-neo-*` 也已有可编辑入口。

缺失且仅由深色 helper 注入的图表 token：

- `--graft-chart-text-color`
- `--graft-chart-placeholder-color`
- `--graft-chart-border-color`
- `--graft-chart-container-color`

缺失且仅由 `createAcrylicMaterial()` 注入的 Material token：

- `--graft-glass-ambient-color`
- `--graft-glass-bg`
- `--graft-glass-border`
- `--graft-glass-shadow`
- `--graft-glass-blur`
- `--graft-glass-content-bg`
- `--graft-glass-content-border`
- `--graft-glass-content-shadow`
- `--graft-glass-content-blur`

Material 基础层另有 `--graft-glass-highlight` 与 `--graft-glass-noise-opacity` 参与实际渲染，但同样不在定义表内。它们不是预设私有 token，却说明 Material 表面没有完整的共享可写 authority。

## 隐藏权威检测

未发现基于具体 preset ID、名称或枚举的应用级 CSS selector、`body/html` class、条件渲染或 `switch` 分支。唯一的 `data-theme-preset-id` 和 neo-brutalist class 位于预设目录缩略卡，不影响应用渲染。

以下为合法共享派生规则：

- `data-graft-hard-surface` 只依赖可编辑的 `radiusPreset === square && shadowPreset === hard-offset`，由 [web/src/store/modules/setting.ts](web/src/store/modules/setting.ts#L323) 写入并由 [web/src/style/neo-brutalist.less](web/src/style/neo-brutalist.less#L9) 消费。
- `data-acrylic-glass` 只依赖可编辑的 `isAcrylicEnabled`，由 [web/src/layouts/index.vue](web/src/layouts/index.vue#L146) 写入并由共享 Acrylic CSS 消费。
- `theme-mode` 只依赖可编辑 `mode`。

真正的违规是运行时对象读取：`selectedThemePresetId -> preset.tokenOverrides/materialTokenOverrides`。此外，`preserveThemePersonalization` 作为持久化状态决定是否注入 Material token，并在首启默认预设上存在例外；同一预设 ID 因历史状态不同可产生不同视觉结果。

## 治理建议

1. 选择或重置预设时，把 mode、品牌色、authority/style patch、双模式语义 token 和 Material token**一次性物化**到现有可编辑状态。
2. 运行时 token 合成不再接收预设对象或 `preserveThemePersonalization`。`selectedThemePresetId` 只保留最近应用/目录展示等非视觉元数据。
3. 将“保留个性化/完整应用”改成一次性命令范围，而不是持久化的运行时优先级条件。
4. 新增共享的“图表”和“Material”高级 token 分组，登记上述 13 个预设注入 token，并补齐 `highlight` 与 `noise`。不要创建预设专属编辑器或 selector。
5. 保留 `data-graft-hard-surface`、`data-acrylic-glass` 等由可编辑字段驱动的共享派生规则；不要以预设 ID 复制这些分支。
6. 未来修复必须增加等价性回归：预设应用后仅保留物化的可编辑状态，移除/替换预设 ID 不得改变 `themeResolvedTokens`、root attributes 或渲染快照。

## 验证证据与限制

- 已完成源码链路、selector、Token 定义和 Workbench 控件的交叉审计。
- 审计阶段历史验证：`bun test src/store/modules/setting.test.ts` 通过 48 项主题状态测试。
- 审计阶段历史验证：同次运行的 `ThemeWorkbenchPresetCatalog.test.ts` 有 6 项失败，原因是 Vue Test Utils stub 注册触发 `WeakMap keys must be objects`；`ThemeWorkbenchPanel.test.ts` 另有 `i18n` named export 缺失。后续修复验证见 [trace](../traces/theme-preset-governance-audit-trace.md)：49 项 store 测试、57 项 Vitest 测试和 `bun run check` 均通过。这些历史装配问题不影响静态 authority 结论。
