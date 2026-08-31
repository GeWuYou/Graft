---
name: graft-multi-agent-loop
description: Repository-specific loop orchestrator for Graft multi-agent tasks. Use when one bounded task should run through repeated same-session serial worker-subagent rounds of `graft-multi-agent-task` under the default `topic-completion-loop` mode until the topic reaches `archive-ready`, becomes `blocked`, or no remaining batches stay in scope.
---

# Graft Multi-Agent Loop

Use this skill for repeated bounded delegated rounds under one outer main agent. Treat root `AGENTS.md` as authority;
this skill owns orchestration state only and never replaces startup, recovery, validation, commit, or closeout rules.

## Modes

- `topic-completion-loop` is the default. Continue accepted batches automatically until archive readiness, a real
  blocker, an explicit stop, or a declared budget boundary.
- `checkpoint-loop` is opt-in. Stop after one accepted batch and return a governed handoff.

Record the mode before dispatch. Do not infer checkpoint mode from an incomplete topic.

## Controller Ownership

The outer main agent is the only topic lifecycle owner, state owner, batch-transition owner, and terminal decision
owner. Workers return round evidence and optional `suggested_follow_up`; worker completion never completes a topic
loop. A worker must not decide or emit `continue`, `pending_batches`, `next_batch`, archive readiness, or stop state.

For tasks with dependency-bearing slices, the outer main agent also owns the best-known hybrid DAG. It creates the initial
graph before dispatch, computes the ready frontier, and increments `topology_revision` when it adds, splits, merges, or
reorders undispatched nodes at a wave boundary. Completed and dispatched nodes remain immutable history. The topology
constrains scheduling only; it does not move architecture, acceptance, recovery, batch-state, or terminal authority to
the batch or workers.

Canonical controller transitions:

```text
INIT -> DISPATCH -> WAIT -> VERIFY -> SETTLE -> ADVANCE -> DISPATCH_NEXT -> DISPATCH
SETTLE -> ARCHIVE_CHECK -> ARCHIVE_READY | BLOCKED
VERIFY -> RECOVERY_REQUIRED
  -> RECOVERY_CONTEXT_RESTORED
  -> RECOVERY_COMPLETE
  -> RESUME_CURRENT_BATCH
  -> DISPATCH
```

Rules:

- A worker return enters `VERIFY`, never a terminal state.
- Only `SETTLE` updates `completed_batches` and `pending_batches`.
- Non-empty pending work advances through `DISPATCH_NEXT`; an empty set must pass `ARCHIVE_CHECK`.
- `RECOVERY_REQUIRED`, `RECOVERY_CONTEXT_RESTORED`, `RECOVERY_COMPLETE`, and `RESUME_CURRENT_BATCH` are
  non-terminal. `RECOVERY_COMPLETE -> RESUME_CURRENT_BATCH -> DISPATCH` is mandatory.
- Only the outer controller may resolve retry exhaustion or recovery evidence into a legal transition.

## Activation

Use this skill only when the user explicitly requests looped delegation or repository intake/governance selects it for
one bounded long-running task. Do not use it for one small local edit, overlapping write scopes, or work lacking an
acceptance boundary.

Before dispatch:

1. Complete root startup preflight and emit the required receipt.
2. Restore parent/subtopic recovery only after preflight when applicable.
3. Confirm task class, authority, owned/forbidden scope, acceptance criteria, and applicable semantic reviews.
4. Establish the initial topology revision, node dependencies, ready frontier, `current_batch`, `completed_batches`,
   `pending_batches`, `next_batch`, and explicit budget.
5. Keep the first blocking architecture or authority decision local to the outer controller.

## Round Workflow

1. Compute the current ready frontier from the topology and dispatch one worker through `graft-multi-agent-task` for the
   selected bounded node or wave. The worker may use `graft-multi-agent-batch` internally only when that frontier cleanly
   splits into disjoint scopes.
2. Pass governance source, task class, recovery source, topology revision, node id, dependencies, objective, owned and
   forbidden scope, authority owner, execution context, validation, closeout shape, commit authority, remaining budget,
   and verified parent/worker `model_relation` with comparison evidence.
   The worker model must be the
   same level as or lower than its direct parent; never infer rank from model names, availability, or reasoning effort.
3. Wait for final round evidence. Treat quiet windows as observations, not automatic stall or takeover permission.
4. Enter `VERIFY`: confirm scope, implementation, validation, comment governance, commit state, risks, blockers, and
   model evidence.
5. On accepted evidence, enter `SETTLE` and update state exactly once.
6. In topic-completion mode:
   - after settling a wave, recompute the ready frontier and update pending batches from the topology
   - when pending batches remain, enter `ADVANCE -> DISPATCH_NEXT -> DISPATCH`
   - when none remain, run topic acceptance and recovery/catalog checks in `ARCHIVE_CHECK`
   - regenerate bounded pending work if archive checks expose clear remaining work; only undispatched nodes may be
     added, split, merged, or reordered, with a new topology revision and recorded replan evidence
7. In checkpoint mode, stop after the accepted batch and emit the governed next-session prompt.
8. On failure, keep the failed batch unsettled and follow the recovery/retry contract; never complete it locally after
   delegation.

Read [references/orchestration-details.md](references/orchestration-details.md) for budgets, waits, checkpoints,
repair, retry, recovery, and handoff mechanics.

## Output Contract

Worker completion is round evidence; batch completion and validation success also never complete a topic loop. A worker
may emit advisory `suggested_follow_up` but must not emit or infer `continue`, `pending_batches`, or `next_batch`.
Only the outer controller settles `retry_exhausted` wave evidence with its `required_context` and emits the
canonical outer-controller closeout schema.

Read [references/output-contract.md](references/output-contract.md) before validating worker JSON, recording recovery,
or emitting controller closeout. Read a mode example only when preparing or checking an actual run:

- [references/topic-completion-loop.md](references/topic-completion-loop.md)
- [references/checkpoint-loop.md](references/checkpoint-loop.md)

## Validation

After changing this skill or the related batch/task worker contract, run:

```bash
python3 scripts/validate_loop_controller_contract.py
```

This focused regression check does not replace repository governance validation.

## Boundaries

- Do not use this skill as a substitute for `graft-boot` or bypass `graft-multi-agent-task`.
- Do not broaden ownership beyond the declared allowed scope.
- Do not skip closeout, validation, scoped commit rules, or controller state updates.
- Do not let checkpoints become real-time remote control or let a delegated slice silently become main-agent work.
- Do not treat recovery, retry exhaustion, commit success, validation success, or an empty pending list as terminal by
  itself.
- Do not reintroduce external fresh-session runners as a second orchestration path.
