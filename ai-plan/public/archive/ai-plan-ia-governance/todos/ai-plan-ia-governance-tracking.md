# AI Plan IA Governance Tracking

## Topic

AI Plan IA Governance

## Scope

收敛 `ai-plan/**` 的 IA、topic metadata、archive lifecycle 与恢复入口规则，先锁定 authority 模型与 topic
contract，再按批次推进 router/template、catalog、validator 与 skill 同步，而不引入第二套 startup、validation
或隐藏 recovery store。

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

- Phase 1 Batch 1 已完成：
  - active topic recovery materials 已落地
  - `ai-plan/AGENTS.md` 已落地
  - ADR-001 与 ADR-002 已落地
  - `ai-plan/public/README.md` 已注册本主题，并保留并发的 compose topic 入口
- Phase 1 Batch 2 已完成：
  - `ai-plan/AGENTS.md` 已明确 router，区分 `design/`、`design/decisions/`、`roadmap/`、`public/`、
    `public/archive/`、`lessons/` 与 `templates/`
  - `ai-plan/templates/` 已创建 active topic README、startup prompt、tracking、trace 与 ADR starter templates
  - `ai-plan/README.md` 已能发现 templates directory 与 router 用法，但未成为第二套 governance source
  - `compose-project-management` recovery docs 已复核，当前无需为 template compatibility 做额外 retrofit
- Phase 1 Batch 3 已完成：
  - `ai-plan/catalog.json` 已作为单一 bounded machine index 落地，只覆盖 shared router docs、收敛 ADR 与
    `ai-plan-ia-governance` active topic materials
  - `ai-plan/AGENTS.md`、`ai-plan/README.md`、`ai-plan/public/README.md` 已明确 catalog 是 supplementary index，
    不会成为第二套 source of truth
  - 未对既有 `ai-plan/**` 文档做 broad frontmatter retrofit，也未引入 projection files
  - `compose-project-management` recovery docs 已再次复核；当前无需为 active-topic contract compatibility 追加修改
- Phase 1 Batch 4 已完成：
  - 新增 `scripts/validate_ai_plan_structure.py`，以最小结构守卫覆盖当前已采纳的 ai-plan governance slice
  - validator 仅校验 bounded `ai-plan/catalog.json`、`ai-plan/public/README.md` 中本主题所需的 shared active-topic
    entries、最小 template starters，以及 `compose-project-management` 的当前 active-topic contract compatibility
  - 新增 `scripts/test_validate_ai_plan_structure.py`，覆盖 validator 的 bounded catalog、active-topic index 与
    compatibility guard 回归场景
  - `compose-project-management` recovery docs 已经通过当前结构守卫，无需为本批次追加最小兼容性补丁
- Phase 1 Batch 5 已完成：
  - 新增最小 repo-local `graft-ai-plan-governance` skill，收口 `ai-plan/**` router、active topic recovery、
    template、catalog 与 bounded validator work 的窄治理入口
  - 同步 root `AGENTS.md`、`ai-plan/AGENTS.md`、`ai-plan/README.md` 与
    `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`，让后续 `ai-plan/**` 文档治理 work 有明确 skill 入口而不形成第二套
    startup / validation / recovery path
  - 最小更新 `scripts/validate_ai_governance.py`，守护新 skill 在 root `AGENTS.md`、`ai-plan/AGENTS.md`、
    `ai-plan/README.md` 与 AI tooling governance doc 的必要 reference
  - `compose-project-management` recovery docs 未改动，shared active-topic router 继续保留其 recovery entry
- 当前 authority 决议：
  - root `AGENTS.md` 继续是唯一 startup-governance source
  - `ai-plan/AGENTS.md` 负责 `ai-plan/**` local execution truth
  - active topic 的最小恢复材料固定为 `README.md`、`startup-prompt.md`、tracking、trace
  - `ai-plan/public/README.md` 只列 active topics
  - archive lifecycle 必须通过显式目录位置和 index 更新表达
  - `ai-plan/templates/**` 只提供 minimal starters，不触发 broad retrofit
  - `ai-plan/catalog.json` 允许作为单一 bounded machine index 存在，但 coverage 不强制 whole-repo retrofit，
    且不替代 shared router 或 topic-local authority files
- Final archive-readiness check 已完成：
  - `ai-plan-ia-governance` 已移入 `ai-plan/public/archive/ai-plan-ia-governance/`
  - `ai-plan/public/README.md` 不再将该 topic 列为 active topic
  - bounded catalog、structure guard、governance validator 与新 skill metadata 已同步到 archived 状态
  - `compose-project-management` active recovery entry 继续保留且未改动

## Task Checklist

- [x] phase-1-batch-1：governance topic、ADR、`ai-plan/AGENTS.md`
- [x] phase-1-batch-2：`ai-plan/AGENTS` router and recovery templates
- [x] phase-1-batch-3：topic metadata catalog and README normalization
- [x] phase-1-batch-4：validator and structure guard rollout
- [x] phase-1-batch-5：skill creation and governance sync closeout

## Phase 1 Acceptance Conditions

- `ai-plan/AGENTS.md` 存在，并被 root `AGENTS.md` 与 `ai-plan/README.md` 正确接入
- `ai-plan/AGENTS.md` 明确区分 `design/`、`design/decisions/`、`roadmap/`、`public/`、`public/archive/`、
  `lessons/` 与 `templates/` 的 router 边界
- active topic 具备 `README.md`、`startup-prompt.md`、tracking、trace
- `ai-plan/templates/` 提供最小 active topic recovery templates，且不会强制批量 retrofit 现有 topics
- `ai-plan/public/README.md` 为本主题保留 active entry，且不破坏并发 topic 状态
- `ai-plan/catalog.json` 作为单一 bounded machine index 落地，且不会把 `public/README.md` 或 topic-local files
  降级成第二层副本
- ADR-001 与 ADR-002 锁定 authority/metadata 与 lifecycle/archive 模型
- 已有最小结构守卫可检测 bounded catalog、shared index、minimal templates 与当前 compose compatibility 漂移
- 已有最小 repo-local `graft-ai-plan-governance` skill，可在不重定义模型的前提下推进后续 `ai-plan/**` 文档治理
- `compose-project-management` 保持 startable，未被本主题的 docs/automation 收口破坏

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
