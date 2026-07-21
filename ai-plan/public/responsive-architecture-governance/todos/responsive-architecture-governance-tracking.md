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

- B1 文档 bootstrap 已完成，结构、空白与链接检查通过，待 scoped commit。
- 当前没有 authority escalation；B2 必须先盘点已有 shared/style 和壳层实现，避免创建平行基础设施。
- Next step: `B2-foundation-runtime`。

## Task Checklist

- [x] `B1-docs-bootstrap`: 主题、规范和开发者 Manifest。
- [ ] `B2-foundation-runtime`: shared token、container/variant 策略及壳层高风险债务。
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
  "completed_batches": ["B1-docs-bootstrap"],
  "pending_batches": [
    "B2-foundation-runtime",
    "B3-responsive-primitives",
    "B4-page-migration-and-governance-gate"
  ],
  "current_batch": "B1-docs-bootstrap",
  "next_batch": "B2-foundation-runtime",
  "closeout_status": "completed"
}
```
