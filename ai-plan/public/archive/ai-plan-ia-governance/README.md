# AI Plan IA Governance

## Status

- Topic: `ai-plan-ia-governance`
- Status: `archived`
- Loop mode: `topic-completion-loop`
- Task class: `docs/automation`
- Closed: `2026-06-28`
- Repository governance authority:
  - `AGENTS.md`
  - `ai-plan/AGENTS.md`
  - `ai-plan/README.md`
  - `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
  - `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`
  - `ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md`
  - `ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md`

## Goal

收敛 `ai-plan/**` 的 IA、topic metadata、archive lifecycle 与恢复入口规则，为后续 router、template、catalog、
validator 与 skill 同步提供长期稳定的 authority，而不引入第二套 startup、validation 或隐藏 recovery store。

## Archive Summary

- `ai-plan/AGENTS.md` 已落地为 `ai-plan/**` local execution truth，并接入 root `AGENTS.md` 与 `ai-plan/README.md`。
- ADR-001、ADR-002 已锁定 authority / metadata 与 lifecycle / archive 模型。
- `ai-plan/templates/**` 已提供最小 active topic 与 ADR starters，且不触发 broad retrofit。
- `ai-plan/catalog.json` 已落地为 bounded machine index，保持 single-file、bounded、non-authoritative。
- `scripts/validate_ai_plan_structure.py` 与 `scripts/test_validate_ai_plan_structure.py` 已落地，守护当前采纳的窄结构切片。
- `.agents/skills/graft-ai-plan-governance/**` 已落地，给后续 `ai-plan/**` governance work 提供最小 skill 入口。
- `scripts/validate_ai_governance.py` 已同步守护新 skill 与其最小 router references。
- `compose-project-management` 保持 startable，未被本治理主题破坏。

## Final Archive-Ready Decision

- Decision: `archived`
- Reason:
  - 所有 Phase 1 batches 已完成。
  - 最终 archive-readiness check 通过。
  - `ai-plan-ia-governance` 已从 `ai-plan/public/README.md` active-topic router 移除，并转入
    `ai-plan/public/archive/ai-plan-ia-governance/`。
  - bounded catalog、validator 与 governance skill 已同步到归档后状态，没有残留 active-topic drift。

## Validation Evidence

- `git diff --check`
- `python3 scripts/validate_ai_plan_structure.py`
- `python3 scripts/validate_ai_governance.py`
- `python3 -m unittest scripts/test_validate_ai_plan_structure.py`
- `python3 -m unittest scripts/test_validate_ai_governance.py`

## Follow-up

- 后续 `ai-plan/**` 文档治理切片应从 root startup preflight 进入，然后使用 `$graft-ai-plan-governance`。
- `compose-project-management` 仍是 active topic，可从 [ai-plan/public/compose-project-management/README.md](/home/gewuyou/project/go/Graft-wt/feat/wt-audit-plugin-mvp/ai-plan/public/compose-project-management/README.md:1)
  继续启动，不依赖本归档主题继续保持 active。
