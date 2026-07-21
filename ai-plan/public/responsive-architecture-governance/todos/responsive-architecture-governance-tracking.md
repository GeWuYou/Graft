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

- B1 文档 bootstrap、B2 shared runtime foundation 与 B3 shared primitives 已完成；B3 没有迁移业务页面、壳层或 Manifest 运行时数据。
- `web/src/shared/responsive/**` 是无业务语义的阈值、container size、variant 与 token-name owner；`web/src/shared/composables/**` 只导出容器测量和 variant 组合式函数。
- 当前没有 authority escalation；B3 必须以 B2 的语义 API 为基础收敛共享组件，不能重建本地 viewport/device 判断。
- `ResponsivePage`、`ResponsiveHeader`、`ResponsiveToolbar`、`ResponsiveContent`、`ResponsiveEmpty` 与 `ResponsiveDialog` 位于 `shared/components/responsive/**`，仅接收 slots 和语义 variant；`ResponsiveDialog` 以 `purpose`/`size` 解析表面，拒绝像素宽度与设备接口。
- B2 Knip temporary exemption 已在 B3 删除，container/variant/token API 已被 shared primitives 真实消费；`bun run deadcode:check` 通过。
- 受控 Exception 与 Responsive Debt 已登记在 `governance-records.md`，B4 必须按其 owner 完成 shell、Table/Form/CardList、治理门禁和页面迁移。
- Next step: `B4-page-migration-and-governance-gate`。

## Task Checklist

- [x] `B1-docs-bootstrap`: 主题、规范和开发者 Manifest。
- [x] `B2-foundation-runtime`: shared token、container/variant 策略与壳层债务基线。
- [x] `B3-responsive-primitives`: 共享 Responsive 组件、exception/debt 治理记录。
- [ ] `B4-page-migration-and-governance-gate`: 页面迁移、CI 规则和全面验收。

## Acceptance Conditions

- 新页面可通过 shared 响应式组件和语义 variant 获得统一布局，不需要设备条件分支。
- runtime 与治理 manifest 彻底分离，且不引入平行 UI 框架或页面体系。
- 所有高风险历史响应式债务有 owner、替代项与迁移批次，最终有四宽度和容器级验收路径。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["B1-docs-bootstrap", "B2-foundation-runtime", "B3-responsive-primitives"],
  "pending_batches": ["B4-page-migration-and-governance-gate"],
  "current_batch": "B3-responsive-primitives",
  "next_batch": "B4-page-migration-and-governance-gate",
  "closeout_status": "completed"
}
```
