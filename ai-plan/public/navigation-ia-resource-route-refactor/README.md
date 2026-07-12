# Navigation IA And Resource Route Refactor

## Current Status Summary

- Topic objective: establish navigation and resource-route authority, then migrate menu contracts and Web UI routes.
- Current status: `active`.
- Task class: `cross-boundary`.
- Intake summary: long-running refactor dispatched through `$graft-multi-agent-loop`.
- Canonical authority: `ai-plan/design/architecture/导航与资源路由信息架构规范.md`.
- Completed so far: `navigation-design-topic-and-skill`.
- Not started yet: backend navigation contract, frontend navigation route migration, cross-boundary validation closeout.

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

- Batch 1 created the design authority, active topic, discovery skill, and minimum routers.
- Next step: `backend-navigation-contract`.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
python3 scripts/validate_ai_governance.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/navigation-ia-resource-route-refactor/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
