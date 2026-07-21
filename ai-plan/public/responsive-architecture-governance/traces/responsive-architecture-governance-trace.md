# Responsive Architecture Governance Trace

## 2026-07-21 B1-docs-bootstrap

- Work Intake 将本主题定为 `refactor`、`long-running`：需要 design/topic/roadmap，不需要 ADR，以 `graft-multi-agent-loop` 推进。
- 建立仓库级响应式规范，固定 Desktop 优先、Mobile Friendly、Container-first、variant 优先和 shared authority。
- 建立面向开发者的 Manifest，明确运行时与治理清单分离；不在本批创建 runtime manifest 或 CI 脚本。
- 验证：`git diff --check`、`python3 scripts/validate_ai_plan_structure.py` 与受影响 Markdown 相对链接检查通过。

## Locked Decisions

- 业务组件不得获得 `isMobile`，只消费 `compact`、`comfortable`、`spacious` 等语义 variant。
- `data` 表格保留 table 语义，`entity` 表格才可由 shared 切换 CardList。
- Monaco、ECharts、xterm、Overlay 等只允许作为 shared/平台受控例外，不得成为业务页面 viewport 检测的入口。

## 2026-07-21 B2-foundation-runtime

- 新增 `web/src/shared/responsive/**`：容器阈值、尺寸归一化、语义 variant 和 CSS token 名称。阈值继续与 `style/variables.less` 的 `768/992/1200` 真值对齐，运行时不暴露设备布尔值。
- 新增 `useContainerSize` 和 `useResponsiveVariant`。它们只以元素 `ResizeObserver` 驱动，容器缺失或 SSR 环境没有 `ResizeObserver` 时安全退回空尺寸；不监听窗口 resize。
- 新增 `style/responsive.less` 并由全局 style 入口加载，仅定义语义 token 与 container/inline-scroll utilities，不复制 `layout.less` 的页面 surface 或壳层规则。
- 保留既有 `useTableHostWidth` 和管理表格调用方作为 B3 迁移基础；`layout.less` 的 `min-width: 760px`、`ContentViewerFrame` 的 viewport 判断仍是后续债务，未在 B2 伪装为已修复。
- Knip public API debt：B2 不越权实现 B3 组件消费者，因此仅对 `shared/responsive/**`、`useContainerSize`、`useResponsiveVariant` 的 exports 建立豁免；B3 首次真实导入 shared primitive 时必须删除，其他路径和 issue 类型未被放宽。
- Focused validation：响应式阈值/语义接口、ResizeObserver 生命周期和 SSR fallback 测试，另含 web typecheck 与 stylelint。
- Full `bun run check` 已通过格式、类型、OpenAPI/i18n、lint、stylelint 与 hygiene（含 Knip）阶段；全量 Vitest 最终因 scope 外 `src/modules/project/pages/configuration-workspace/index.test.ts` 未找到 `configuration-diff-confirm-save` 失败。B2 不修改该业务测试，后续批次应在集成验证时复核。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["B1-docs-bootstrap", "B2-foundation-runtime"],
  "pending_batches": [
    "B3-responsive-primitives",
    "B4-page-migration-and-governance-gate"
  ],
  "current_batch": "B2-foundation-runtime",
  "next_batch": "B3-responsive-primitives",
  "closeout_status": "completed"
}
```

## 2026-07-21 B3-responsive-primitives

- 新增 `ResponsivePage`、`ResponsiveHeader`、`ResponsiveToolbar`、`ResponsiveContent`、`ResponsiveEmpty` 与 `ResponsiveDialog`。它们是无业务语义的 shared 组件，只消费 slots、layout/tone/purpose/size 等语义输入，不暴露 device 或像素宽度接口。
- primitives 通过 `useResponsiveVariant` 或 `useContainerSize` 实际消费 B2 容器基础设施；Dialog policy 以 `purpose`、`size` 和容器宽度在 `dialog`、`drawer`、`sheet`、`fullscreen` 间解析。workspace 在紧凑容器保持 readonly。
- 沿用 Management/PageHeader 的 composition-first 结构，但没有迁移或改写其查询、分页、选择、i18n、TDesign action 或业务数据语义。新增组件不改变 TDesign usage，因此 TDesign MCP preflight 不适用。
- 删除 B2 Knip public API temporary exemption；`bun run deadcode:check` 通过。受控 Exception 与 Responsive Debt 记录落在 `governance-records.md`，不进入 runtime bundle。
- 下一轮 B4 应先处理 shell integration，再按 shared strategy 推进 ResponsiveTable、ResponsiveForm、ResponsiveCardList、治理脚本/manifest 和页面迁移；不得扩散业务设备判断。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["B1-docs-bootstrap", "B2-foundation-runtime", "B3-responsive-primitives"],
  "pending_batches": ["B4-page-migration-and-governance-gate"],
  "current_batch": "B3-responsive-primitives",
  "next_batch": "B4-page-migration-and-governance-gate",
  "closeout_status": "completed"
}
```
