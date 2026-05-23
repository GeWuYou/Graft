---
name: graft-multi-agent-task
description: Repository-specific workflow wrapper for Graft multi-agent tasks. Use when a task should be executed through `graft-multi-agent-batch`, and the slice may need `graft-task-closeout` plus `graft-commit` to finish with a safe handoff after validation.
---

# Graft Multi-Agent Task

Use this skill when the current `Graft` task should run as a bounded multi-agent slice and still close out under normal repository governance.

Treat root `AGENTS.md` as the only governance source. This skill is a thin workflow wrapper around existing repository skills; it does not define a second boot path, validation contract, or commit policy.

## When To Use

Use this skill when all of the following are true:

- the task should actively use `graft-multi-agent-batch` during execution
- the work may end with a next-session handoff instead of a same-turn finish
- the slice should close out through the repository's normal handoff and commit path

Typical triggers:

- `run this as a multi-agent task`
- `coordinate this slice with subagents and hand off safely`
- `use the repository multi-agent workflow for this task`

## Workflow

1. Ensure the current turn already has the startup receipt required by root `AGENTS.md`.
2. Use `graft-multi-agent-batch` as the execution workflow when the active slice actually benefits from internal
   delegation:
   - the agent executing the current slice keeps its immediate blocking step local
   - when this wrapper is running as one round under `graft-multi-agent-loop`, that execution owner is the delegated
     worker subagent rather than the outer loop orchestrator
   - split only disjoint, reviewable slices
   - pass inherited startup context to every subagent
3. Keep this wrapper concise during execution:
   - do not restate `graft-multi-agent-batch` in full
   - do not expand repository governance into a second checklist
4. If the current task explicitly asks for a sidecar skill to be authored, the main rollout may delegate that bounded skill-authoring slice to one subagent:
   - keep the ownership boundary explicit
   - keep the main agent responsible for integration, validation planning, and acceptance
5. When the active slice reaches an end state or may need a future-session handoff, route closeout through `graft-task-closeout`.
6. If closeout determines that the validated owned scope should be committed before handoff, execute that commit through `graft-commit`.
7. Emit the explicit next-session startup prompt required by root `AGENTS.md` whenever work is being handed to a future turn.
8. Ensure reusable-lesson evaluation is not skipped:
   - prefer letting `graft-task-closeout` run the Experience Capture Check
   - if this wrapper is ever forced to produce a bounded closeout without normal closeout delegation, it must still
     delegate lesson evaluation to `graft-lessons-learned`
9. When the current task is being orchestrated by `graft-multi-agent-loop`, treat the current slice as one delegated
   round and end the closeout with both:
   - a human-readable line beginning with `Next-session startup prompt:` when another round is required
   - one fenced ` ```json ` block containing the machine-readable closeout result
10. When the current task is being orchestrated by `graft-multi-agent-loop`, it may receive bounded checkpoint requests
    from the outer main agent:

- treat checkpoint interrupts as health checks only
- do not change the round goal, broaden scope, or append extra implementation work because of a checkpoint
- reply with a structured status containing `current_phase`, `changed_files`, `last_validation`, `next_action`,
  `can_continue`, `estimated_remaining_minutes`, `eta_confidence`, and `risks_or_blockers`
- keep the final implementation responsibility, validation, and closeout with the current round worker even if the
  round used `graft-multi-agent-batch` internally

11. If a delegated round cannot safely emit the required closeout, stop and return a clearly blocked state to the main
    agent instead of silently continuing outside the loop contract.
12. When this wrapper is running under `graft-multi-agent-loop`, it owns only the delegated round:

- it must not assume the outer loop orchestrator will finish the implementation locally
- it must return a usable closeout or an explicit blocked state for the current round

## Boundaries

- do not use this skill as a substitute for `graft-boot`
- do not treat this skill as permission to skip `graft-multi-agent-batch` suitability checks
- do not duplicate `graft-task-closeout` or `graft-commit`
- do not invent a second governance source, second closeout format, or second commit workflow
- do not broaden ownership beyond the confirmed slice

## Output Expectations

When reporting progress or closeout from this wrapper, keep the result brief and include:

1. whether `graft-multi-agent-batch` was used for execution
2. whether `graft-task-closeout` was used for handoff evaluation
3. whether `graft-commit` created a scoped commit for the validated owned scope
4. whether `graft-lessons-learned` was reached through `graft-task-closeout` or explicit lesson delegation
5. the next-session startup prompt, if a handoff is required
6. when the task is loop-orchestrated, a trailing JSON closeout object for the current delegated round with:
   - `closeout_status`
   - `continue`
   - `next_prompt`
   - `stop_reason`
   - `validation`
   - `commit`
   - `consumed_budget`
   - `remaining_budget`
   - `scope_expanded`
   - `risk_level`

When a loop-orchestrated worker answers a checkpoint request instead of a final closeout, keep the response short and
structured. It must include:

1. `current_phase`
2. `changed_files`
3. `last_validation`
4. `next_action`
5. `can_continue`
6. `estimated_remaining_minutes`
7. `eta_confidence`
8. `risks_or_blockers`
