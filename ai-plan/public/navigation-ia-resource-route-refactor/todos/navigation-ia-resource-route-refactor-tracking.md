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

- All implementation batches are committed. Archive readiness is blocked by the final full Web validation retry: `src/modules/project/shared/project-monaco.test.ts > project-monaco relayout bridge > resolves relayout after the scheduled animation frame runs` timed out after 20 seconds.
- The retry result was `1 failed | 192 passed (193)` test files and `1 failed | 1271 passed (1272)` tests; all preceding `bun run check` stages passed before `test:run` stopped the chain.
- This batch owns neither the Monaco bridge nor its test. Do not archive or modify unrelated runtime code in this topic-closeout batch; the next owner must resolve or formally reclassify the full-Web-check blocker, then rerun archive readiness.

## Task Checklist

- [x] navigation-design-topic-and-skill
- [x] backend-navigation-contract
- [x] frontend-navigation-route-migration
- [x] cross-boundary-validation-closeout
- [x] web-hygiene-and-final-validation

## Acceptance Conditions

- Canonical navigation domains, current mapping, Runtime/Source boundaries, and UI route rules have one design authority.
- Future agents discover and apply the placement gate before changing menus or UI routes.
- Server menu/bootstrap and Web route behavior converge on the design authority without compatibility routing.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract", "frontend-navigation-route-migration", "cross-boundary-validation-closeout", "web-hygiene-and-final-validation"],
  "pending_batches": [],
  "current_batch": "navigation-final-validation-and-archive",
  "next_batch": null,
  "closeout_status": "blocked-validation-failed"
}
```
