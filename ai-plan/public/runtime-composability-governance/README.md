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
- Completed: Work Intake, design, roadmap, topic bootstrap, lifecycle inventory/cleanup, narrow-scope evaluation,
  Phase 3 typed composition declarations, Phase 4 controlled-change evaluation, and Phase 5 remaining P0 lifecycle evidence.
- Active subtopic: `subtopics/docker-runtime-agent/todos/docker-runtime-agent-tracking.md`; it owns the migration from
  server-local Docker execution to one Task-governed Runtime Agent.

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

- Runtime composition Phase 5 evidence is complete and its dynamic-change rejection remains unchanged.
- Continue the Docker Runtime Agent subtopic. Current batches are architecture authority/recovery followed by Task
  Runtime external execution foundation.

## Archive Readiness

- Decision: archive is not approved while the Docker Runtime Agent subtopic is active.

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
- Current bounded recovery: `subtopics/docker-runtime-agent/todos/docker-runtime-agent-tracking.md`
