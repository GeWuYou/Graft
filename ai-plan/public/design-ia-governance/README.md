# Design IA Governance

## 当前状态摘要

- 当前主题目标是治理 `ai-plan/design/**` 的 Information Architecture，使 design 文档从长期平铺状态演进到可维护、
  AI 友好、可逐步迁移的目录结构。
- 当前状态：`active`。
- 任务分类：`docs/automation`。
- Canonical authority：
  - `AGENTS.md`
  - `ai-plan/AGENTS.md`
  - `ai-plan/README.md`
  - `ai-plan/design/AI任务追踪与恢复设计.md`
  - `ai-plan/design/AI工具与MCP接入治理规范.md`
  - `ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md`
  - `ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md`

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`docs/automation`
- recovery source：`parent topic`
- authority summary：`ai-plan/AGENTS.md` + `ai-plan/README.md` + `ai-plan/design/**` + `ai-plan/public/README.md`

## Owned Scope

当前 topic 允许从这些路径起步：

- `ai-plan/design/**`
- `ai-plan/README.md`
- `ai-plan/AGENTS.md`
- `ai-plan/catalog.json`
- `ai-plan/public/README.md`
- `ai-plan/public/design-ia-governance/**`
- `ai-plan/templates/**`
- `.agents/skills/graft-ai-plan-governance/**`
- `scripts/validate_ai_plan_structure.py`
- `scripts/test_validate_ai_plan_structure.py`

只有在 authority discovery 明确需要时，才允许最小化扩到：

- `scripts/validate_ai_governance.py`
- `scripts/test_validate_ai_governance.py`

禁止误触：

- 不得启动 `compose-project-management` 实现。
- 不得把 whole-repo frontmatter retrofit、graph/projection 扩展、文档网站或多语言平台化设计带入本主题。
- 不得在未完成分类与 README 骨架前批量重命名所有 design 文档。
- 不得破坏现有 active topic 的 recovery entry。

## Locked Decisions

1. root `AGENTS.md` 继续是唯一 startup-governance source。
2. `ai-plan/AGENTS.md` 继续是 `ai-plan/**` local execution-truth。
3. `ai-plan/design/**` 的治理目标是长期可维护、AI 友好和逐步迁移，而不是一次性做完整知识平台。
4. `ai-plan/catalog.json` 继续保持 bounded、single-file、non-authoritative。
5. design IA 迁移必须分 phase/batch，允许随时停止，并优先保留 Git history 与现有引用可修复性。

## Phase Plan

- Phase 1 Batch 1：design inventory、分类矩阵、目标目录骨架与 README 责任模型
- Phase 1 Batch 2：建立 `ai-plan/design` 目标目录与各目录 README
- Phase 1 Batch 3：迁移 architecture / governance / release / design-system 等低耦合文档
- Phase 1 Batch 4：迁移 domain-oriented 文档并处理交叉引用
- Phase 1 Batch 5：archive / naming / README / catalog / validator / skill sync closeout

## Current Recovery Point

- `ai-plan` 治理基线已完成，`graft-ai-plan-governance` 已可用于本主题。
- `ai-plan/design/**` 当前仍以平铺为主，仅有少量已有子目录：
  - `design/decisions/`
  - `design/release/`
  - `design/graft-design-system/`
- Batch 1 已完成：
  - 当前 design inventory 已盘点
  - 目标 IA 骨架已定义
  - README 责任模型已定义
  - 后续迁移批次边界已收敛
- Batch 1 输出：
  - `ai-plan/public/design-ia-governance/design/phase-1-design-inventory-and-target-ia.md`
- 当前下一步：执行 Batch 2，先建立 `ai-plan/design/**` 目标目录与 README 骨架，不在本轮批量迁移 design 文档。

## Pending Batch Direction

- `phase-1-batch-2-create-target-design-directories-and-readmes`

## Validation Targets

当前 docs/automation 批次至少运行：

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

如触及 `.agents/skills/**`、`scripts/**` 或 `ai-plan/design/AI工具与MCP接入治理规范.md`，再追加：

```bash
python3 scripts/validate_ai_governance.py
```

## Loop Entry

- 推荐使用：`ai-plan/public/design-ia-governance/startup-prompt.md`
- 推荐执行模式：`$graft-multi-agent-loop`
