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
   - pass `parent_model`, `worker_model`, verified `model_relation`, and comparison evidence to every
     dispatch; the worker model must be the same level as or lower than the current execution owner's model
   - if the relation is higher or unknown, pause and request explicit user approval or direction before dispatch;
     do not infer rank from model names, availability, or reasoning effort
3. Keep this wrapper concise during execution:
   - do not restate `graft-multi-agent-batch` in full
   - do not expand repository governance into a second checklist
4. If the current task explicitly asks for a sidecar skill to be authored, the main rollout may delegate that bounded skill-authoring slice to one subagent:
   - keep the ownership boundary explicit
   - keep the main agent responsible for integration, validation planning, and acceptance
5. When the active slice reaches an end state or may need a future-session handoff, route closeout through `graft-task-closeout`.
6. After successful validation of an implementation Phase or loop batch, run `$graft-commit` for the confirmed owned
   scope before starting or authorizing the next Phase or batch:
   - if validation, ownership, mixed-worktree, or scoped-staging rules block the commit, report that blocker in
     closeout and do not present the next Phase or batch as implementation-ready
   - do not use this requirement to stage unrelated files, skip validation, or weaken root `AGENTS.md` handoff rules
7. If closeout determines that the validated owned scope should be committed before handoff, execute that commit through `graft-commit`.
8. Emit the explicit next-session startup prompt required by root `AGENTS.md` whenever work is being handed to a future turn.
9. Ensure reusable-lesson evaluation is not skipped:
   - prefer letting `graft-task-closeout` run the Experience Capture Check
   - if this wrapper is ever forced to produce a bounded closeout without normal closeout delegation, it must still
     delegate lesson evaluation to `graft-lessons-learned`
10. When the current task is being orchestrated by `graft-multi-agent-loop`, treat the current slice as one delegated
   round and end the closeout with one fenced ` ```json ` block containing the machine-readable closeout result:
   - in the default `topic-completion-loop` mode, ordinary batch success must not emit `Next-session startup prompt:`
   - return round evidence and, when useful, an advisory `suggested_follow_up` containing candidate work or recovery
     context; do not return controller state or terminal decisions
   - only the outer main agent decides whether to continue, selects any next batch, and emits terminal handoff output
   - ordinary lint, type, style, or test failures remain owned by the current round worker for diagnosis. Before any
     repair edit, the worker must return the root `AGENTS.md` `Repair Confirmation Interaction Contract` proposal to
     the outer agent; the outer agent must invoke the user's native structured-choice control, and only
     `execute_repair` permits the declared repair, validation rerun, or later `$graft-commit`
   - a fixable validation failure without `execute_repair` is recovery evidence for the outer controller, not
     permission for worker self-repair or a topic-level blocked decision; record the proposal, selected option, and any
     ownership or authority conflict in `required_context` and `recovery_requirements`
11. When the current task is being orchestrated by `graft-multi-agent-loop`, it may receive bounded checkpoint requests
    from the outer main agent:

- treat checkpoint interrupts as health checks only
- do not treat `no visible diff yet`, `no final closeout yet`, one quiet wait timeout, or a checkpoint reply itself as
  permission to stop, hand off, or let the outer main agent take over the delegated implementation
- do not change the round goal, broaden scope, or append extra implementation work because of a checkpoint
- label checkpoint replies clearly as checkpoint status rather than final closeout
- reply with a structured status containing `current_phase`, `changed_files`, `last_validation`, `next_action`,
  `can_continue`, `estimated_remaining_minutes`, `eta_confidence`, and `risks_or_blockers`
- keep the final implementation responsibility, validation, and closeout with the current round worker even if the
  round used `graft-multi-agent-batch` internally
- after replying to a checkpoint with `can_continue=true`, expect the same round to continue under the current worker;
  do not treat the checkpoint reply as permission to stop before emitting the required final closeout

12. If a delegated round cannot safely emit the required closeout, return retry-exhaustion or another clearly blocked
    round result with its `required_context` and `recovery_requirements` to the main agent instead of silently
    continuing outside the loop contract. Retry exhaustion is wave evidence/recovery input only; it does not suspend
   or terminate the topic, and only the outer controller may resolve it into a terminal state.
    For retry exhaustion, `required_context` must preserve the failed round and evidence, while
    `recovery_requirements` must identify repair authority, repair eligibility, and safe retry inputs without naming a
    controller transition.
13. When this wrapper is running under `graft-multi-agent-loop`, it owns only the delegated round:

- it must not assume the outer loop orchestrator will finish the implementation locally
- it must return a usable closeout or an explicit blocked state for the current round
- it must not decide topic completion, update `pending_batches`, choose `next_batch`, or mark the topic
  `archive-ready`
- it must not resume the current batch, dispatch a repaired worker, create a topic recovery receipt, suspend the
  topic, or declare a terminal state

## Boundaries

- do not use this skill as a substitute for `graft-boot`
- do not treat this skill as permission to skip `graft-multi-agent-batch` suitability checks
- do not duplicate `graft-task-closeout` or `graft-commit`
- do not invent a second governance source, second closeout format, or second commit workflow
- do not broaden ownership beyond the confirmed slice
- after changing this skill or the related loop/batch worker contract, manually run
  `python3 scripts/validate_loop_controller_contract.py`; it is not a normal repository completion gate

## Output Expectations

When reporting progress or closeout from this wrapper, keep the result brief and include:

1. whether `graft-multi-agent-batch` was used for execution
2. whether `graft-task-closeout` was used for handoff evaluation
3. whether `graft-commit` created a scoped commit for the validated owned scope
4. whether `graft-lessons-learned` was reached through `graft-task-closeout` or explicit lesson delegation
5. the next-session startup prompt, if a handoff is required
6. when the task is loop-orchestrated, a trailing JSON closeout object for the current delegated round with:
   - `round_status`
   - `implementation_result`
   - `changed_scope`
   - `commit`
   - `validation_evidence`
   - `risks`
   - `blockers`
   - `required_context` when recovery or retry needs prior evidence
   - optional `recovery_requirements`, containing cause, repair proposal reference, authority evidence, and safe retry
     inputs without directing controller state
   - `model_delegation_evidence`
   - optional `suggested_follow_up`
  - do not include `continue`, `pending_batches`, `next_batch`, `archive_ready`, `topic_complete`, `stop_loop`,
    `suspend_topic`, or `wait_for_user`

The worker JSON above is evidence only. The outer controller must separately emit the canonical controller closeout
record defined by `graft-multi-agent-loop`; a worker must never substitute that record with its own `pending_batches`,
`next_batch`, terminal state, or recovery transition.

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

Checkpoint responses should also follow these formatting rules:

- begin with `Checkpoint status:`
- do not include `Next-session startup prompt:`
- do not append the final closeout JSON block
- if the worker can continue, leave final completion, validation, and closeout to the same worker round

When this wrapper runs as a `graft-multi-agent-loop` worker in the default `topic-completion-loop` mode:

- worker completion returns round evidence to the outer main agent; it never completes the topic loop
- `suggested_follow_up` may preserve recovery context or describe candidate next work, but it is non-authoritative
- only the outer controller maintains `completed_batches`, `pending_batches`, `current_batch`, and `next_batch`, and
  only it can select recovery transitions, `archive-ready`, or `blocked`
