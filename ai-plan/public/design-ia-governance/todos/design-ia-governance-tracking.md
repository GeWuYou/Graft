# Design IA Governance Tracking

## Topic

Design IA Governance

## Scope

治理 `ai-plan/design/**` 的信息架构、README 责任模型、目录边界、迁移顺序与归档策略，在不引入过度设计的前提下，
让 design 文档更适合长期维护和 AI 协作。

## Repository Truth

- `AGENTS.md`
- `ai-plan/AGENTS.md`
- `ai-plan/README.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`
- `ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md`
- `ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md`
- `.agents/skills/graft-ai-plan-governance/SKILL.md`
- `.agents/skills/graft-multi-agent-loop/SKILL.md`

## Current Recovery Point

- `ai-plan` 治理基线已经落地：
  - `ai-plan/AGENTS.md`
  - bounded `ai-plan/catalog.json`
  - minimal templates
  - structure guard
  - `graft-ai-plan-governance`
- `compose-project-management` 已作为 active topic 保留，当前主题不得影响其可启动状态。
- `ai-plan/design/**` 当前仍主要平铺，只有 `decisions/`、`release/`、`graft-design-system/` 等少量目录。
- Batch 1 已产出 topic-local 执行文档：
  - `ai-plan/public/design-ia-governance/design/phase-1-design-inventory-and-target-ia.md`
- 当前推荐 target IA 已确定为：
  - `architecture/`
  - `governance/ai/`
  - `governance/backend/`
  - `governance/frontend/`
  - `governance/platform/`
  - `domains/<domain>/`
  - 保留 `decisions/`、`release/`、`graft-design-system/`
- Batch 2 已完成：
  - `ai-plan/design/**` 目标目录已建立
  - 一级/二级 README skeleton 已落地为 router，而不是重复设计正文
  - `decisions/`、`release/` 目录已补齐 README
  - 本批次未移动 existing design docs，保持后续低耦合迁移批次边界清晰
- Batch 3 已完成：
  - 低耦合 architecture / governance 设计文档已迁入 `architecture/` 与 `governance/**`
  - 根层 canonical path 消费方已在最小必要范围内同步，包括 active / archived recovery、repo skills、plugin skill 与 AI governance validator
  - `compose-project-management` recovery docs 已同步新 authority 路径，保持 startability
- Batch 3b 已完成：
  - shared-asset registry 中被 Batch 3 移动的 design authority path 已同步
  - `scripts/plugin_residual/allowlist.json`、`ai-plan/lessons/**` 与列出的 `server/internal/**/README.md` 已修复同批次遗留旧路径
  - `python3 scripts/validate_ai_governance.py` 与 `python3 scripts/validate_ai_plan_structure.py` 已恢复通过
- 当前下一步：
  - 进入 Batch 4 迁移 domain / cross-cutting design 文档与剩余交叉引用

## Task Checklist

- [x] phase-1-batch-1：design inventory、分类矩阵、目标目录骨架、README 责任模型
- [x] phase-1-batch-2：建立 design 目标目录与 README
- [x] phase-1-batch-3：迁移低耦合 design 文档
- [x] phase-1-batch-3b：同步 shared-asset registry path
- [ ] phase-1-batch-4：迁移 domain / cross-cutting design 文档并修复引用
- [ ] phase-1-batch-5：archive / naming / governance sync closeout

## Acceptance Conditions

- 有明确的 design 分类矩阵和最终目录结构
- 一级/二级 README 责任清晰
- 迁移顺序分批可执行，且每批可停
- `compose-project-management` 和其他 active topic 仍可启动
- `ai-plan/design/**` 的 IA 比当前平铺状态更清晰

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-1-batch-1-design-inventory-and-target-ia-skeleton",
    "phase-1-batch-2-create-target-design-directories-and-readmes",
    "phase-1-batch-3-migrate-low-coupling-design-docs",
    "phase-1-batch-3b-sync-shared-asset-registry-paths"
  ],
  "pending_batches": [
    "phase-1-batch-4-migrate-domain-design-docs-and-fix-references",
    "phase-1-batch-5-design-archive-naming-and-governance-sync-closeout"
  ],
  "current_batch": "phase-1-batch-3b-sync-shared-asset-registry-paths",
  "next_batch": "phase-1-batch-4-migrate-domain-design-docs-and-fix-references",
  "closeout_status": "batch-3b-complete"
}
```
