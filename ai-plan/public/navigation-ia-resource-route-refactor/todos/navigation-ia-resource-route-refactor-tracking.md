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

- The previous resource-oriented UI route migration is superseded by the approved IA-aligned mapping. This topic now owns a replacement cross-boundary migration of all visible menu paths, including detail paths, deep links and persisted-tab validation, without aliases or redirects.

## Task Checklist

- [x] navigation-design-topic-and-skill
- [x] backend-navigation-contract
- [x] frontend-navigation-route-migration
- [x] cross-boundary-validation-closeout
- [x] web-hygiene-and-final-validation
- [x] ia-aligned-route-authority-and-skill
- [x] ia-aligned-server-menu-contract
- [x] ia-aligned-web-route-migration
- [ ] ia-aligned-cross-boundary-validation

## Acceptance Conditions

- Canonical navigation domains, current mapping, Runtime/Source boundaries, and UI route rules have one design authority.
- Future agents discover and apply the placement gate before changing menus or UI routes.
- Server menu/bootstrap and Web route behavior converge on the design authority without compatibility routing.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract", "frontend-navigation-route-migration", "cross-boundary-validation-closeout", "web-hygiene-and-final-validation", "ia-aligned-route-authority-and-skill", "ia-aligned-server-menu-contract", "ia-aligned-web-route-migration"],
  "pending_batches": ["ia-aligned-cross-boundary-validation"],
  "current_batch": "ia-aligned-cross-boundary-validation",
  "next_batch": "ia-aligned-cross-boundary-validation",
  "closeout_status": "web-validation-retry-required"
}
```
