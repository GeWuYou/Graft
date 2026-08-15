# Runtime Composability Governance

## Current Status Summary

- Topic objective: 将运行时资源 ownership、capability visibility 和 composition unit 声明从设计固定到可分批实施的治理路径。
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running architecture/governance refactor; design, roadmap and active topic are required before implementation.
- Canonical authority:
  - `ai-plan/design/architecture/运行时组合与资源治理设计.md`
  - `ai-plan/design/architecture/项目设计.md`
  - `ai-plan/design/architecture/模块与依赖注入设计.md`
- Completed so far: Work Intake, design, roadmap, topic bootstrap, lifecycle inventory/cleanup, narrow-scope evaluation, and Phase 3 typed composition declarations.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: Runtime owns lifecycle; ModuleSpec and typed contracts own composition and capability boundaries; Task/Submission, Agent and Provider retain their existing domain authorities.

## Owned Scope

- Runtime/Module/Provider/Agent/Runner/Task/realtime lifecycle and capability-boundary design.
- `ai-plan/design/architecture/运行时组合与资源治理设计.md` and its implementation roadmap.

Out of scope:

- runtime plugin loader, filesystem discovery, marketplace, HMR or generic external framework port.
- replacement of Task/Submission state authority, module compile-time registry or existing configuration governance.

## Locked Decisions

1. Runtime resource ownership, capability visibility and composition declaration are independent but coordinated rules.
2. Scope is a narrow resource-lifecycle tool, not a universal dynamic Context hierarchy.
3. Dynamic enable/disable is Future-only and cannot precede cleanup, state and rollback proof.

## Current Recovery Point

- Design authority is fixed for Graft's runtime composition model.
- Next step: Phase 0 inventory and classify existing long-lived resources by creator, owner and disposer.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is in `todos/runtime-composability-governance-tracking.md`.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/runtime-composability-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
