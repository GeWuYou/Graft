# Docker Resource Context IA Tracking

## Topic

Docker Resource Context IA

## Scope

Implement the context-first Docker resource IA for Network and Volume, including normalized server-owned Context, relationship trust, contract filters, and the Graft web presentation.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/domains/container/容器管理设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: OpenAPI owns Docker resource wire contracts; Container owns normalized Context and relationship trust; web owns presentation only.
requires:
  design: true
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-batch
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - topic
    - design
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Contract, server, and Web implementation batches are complete: OpenAPI and Container provide one normalized resource-detail projection, Context, relationship trust, and server-side advanced filters; Network and Volume consume those contracts without frontend aggregation.
- No compatibility bridge was introduced: web does not parse Labels to infer Context, and the obsolete inspect-shaped Network detail schema was removed after its references were removed.
- Current stage: cross-boundary validation, browser evidence when a developer-owned runtime is available, and closeout.

## Task Checklist

- [x] Contract and design guideline.
- [x] Server projection and relationship trust.
- [x] Network and Volume Web IA implementation.
- [ ] Cross-boundary validation, browser evidence, and closeout.

## Acceptance Conditions

- Network and Volume show server-owned Context before relations and metadata.
- Metadata cannot drive business presentation or navigation.
- Lists use keyword/status by default; source, Compose project, driver, and scope are advanced server-side filters.
- Details follow `Overview -> Context -> Relations -> Configuration -> Metadata -> Danger Zone`.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "context-contract-and-design-guideline",
    "container-server-projection",
    "network-volume-web-ia"
  ],
  "pending_batches": [
    "cross-boundary-validation-and-closeout"
  ],
  "current_batch": "cross-boundary-validation-and-closeout",
  "next_batch": "browser-evidence-and-closeout",
  "closeout_status": "in-progress"
}
```
