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
- Next batch must add the server-owned explicit navigation contract before Web migration.

## Task Checklist

- [x] navigation-design-topic-and-skill
- [ ] backend-navigation-contract
- [ ] frontend-navigation-route-migration
- [ ] cross-boundary-validation-closeout

## Acceptance Conditions

- Canonical navigation domains, current mapping, Runtime/Source boundaries, and UI route rules have one design authority.
- Future agents discover and apply the placement gate before changing menus or UI routes.
- Server menu/bootstrap and Web route behavior converge on the design authority without compatibility routing.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill"],
  "pending_batches": [
    "backend-navigation-contract",
    "frontend-navigation-route-migration",
    "cross-boundary-validation-closeout"
  ],
  "current_batch": "navigation-design-topic-and-skill",
  "next_batch": "backend-navigation-contract",
  "closeout_status": "batch-1-complete"
}
```
