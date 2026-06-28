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
- `ai-plan/design/README.md`
- `ai-plan/design/architecture/README.md`
- `ai-plan/design/domains/README.md`
- `ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md`
- `ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md`
- `.agents/skills/graft-ai-plan-governance/SKILL.md`
- `.agents/skills/graft-multi-agent-loop/SKILL.md`

## Final Recovery Point

- Topic 已归档到 `ai-plan/public/archive/design-ia-governance/`。
- `ai-plan/design/README.md` 当前只承担目录路由；baseline architecture authority 已稳定落在
  `ai-plan/design/architecture/`，domain authority 已稳定落在 `ai-plan/design/domains/*/`。
- Batch 5 已完成对 stale future-migration wording 的收口，避免 README 继续暗示待迁移的根层 canonical docs。
- `ai-plan/public/README.md` 已移除 `design-ia-governance` active-topic entry。
- `compose-project-management` 的 shared recovery entry 保持不变，active-topic startability 未受影响。
- 历史设计盘点说明保留在：
  - `ai-plan/public/archive/design-ia-governance/design/phase-1-design-inventory-and-target-ia.md`

## Task Checklist

- [x] phase-1-batch-1：design inventory、分类矩阵、目标目录骨架、README 责任模型
- [x] phase-1-batch-2：建立 design 目标目录与 README
- [x] phase-1-batch-3：迁移低耦合 design 文档
- [x] phase-1-batch-3b：同步 shared-asset registry path
- [x] phase-1-batch-4：迁移 domain / cross-cutting design 文档并修复引用
- [x] phase-1-batch-5：archive / naming / governance sync closeout

## Archive-Ready Check

- [x] 有明确的 design 分类矩阵和最终目录结构
- [x] 一级/二级 README 责任清晰
- [x] 迁移顺序已按批次完成，且每批都保持可停
- [x] `compose-project-management` 和其他 active topic 仍可从 shared public index 启动
- [x] `ai-plan/design/**` 的 IA 已比原平铺状态更清晰
- [x] 当前已无新的 bounded batch 留在本 topic scope 内

## Validation Evidence

- `git diff --check`
- `python3 scripts/validate_ai_plan_structure.py`
- `python3 scripts/validate_ai_governance.py`

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-1-batch-1-design-inventory-and-target-ia-skeleton",
    "phase-1-batch-2-create-target-design-directories-and-readmes",
    "phase-1-batch-3-migrate-low-coupling-design-docs",
    "phase-1-batch-3b-sync-shared-asset-registry-paths",
    "phase-1-batch-4-migrate-domain-design-docs-and-fix-references",
    "phase-1-batch-5-design-archive-naming-and-governance-sync-closeout"
  ],
  "pending_batches": [],
  "current_batch": "phase-1-batch-5-design-archive-naming-and-governance-sync-closeout",
  "next_batch": null,
  "closeout_status": "archived"
}
```
