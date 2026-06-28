# Design IA Governance Trace

## 2026-06-28 Topic initialization

- 建立本 topic 的 recovery materials，当前归档位置为 `ai-plan/public/archive/design-ia-governance/`
- 目标明确为 `ai-plan/design/**` 内容本身的 IA 治理，而不是继续只做外围治理基线
- 第一批次定义为 design inventory、分类矩阵、目标目录骨架与 README 责任模型

## 2026-06-28 Batch 1 completed: design inventory and target IA skeleton

- 盘点 `ai-plan/design/**` 的根层平铺现状，并确定 phased migration 仍是最小可行路径。
- 收敛 target IA：
  - `architecture/`
  - `governance/ai/`
  - `governance/backend/`
  - `governance/frontend/`
  - `governance/platform/`
  - `domains/<domain>/`
  - 保留 `decisions/`、`release/`、`graft-design-system/`
- 产出设计盘点说明：
  - `ai-plan/public/archive/design-ia-governance/design/phase-1-design-inventory-and-target-ia.md`

## 2026-06-28 Batch 2 completed: target design directories and router readmes

- 建立 `ai-plan/design/**` 目标目录骨架与 router README。
- 保持 README 只承担目录边界与 authority 路由，不复制 design 正文。
- 明确后续迁移会先处理 low-coupling 文档，再处理 domain-oriented 文档。

## 2026-06-28 Batch 3 completed: low-coupling design-doc migration and path repair

- 使用 `git mv` 迁移 architecture 与 governance 下的低耦合 canonical design docs。
- 在最小必要范围内修复根治理文档、active/archive recovery materials、skills 与 validator 的 path drift。
- 发现 shared-asset registry 仍保留旧 authority path，并据此拆出 Batch 3b 作为最小 follow-up。

## 2026-06-28 Batch 3b completed: shared-asset registry path sync

- 同步 shared-asset registry、allowlist 与最小下游文档引用中的 stale low-coupling design authority path。
- 恢复 `python3 scripts/validate_ai_governance.py` 与 `python3 scripts/validate_ai_plan_structure.py` 通过。

## 2026-06-28 Batch 4 completed: domain design-doc migration and path repair

- 使用 `git mv` 迁移 notification、container、compose、audit 的 domain-oriented canonical design docs。
- 修复最小必要的 downstream path 消费方，并让 `domains/*/README.md` 成为对应 canonical 文档入口。
- 保持 `compose-project-management` 的 recovery entry 可启动。

## 2026-06-28 Batch 5 completed: design archive, naming, and governance-sync closeout

- 复核 live repository state：
  - `ai-plan/design/` 根层只保留 router README
  - baseline architecture docs 已稳定落在 `ai-plan/design/architecture/`
  - current domain authorities 已稳定落在 `ai-plan/design/domains/*/`
- 清理 stale future-migration wording：
  - `ai-plan/design/README.md`
  - `ai-plan/design/architecture/README.md`
- 使用 `git mv` 将 topic 移入：
  - `ai-plan/public/archive/design-ia-governance/`
- 从 `ai-plan/public/README.md` 移除 `design-ia-governance` active-topic entry。
- 将 archived README、tracking、trace 与 startup prompt 改写为 terminal historical evidence。
- 保持 `compose-project-management` shared recovery entry 不变。
- 验证通过：
  - `git diff --check`
  - `python3 scripts/validate_ai_plan_structure.py`
  - `python3 scripts/validate_ai_governance.py`

## 2026-06-28 Final archive-ready decision

- Final decision: `archive-ready`
- Topic lifecycle state: `archived`
- Stop reason:
  - 所有计划 batch 已完成，acceptance conditions 已与 live repository state 对齐，且当前无新的 bounded batch 留在本
    topic scope 内。

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
