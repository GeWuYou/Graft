# Navigation IA And Resource Route Refactor

## Current Status Summary

- Topic objective: establish navigation and resource-route authority, then migrate menu contracts and Web UI routes.
- Current status: `active`, blocked on the final full Web validation retry.
- Task class: `cross-boundary`.
- Intake summary: long-running refactor dispatched through `$graft-multi-agent-loop`.
- Canonical authority: `ai-plan/design/architecture/导航与资源路由信息架构规范.md`.
- Completed so far: navigation design, backend graph contract, frontend route migration, and cross-boundary validation closeout.
- Archive-readiness is blocked: the final `cd web && bun run check` retry repeated the known Monaco RAF timeout. The topic remains active until that validation passes or the owning Web scope resolves the test instability.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary: repository navigation and resource-route design truth, followed by server menu/bootstrap contract and web route consumers.

## Owned Scope

- navigation design and active-topic recovery materials
- future server menu contract and bootstrap metadata
- future web navigation assembly and UI route migration

Out of scope:

- HTTP API endpoint path migration
- Runtime-specific navigation or URL compatibility layers

## Locked Decisions

1. Navigation domains and resource-oriented UI URLs are independent; UI URLs do not encode navigation names, Runtime, or Source.
2. No empty visible groups, no UI route aliases or redirects, and no menu placement without a resolved owner and stable resource boundary.

## Current Recovery Point

- Batch 4 verified the complete contract and repaired explicit navigation ancestry for breadcrumbs and tab titles.
- Final Web validation retry failed only at `src/modules/project/shared/project-monaco.test.ts > project-monaco relayout bridge > resolves relayout after the scheduled animation frame runs`, which timed out after 20 seconds. Preserve this topic as active and resolve or explicitly reclassify that validation blocker before archive review.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
python3 scripts/validate_ai_governance.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/navigation-ia-resource-route-refactor/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
