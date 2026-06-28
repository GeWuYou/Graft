# Design IA Governance

## Status

- Topic: `design-ia-governance`
- Status: `archived`
- Loop mode: `topic-completion-loop`
- Task class: `docs/automation`
- Closed: `2026-06-28`
- Canonical authority:
  - `AGENTS.md`
  - `ai-plan/AGENTS.md`
  - `ai-plan/README.md`
  - `ai-plan/design/README.md`
  - `ai-plan/design/architecture/README.md`
  - `ai-plan/design/domains/README.md`
  - `ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md`
  - `ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md`

## Goal

收敛 `ai-plan/design/**` 的 Information Architecture，让 design authority 从长期平铺状态转入可维护、AI 友好、
可分批迁移并可安全归档的目录结构。

## Archive Summary

- `ai-plan/design/README.md` 已回到 router-only 角色，根层不再保留 canonical design documents。
- baseline architecture authorities 已稳定落在 `ai-plan/design/architecture/`。
- notification、container、compose、audit 相关设计 authority 已稳定落在 `ai-plan/design/domains/*/`。
- stale future-migration wording 已从 `ai-plan/design/README.md` 与 `ai-plan/design/architecture/README.md` 清理。
- Batch 1 的设计盘点与目标 IA 说明保留在：
  - `ai-plan/public/archive/design-ia-governance/design/phase-1-design-inventory-and-target-ia.md`
- `compose-project-management` 继续保留在 `ai-plan/public/README.md` 的 active-topic index 中，recovery entry 未改变。

## Final Archive-Ready Decision

- Decision: `archived`
- Reason:
  - Phase 1 Batch 1、2、3、3b、4、5 已全部完成。
  - 最终 acceptance conditions 已根据 live repository state 复核通过。
  - `design-ia-governance` 已在同一变更中移入 `ai-plan/public/archive/design-ia-governance/`，并从
    `ai-plan/public/README.md` 的 active-topic index 移除。
  - 当前已无新的 bounded Batch 留在本 topic scope 内。

## Validation Evidence

- `git diff --check`
- `python3 scripts/validate_ai_plan_structure.py`
- `python3 scripts/validate_ai_governance.py`

## Follow-up

- 这份目录只保留历史证据，不再作为 active loop entry。
- 后续若需要新的 `ai-plan/design/**` IA 治理切片，应重新执行 root startup preflight，并按当时的真实范围建立新
  topic，而不是恢复本归档目录为 active 状态。
