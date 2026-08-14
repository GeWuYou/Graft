# Runtime Composability Governance Tracking

## Topic

Runtime Composability Governance

## Scope

Establish and implement resource ownership, bounded capability visibility, and composition declarations across Graft Runtime, Module, Provider, Agent, Runner, Task Runtime and realtime boundaries.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/AGENTS.md`
- `ai-plan/design/architecture/运行时组合与资源治理设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/architecture/项目文件组织与扩展点设计.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Runtime owns lifecycle orchestration; each composition unit owns its resources; ModuleSpec and typed contracts own dependency and capability boundaries; Task/Submission, Agent, Provider and realtime retain their existing facts.
requires:
  design: true
  topic: true
  roadmap: true
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/design/architecture/运行时组合与资源治理设计.md
    - ai-plan/roadmap/运行时组合与资源治理实施路线.md
    - ai-plan/public/runtime-composability-governance/README.md
    - ai-plan/public/runtime-composability-governance/startup-prompt.md
    - ai-plan/public/runtime-composability-governance/todos/runtime-composability-governance-tracking.md
    - ai-plan/public/runtime-composability-governance/traces/runtime-composability-governance-trace.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current batch: `phase-0-resource-inventory`.
- Completed: architecture research, Work Intake, repository-wide design, roadmap and active-topic bootstrap.
- Current risk: existing runtime resources use several lifecycle patterns; implementation must inventory before introducing shared cleanup abstractions.
- Next step: produce a bounded server-side resource inventory and classify P0 lifecycle gaps.

## Task Checklist

- [x] Work Intake, design, roadmap and active-topic bootstrap.
- [ ] Phase 0: inventory creators, owners and disposers for current long-lived resources.
- [ ] Phase 1: unify lifecycle cleanup and shutdown evidence for P0 resources.
- [ ] Phase 2: introduce a narrow Resource Scope only if duplicate ownership patterns prove it necessary.
- [ ] Phase 3: add capability/composition declarations and capability-local health where justified.
- [ ] Phase 4: evaluate controlled dynamic change only if isolation, state migration and rollback requirements are proven.

## Acceptance Conditions

- Every newly changed long-lived resource has one creator, owner, disposer and shutdown test.
- Cross-module capability contracts remain typed and private implementation remains inaccessible outside its owner.
- Module/Provider/Agent/Runner/Task composition does not create a second DI, scheduler, task runtime or dynamic plugin platform.
- Dynamic enable/disable and HMR stay out of implementation unless a later approved decision changes the design.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-design-bootstrap"],
  "pending_batches": [
    "phase-0-resource-inventory",
    "phase-1-lifecycle-cleanup",
    "phase-2-narrow-resource-scope",
    "phase-3-capability-composition-declarations",
    "phase-4-controlled-change-evaluation"
  ],
  "current_batch": "phase-0-resource-inventory",
  "next_batch": "phase-1-lifecycle-cleanup",
  "closeout_status": "active"
}
```
