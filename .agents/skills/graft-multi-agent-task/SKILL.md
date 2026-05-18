---
name: graft-multi-agent-task
description: Repository-specific workflow wrapper for Graft multi-agent tasks. Use when a task should be executed through `graft-multi-agent-batch`, and the slice may need `graft-task-closeout` plus `graft-commit` to finish with a safe handoff after validation.
---

# Graft Multi-Agent Task

Use this skill when the current `Graft` task should run as a bounded multi-agent slice and still close out under normal repository governance.

Treat root `AGENTS.md` as the only governance source. This skill is a thin workflow wrapper around existing repository skills; it does not define a second boot path, validation contract, or commit policy.

## When To Use

Use this skill when all of the following are true:

* the task should actively use `graft-multi-agent-batch` during execution
* the work may end with a next-session handoff instead of a same-turn finish
* the slice should close out through the repository's normal handoff and commit path

Typical triggers:

* `run this as a multi-agent task`
* `coordinate this slice with subagents and hand off safely`
* `use the repository multi-agent workflow for this task`

## Workflow

1. Ensure the current turn already has the startup receipt required by root `AGENTS.md`.
2. Use `graft-multi-agent-batch` as the execution workflow:
   - keep the critical path local
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

## Boundaries

* do not use this skill as a substitute for `graft-boot`
* do not treat this skill as permission to skip `graft-multi-agent-batch` suitability checks
* do not duplicate `graft-task-closeout` or `graft-commit`
* do not invent a second governance source, second closeout format, or second commit workflow
* do not broaden ownership beyond the confirmed slice

## Output Expectations

When reporting progress or closeout from this wrapper, keep the result brief and include:

1. whether `graft-multi-agent-batch` was used for execution
2. whether `graft-task-closeout` was used for handoff evaluation
3. whether `graft-commit` created a scoped commit for the validated owned scope
4. the next-session startup prompt, if a handoff is required
