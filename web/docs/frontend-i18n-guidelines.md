# Frontend I18n Guidelines

本文档记录 `web` 前端页面的 i18n 治理规则。Theme Center 是本规则的样例页面，后续 User、RBAC、Audit、
Access Log 模块沿用同一规则。

## Scope

- 适用范围：`web` shell 页面、业务模块页面、共享组件中所有用户可见文案
- 验收语言：`zh-CN`、`en-US`
- 验收主题：light、dark
- Web 完成入口仍以 `bun run check` 为准
- CI / web check 必须包含 `bun run lint:i18n`

## Copy Rules

- Web UI 禁止硬编码用户可见文案
- 按钮、导航、标题、副标题、placeholder、空状态、抽屉、弹窗、页脚操作必须使用 i18n
- 表格列名、筛选项、标签页、操作菜单、确认/取消/关闭等通用动作也必须使用 i18n
- 新增页面必须同时完成 `zh-CN` 与 `en-US` 文案，不得只补当前开发语言

## Length Budget

| Surface | zh-CN | en-US |
| ------- | ----- | ----- |
| 导航    | 4     | 14    |
| 标签页  | 6     | 18    |
| 按钮    | 6     | 20    |

- 导航、按钮、标签页必须支持英文长度，不得断词
- 自然语言 UI 表面禁止使用 `word-break: break-all`
- 自然语言 UI 表面禁止使用 `overflow-wrap: anywhere`
- 副标题可使用两行 clamp 与稳定 `min-height`
- 标题、卡片标题、卡片说明可按信息层级使用 ellipsis 或 line clamp

## Visual Checks

- 新页面必须通过 `zh-CN` / `en-US` 视觉检查
- 新页面必须通过 light / dark 视觉检查
- 重点检查导航、按钮、标签页、抽屉标题、弹窗标题、空状态和页脚操作是否因英文长度溢出、断词或遮挡
- Theme Center 作为当前样例页，应优先覆盖以上检查点

## Review Checklist

- 页面无硬编码用户可见文案
- `zh-CN` 与 `en-US` 文案完整
- 导航、按钮、标签页符合长度预算
- 自然语言表面未使用 `word-break: break-all` 或 `overflow-wrap: anywhere`
- 标题、副标题、卡片文案的 ellipsis / line clamp 与层级匹配
- `bun run check` 能覆盖 web 完成入口
- `bun run lint:i18n` 已纳入 CI / web check
