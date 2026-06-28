# AI Plan IA Governance Trace

## 2026-06-28 Phase 1 Batch 1 governance topic and ADRs

- 建立 active topic：`ai-plan/public/ai-plan-ia-governance/README.md`
- 建立 loop startup prompt：`ai-plan/public/ai-plan-ia-governance/startup-prompt.md`
- 建立 tracking：`ai-plan/public/ai-plan-ia-governance/todos/ai-plan-ia-governance-tracking.md`
- 建立 trace：`ai-plan/public/ai-plan-ia-governance/traces/ai-plan-ia-governance-trace.md`
- 建立 `ai-plan/AGENTS.md` 作为 `ai-plan/**` local execution truth
- 建立 ADR-001 与 ADR-002，锁定 authority/metadata 与 lifecycle/archive 模型
- 更新 `ai-plan/README.md` 以接入 `ai-plan/AGENTS.md`、ADR 目录与 topic recovery artifacts
- 最小更新 root `AGENTS.md`，要求修改 `ai-plan/**` 的 `docs/automation` 任务读取 `ai-plan/AGENTS.md`
- 将 active topic 注册进 `ai-plan/public/README.md`，同时保留并发的 `compose-project-management` 入口

## 2026-06-28 Locked decisions

- root `AGENTS.md` 继续是唯一 startup-governance source。
- `ai-plan/AGENTS.md` 是 `ai-plan/**` 的 local execution-truth document。
- active topic 的最小恢复材料固定为 `README.md`、`startup-prompt.md`、tracking、trace。
- `ai-plan/public/README.md` 只列 active topics，不承担 archive 摘要。
- `ai-plan/design/decisions/**` 用于收敛多个后续 batch 之前必须先锁定的治理 ADR。
- archive lifecycle 必须通过显式目录位置与 index 更新表达，不能靠隐式状态文本漂移。
- `ai-plan/templates/**` 只提供 minimal starters；copied files 必须改写为 live topic truth，模板变化本身不要求
  broad retrofit。

## 2026-06-28 Phase 1 Batch 2 router and templates

- 更新 `ai-plan/AGENTS.md`，明确 `design/`、`design/decisions/`、`roadmap/`、`public/`、`public/archive/`、
  `lessons/` 与 `templates/` 的 router 边界。
- 新增 `ai-plan/templates/active-topic/**`，提供 active topic README、startup prompt、tracking 与 trace starter
  templates。
- 新增 `ai-plan/templates/adr/ADR-XXX-short-title.md`，为后续收敛型 ADR 提供最小 starter template。
- 更新 `ai-plan/README.md`，让 templates directory 与 router usage 可发现，但不把 README 升级成第二套治理真值。
- 复核 `compose-project-management` recovery docs；当前最小 topic contract 已满足，无需为本批次追加 retrofit。

## 2026-06-28 Phase 1 Batch 3 metadata catalog and README normalization

- 新增单一 `ai-plan/catalog.json`，作为 bounded machine index，仅覆盖 shared router docs、ADR-001、ADR-002 与
  `ai-plan-ia-governance` active topic recovery materials。
- 更新 `ai-plan/AGENTS.md` 与 `ai-plan/README.md`，明确 catalog 是 supplementary、single-file、explicit-coverage
  的 machine index，不要求 per-document frontmatter，也不成为第二套 source of truth。
- 更新 `ai-plan/public/README.md`，明确 shared human router 继续是 active-topic authority，即使当前 catalog 覆盖更窄。
- 更新 ADR-001，把 bounded catalog 纳入 metadata model，而不把它升级成 whole-repo retrofit mandate。
- 再次复核 `compose-project-management` recovery docs；当前无需为本批次追加兼容性修改。

## 2026-06-28 Phase 1 Batch 4 validator and structure guard rollout

- 新增 `scripts/validate_ai_plan_structure.py`，以最小 docs/automation 结构守卫覆盖当前已采纳的 ai-plan governance
  slice，而不把它升级成 whole-repo ai-plan linter。
- validator 目前只校验：
  - bounded `ai-plan/catalog.json` coverage、authority map 与 approved entry shape
  - `ai-plan/public/README.md` 中本主题与 `compose-project-management` 的共享 active-topic index entry consistency
  - `ai-plan/templates/**` 的最小 starter files
  - `ai-plan-ia-governance` 与 `compose-project-management` 的当前 active-topic required files
- 新增 `scripts/test_validate_ai_plan_structure.py`，覆盖 catalog scope expansion、index drift、compatibility topic
  file drift 与 extra metadata regression。
- 更新 `ai-plan/AGENTS.md` 与 `ai-plan/README.md`，记录 validator command、bounded scope 与 focused unittest 入口。
- 复核 `compose-project-management` recovery docs；当前无需为 validator contract 做额外补丁。

## 2026-06-28 Phase 1 Batch 5 skill creation and governance sync closeout

- 新增 `.agents/skills/graft-ai-plan-governance/SKILL.md`，将 `ai-plan/**` router、active topic recovery、
  template、catalog 与 bounded validator work 收口到一个窄 docs/automation skill。
- 更新 root `AGENTS.md` repository skill list，显式区分 `graft-ai-plan-governance` 与
  `graft-ai-governance-audit` 的使用边界。
- 更新 `ai-plan/AGENTS.md` 与 `ai-plan/README.md`，让 future `ai-plan/**` governance work 能在 startup preflight
  后发现该 skill，同时不形成第二套 startup 或 validation path。
- 更新 `ai-plan/design/AI工具与MCP接入治理规范.md`，把该 skill 纳入 repo-local AI governance skill contract，并保留
  `graft-ai-governance-audit` 作为 broader tooling / MCP / inventory drift audit。
- 最小更新 `scripts/validate_ai_governance.py`，守护新 skill 在 root `AGENTS.md`、`ai-plan/AGENTS.md`、
  `ai-plan/README.md` 与 AI tooling governance doc 的必要 reference。
- 更新 `ai-plan/public/ai-plan-ia-governance/README.md`、tracking、trace 与 startup prompt，标记所有 Phase 1
  batches 已完成，并把下一步收敛为 outer loop 的 final archive-readiness check。
- 未修改 `compose-project-management` recovery docs；shared active-topic router 继续保留其 recovery entry，topic
  仍可 start。

## 2026-06-28 Final archive-readiness check

- 最终 archive-readiness check 通过。
- `ai-plan-ia-governance` 已从 `ai-plan/public/README.md` 的 active-topic index 移除。
- topic recovery materials 已移动到 `ai-plan/public/archive/ai-plan-ia-governance/`。
- `ai-plan/catalog.json` 已同步为 archived topic evidence，而不是 active-topic metadata。
- `scripts/validate_ai_plan_structure.py` 与 `scripts/test_validate_ai_plan_structure.py` 已同步为：
  - 继续守护 `compose-project-management` active-topic entry
  - 守护 `ai-plan-ia-governance` archived evidence 不发生路径漂移
- 为 `graft-ai-plan-governance` 补齐 `agents/openai.yaml`，保持 skill discovery 与 `validate_ai_governance.py`
  约束一致。
- `compose-project-management` recovery entry 未变，主题仍可直接启动。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-1-batch-1-governance-topic-and-adrs",
    "phase-1-batch-2-ai-plan-agents-router-and-templates",
    "phase-1-batch-3-topic-metadata-catalog-and-readme-normalization",
    "phase-1-batch-4-validator-and-structure-guard-rollout",
    "phase-1-batch-5-skill-creation-and-governance-sync-closeout"
  ],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archived"
}
```
