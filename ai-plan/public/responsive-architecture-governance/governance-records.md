# Responsive Governance Records

本文件是 `responsive-architecture-governance` topic 在 B3 建立的过渡治理记录。它不进入 `web` 运行时 bundle，也不替代长期规范中的 Exception Policy、Responsive Debt 或后续 B4 机器可读 Manifest。

## Controlled Exceptions

| Exception | Owner | Reason | Scope | Replacement / cleanup trigger |
| --- | --- | --- | --- | --- |
| `ResizeObserver` | `web/src/shared/composables/**` | Drawer、Split、Dialog 的可用宽度不等于 viewport | `useContainerSize`、`useResponsiveVariant` 与已有 table/chart/editor sizing | 保留为 container-first 基础设施；不得向业务页面暴露设备布尔值 |
| Monaco/ECharts/xterm instance sizing | shared platform surfaces | 第三方实例需要显式尺寸同步 | 既有 shared editor/chart/terminal surface | B4 逐一登记 consumer，替换业务层 viewport resize |
| Overlay/Popover positioning | TDesign/shared overlay owner | 由浏览器几何决定锚点位置 | 平台 overlay | 不作为业务布局或设备判断入口 |

## Responsive Debt Register

| Issue | Owner | Deadline | Migration Phase | Reason | Replacement |
| --- | --- | --- | --- | --- |
| 壳层 `min-width: 760px` | shell/shared | B4 | shell integration | 窄容器被阻断 | `ResponsiveLayout` 容器策略 |
| `ContentViewerFrame` window resize/viewport 判断 | shared viewer | B4 | page migration | workspace 以 viewport 而非容器决定布局 | shared container variant 与 Desktop Preferred 策略 |
| 管理组件本地 `@media` | shared management | B4 | page migration | 断点规则仍分散于管理域组件 | Responsive primitives 和共享 container rules |
| 固定像素 Dialog/Drawer consumer | shared + module consumers | B4 | page migration | 尚未消费 `ResponsiveDialog` purpose/size facade | `ResponsiveDialog` 表面策略 |

## B3 Disposition

- `ResponsivePage`、`ResponsiveHeader`、`ResponsiveToolbar`、`ResponsiveContent`、`ResponsiveEmpty` 与 `ResponsiveDialog` 是 shared 唯一的新增响应式组件入口。
- `ResponsiveDialog` 接受 `purpose`、`size` 与 slots；它没有 `width`、device 或 `isMobile` 公共接口。
- B2 的 Knip temporary export exemption 在 B3 首次真实消费容器/variant API 后删除。若 Knip 仍报告未使用 public export，应由 B3 在 shared consumer 内修复，而不是恢复豁免。
