# Navigation IA And Resource Route Refactor Tracking

## Topic

Navigation IA And Resource Route Refactor

## Repository Truth

- `AGENTS.md`
- `ai-plan/AGENTS.md`
- `ai-plan/design/architecture/导航与资源路由信息架构规范.md`
- `server/AGENTS.md`
- `web/AGENTS.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Repository-wide navigation and resource-route design authority with server menu/bootstrap and web route consumers.
requires:
  design: true
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-loop
bootstrap:
  targets:
    - topic
    - design
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Work Intake classified this as a long-running cross-boundary refactor.
- Batch 1 established the design authority and placement skill.
- Batch 2 added the server-owned explicit navigation graph and bootstrap contract; the next batch consumes it in the Web shell.

## Task Checklist

- [x] navigation-design-topic-and-skill
- [x] backend-navigation-contract
- [x] frontend-navigation-route-migration
- [x] cross-boundary-validation-closeout

## Acceptance Conditions

- Canonical navigation domains, current mapping, Runtime/Source boundaries, and UI route rules have one design authority.
- Future agents discover and apply the placement gate before changing menus or UI routes.
- Server menu/bootstrap and Web route behavior converge on the design authority without compatibility routing.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract", "frontend-navigation-route-migration", "cross-boundary-validation-closeout"],
  "pending_batches": [],
  "current_batch": "cross-boundary-validation-closeout",
  "next_batch": null,
  "closeout_status": "batch-4-complete-pending-archive-readiness"
}
```
