# Responsive Architecture Governance

## Current Status Summary

- Topic objective: 建立 Graft Desktop 优先、Mobile Friendly 的共享响应式架构与治理路径，并分批迁移现有前端。
- Current status: `active`
- Task class: `web + docs/automation`
- Intake summary: Work Intake 确认为跨批次前端架构治理，先固化设计、路线与 topic，再按 loop 迁移运行时基础设施和页面。
- Canonical authority:
  - `ai-plan/design/governance/frontend/Graft响应式架构治理规范.md`
  - `ai-plan/roadmap/Graft响应式架构迁移路线.md`
  - `web/AGENTS.md`
  - `web/src/shared/**` 与 `web/src/style/**`
- Completed so far: B1 已建立规范、开发者 Manifest 与主题恢复材料；B2 已建立 shared 容器/variant 基础设施；B3 已建立首批 shared Responsive primitives 与过渡治理记录。
- Not started yet: shell integration、ResponsiveTable/Form/CardList、页面迁移与 CI 治理门禁。

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `web + docs/automation`
- recovery source: `none`
- authority summary: `web/src/shared/**` 是运行时响应式能力 owner，`web/src/style/**` 是样式 token owner，`ai-plan/design/governance/frontend/**` 是长期治理真值。

## Owned Scope

- `ai-plan/design/governance/frontend/**`
- `ai-plan/roadmap/Graft响应式架构迁移路线.md`
- `ai-plan/public/responsive-architecture-governance/**`
- `web/docs/**`
- 后续受该 topic 约束的 `web/src/shared/**`、`web/src/style/**`、`web/src/layouts/**` 与分批页面迁移范围。

Out of scope:

- 独立 Mobile UI、移动 API、第二套组件或移动业务流程。
- 在未完成 shared 策略前为业务页面添加设备检测或局部响应式实现。

## Locked Decisions

1. 响应式是 shared 层的容器优先能力，业务页面只声明语义 variant。
2. 机器可读 manifest 是治理资产，未来位于 `docs/responsive/` 或 `scripts/responsive/`，不进入 shared runtime bundle。
3. Desktop 保持完整能力；复杂 workspace 在 Mobile 只读降级。

## Phase Plan

- B1：规范、Work Contract、开发者 Manifest 与恢复材料。
- B2：shared 运行时 token、容器/variant 基础设施与壳层高风险债务。
- B3：Responsive 公共组件与受控 exception/debt 清单。
- B4：页面迁移、CI 治理门禁与全量验收。

## Current Recovery Point

- B3 已收尾：首批无业务语义 primitives 实际消费 B2 container/variant API，且删除了 Knip temporary exemption。
- 风险：B4 必须从 shell 与现有管理组件逐批接入，不能用业务状态、查询或分页语义替换 shared layout contract。
- Next step: `B4-page-migration-and-governance-gate`。

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full `Work Contract` in the tracking file, not here.

## Pending Batch Direction

- `B2-foundation-runtime`
- `B3-responsive-primitives`
- `B4-page-migration-and-governance-gate`

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/responsive-architecture-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
