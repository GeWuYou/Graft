# Design IA Governance Trace

## 2026-06-28 Topic initialization

- 建立 active topic：`ai-plan/public/design-ia-governance/README.md`
- 建立 startup prompt：`ai-plan/public/design-ia-governance/startup-prompt.md`
- 建立 tracking：`ai-plan/public/design-ia-governance/todos/design-ia-governance-tracking.md`
- 建立 trace：`ai-plan/public/design-ia-governance/traces/design-ia-governance-trace.md`
- 目标明确为 `ai-plan/design/**` 内容本身的 IA 治理，而不是继续只做外围治理基线
- 当前第一批次定义为：
  - design inventory
  - 分类矩阵
  - 目标目录骨架
  - README 责任模型

## 2026-06-28 Batch 1 completed: design inventory and target IA skeleton

- 盘点 `ai-plan/design/**` 当前文档总量：
  - `46` 个 Markdown 文档
  - 其中 `33` 个仍位于 `ai-plan/design/` 根层
  - 已存在子目录仅有：
    - `decisions/`
    - `release/`
    - `graft-design-system/`
- 产出 topic-local 执行文档：
  - `ai-plan/public/design-ia-governance/design/phase-1-design-inventory-and-target-ia.md`
- 收敛的 target IA：
  - `architecture/`
  - `governance/ai/`
  - `governance/backend/`
  - `governance/frontend/`
  - `governance/platform/`
  - `domains/<domain>/`
  - 保留 `decisions/`、`release/`、`graft-design-system/`
- 明确 README 责任模型：
  - root `design/README.md` 负责目录路由
  - 一级目录 README 负责边界定义
  - 二级目录 README 负责域内 authority 与文档入口说明
- authority decision：
  - 本批次只需更新 topic-local recovery 与 design note
  - 共享 router、catalog、validator 现阶段无需同步
- 下一批方向：
  - 在 `ai-plan/design/**` 下创建目标目录与 README 骨架
  - 仍不批量移动 design 文档

## 2026-06-28 Batch 2 completed: target design directories and router readmes

- 在 `ai-plan/design/**` 下建立目标目录骨架：
  - `architecture/`
  - `governance/ai/`
  - `governance/backend/`
  - `governance/frontend/`
  - `governance/platform/`
  - `domains/compose/`
  - `domains/container/`
  - `domains/notification/`
  - `domains/audit/`
- 新增或补齐 router README：
  - `ai-plan/design/README.md`
  - `ai-plan/design/architecture/README.md`
  - `ai-plan/design/governance/**/README.md`
  - `ai-plan/design/domains/**/README.md`
  - `ai-plan/design/decisions/README.md`
  - `ai-plan/design/release/README.md`
- README 责任保持为目录路由与边界定义，不复制现有 design 正文。
- 保留已有目录 `decisions/`、`release/`、`graft-design-system/`，本批次不移动 existing design docs。
- `compose-project-management` 与其他 active topic 的 recovery entry 未改动。
- 下一批方向：
  - 迁移 low-coupling design docs 到 `architecture/`、`governance/`、`release/` 等目标目录
  - 在最小范围内修复引用，不扩大到 domain-heavy 文档迁移

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-1-batch-1-design-inventory-and-target-ia-skeleton",
    "phase-1-batch-2-create-target-design-directories-and-readmes"
  ],
  "pending_batches": [
    "phase-1-batch-3-migrate-low-coupling-design-docs",
    "phase-1-batch-4-migrate-domain-design-docs-and-fix-references",
    "phase-1-batch-5-design-archive-naming-and-governance-sync-closeout"
  ],
  "current_batch": "phase-1-batch-2-create-target-design-directories-and-readmes",
  "next_batch": "phase-1-batch-3-migrate-low-coupling-design-docs",
  "closeout_status": "batch-2-complete"
}
```
