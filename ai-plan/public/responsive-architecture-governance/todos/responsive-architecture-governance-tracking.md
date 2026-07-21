# Responsive Architecture Governance Tracking

## Topic

Responsive Architecture Governance

## Scope

以最小业务改动建立并迁移 Desktop 优先、Container-first、Mobile Friendly 的共享响应式架构。范围覆盖长期规范、shared/runtime 基础设施、公共组件、壳层与页面迁移及治理门禁；不覆盖独立 Mobile 产品。

## Repository Truth

- `AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/governance/frontend/Graft响应式架构治理规范.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: web/src/shared owns runtime responsive capability, web/src/style owns responsive style tokens, and frontend governance docs own the durable architecture contract.
requires:
  design: true
  topic: true
  roadmap: true
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-loop
bootstrap:
  targets:
    - topic
    - design
    - roadmap
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- B1 文档 bootstrap 与 B2 shared runtime foundation 已完成；B2 没有迁移业务页面、壳层或 Manifest 运行时数据。
- `web/src/shared/responsive/**` 是无业务语义的阈值、container size、variant 与 token-name owner；`web/src/shared/composables/**` 只导出容器测量和 variant 组合式函数。
- 当前没有 authority escalation；B3 必须以 B2 的语义 API 为基础收敛共享组件，不能重建本地 viewport/device 判断。
- Responsive Debt：B2 新增 public API 暂无 Vue 入口消费者，因此 Knip 仅对 `shared/responsive/**`、`useContainerSize`、`useResponsiveVariant` 的 `exports` 建立受控豁免；owner=`B3-responsive-primitives`，deadline=`B3`，replacement=共享 primitive 的真实导入，完成首次消费后删除该豁免。
- Next step: `B3-responsive-primitives`。

## Task Checklist

- [x] `B1-docs-bootstrap`: 主题、规范和开发者 Manifest。
- [x] `B2-foundation-runtime`: shared token、container/variant 策略与壳层债务基线。
- [ ] `B3-responsive-primitives`: 共享 Responsive 组件、exception/debt 治理记录。
- [ ] `B4-page-migration-and-governance-gate`: 页面迁移、CI 规则和全面验收。

## Acceptance Conditions

- 新页面可通过 shared 响应式组件和语义 variant 获得统一布局，不需要设备条件分支。
- runtime 与治理 manifest 彻底分离，且不引入平行 UI 框架或页面体系。
- 所有高风险历史响应式债务有 owner、替代项与迁移批次，最终有四宽度和容器级验收路径。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["B1-docs-bootstrap", "B2-foundation-runtime"],
  "pending_batches": [
    "B3-responsive-primitives",
    "B4-page-migration-and-governance-gate"
  ],
  "current_batch": "B2-foundation-runtime",
  "next_batch": "B3-responsive-primitives",
  "closeout_status": "completed"
}
```
