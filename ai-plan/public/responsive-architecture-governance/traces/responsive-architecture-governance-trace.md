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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["B1-docs-bootstrap"],
  "pending_batches": [
    "B2-foundation-runtime",
    "B3-responsive-primitives",
    "B4-page-migration-and-governance-gate"
  ],
  "current_batch": "B1-docs-bootstrap",
  "next_batch": "B2-foundation-runtime",
  "closeout_status": "completed"
}
```
