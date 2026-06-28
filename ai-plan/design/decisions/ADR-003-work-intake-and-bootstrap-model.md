# ADR-003: Work Intake And Bootstrap Model

- Status: accepted
- Date: 2026-06-28
- Scope: `AGENTS.md`, `ai-plan/**`, `.agents/skills/**`, `scripts/**`

## Context

`ai-plan/` already distinguishes repository-wide design truth, roadmap truth, active-topic recovery materials, and
archived history. It also has stable loop and closeout workflows. What it still lacked was a single authority layer
that decides how new long-running work enters that system.

Without one explicit intake model, later agents could keep treating `design`, `roadmap`, `topic`, or `ADR` as parallel
entrypoints instead of derived artifacts, which would make future skill growth drift toward many special-case intake
flows such as “design intake”, “bug intake”, or “feature intake”.

## Decision

1. `Work Intake` becomes the only authority-approved entry workflow for new long-running work in `Graft`.
2. `Design`, `Roadmap`, `Topic`, and `ADR` are not first-class entrypoints; they are artifacts selected by intake.
3. The authority stack stays document-first:
   - root `AGENTS.md`
   - `ai-plan/AGENTS.md`
   - this ADR
   - `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
   - `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`
4. A thin orchestration skill named `graft-work-intake` may exist, but it must not own independent business rules.
   It only interprets and executes the documented intake workflow.
5. `Work Contract v1` is the single intake decision payload for long-running work.
6. `Work Contract v1` is persisted only when intake decides the work needs an active topic.
7. `Work Contract v1` is stored inside the topic tracking file, not as a new standalone file and not in
   `ai-plan/catalog.json`.
8. `README.md` remains navigation-oriented; it does not become the full workflow-state authority.
9. `ai-plan/catalog.json` remains a bounded machine index only; it does not store complete intake contracts.
10. Intake uses a fixed decision table rather than open-ended per-skill reinterpretation.

## Work Contract v1

The persisted `Work Contract v1` lives in `ai-plan/public/<topic>/todos/<topic>-tracking.md` under a dedicated
`## Work Contract` section with one fenced `yaml` block.

Required fields:

- `version`
- `kind`
- `scope`
- `authority_summary`
- `requires.design`
- `requires.topic`
- `requires.roadmap`
- `requires.adr`
- `execution.engine`
- `execution.dispatch_skill`
- `bootstrap.targets`
- `closeout.archive`
- `closeout.lessons_review`

Allowed values in v1:

- `kind`: `feature | bug | refactor | audit | research | spike | docs`
- `scope`: `short-lived | long-running`

## Intake Rules

1. All new long-running work must pass through `Work Intake` before creating a new topic, roadmap, design authority,
   or ADR.
2. Short-lived work may still be handled directly after normal startup preflight.
3. `requires.topic=true` when the work needs multi-batch execution, stable recovery state, or loop orchestration.
4. `requires.design=true` when the work needs explicit authority-setting design truth before normal implementation.
5. `requires.roadmap=true` when the work needs an explicit staged plan across multiple batches.
6. `requires.adr=true` only when one decision must converge future batches before broader implementation or governance
   rollout can proceed safely.
7. When `requires.topic=true`, the default execution engine is `graft-multi-agent-loop` unless the documented
   authority chooses a narrower path.

## Bootstrap Rules

`Bootstrap` follows `Contract-driven Minimal Bootstrap`:

- create only the minimal skeleton required by the contract
- create no content beyond structural starter text
- do not pre-create artifacts that the contract did not request

Minimum targets:

- if `requires.topic=true`
  - create `README.md`
  - create `startup-prompt.md`
  - create tracking file
  - create trace file
  - update `ai-plan/public/README.md`
- if `requires.design=true`
  - create the narrowest design skeleton in repository-wide or topic-local authority space
- if `requires.roadmap=true`
  - create the narrowest roadmap skeleton in repository-wide or topic-local authority space
- if `requires.adr=true`
  - create one ADR skeleton from `ai-plan/templates/adr/`

## Consequences

- `Graft` keeps one long-running-work entrypoint instead of growing many parallel intake skills.
- `design`, `roadmap`, `topic`, and `ADR` stay derived from one decision payload.
- Later automation can validate intake structure without adding a hidden metadata store or a second active-work index.
- A thin `graft-work-intake` skill can route future work consistently while remaining subordinate to documentation
  authority.
