---
name: graft-multi-agent-batch
description: Repository-specific multi-agent coordination workflow for Graft. Use when the user explicitly wants subagent delegation, or when the current task cleanly splits into two or more disjoint slices across server, web, docs, or automation, and the main agent should keep ownership of review, validation, and final integration. `graft-boot` should assess whether this skill is justified before delegation starts.
---

# Graft Multi-Agent Batch

Use this skill for one execution wave of bounded, reviewable subagent slices. Treat root `AGENTS.md` as authority; this
skill coordinates a wave and never replaces startup, validation, commit, recovery, or closeout.

## Hierarchy

When nested in `graft-multi-agent-loop`:

```text
topic loop -> batch wave -> worker execution
```

Control returns only `worker -> batch coordinator -> outer loop controller`. The batch returns control to the outer
controller after the wave; it never completes or suspends the topic loop, selects the next loop batch, resumes a failed
loop batch, or makes a terminal decision.

## Activation

Use a batch only when:

- the task is large enough for parallelism to matter
- at least two write sets are disjoint and reviewable
- the current execution owner keeps its immediate blocking decision local
- `graft-boot` or the explicit caller has established the startup receipt and delegation authority

Use read-only explorers for discovery. Use write-capable workers only for bounded implementation ownership. Do not
delegate small, overlapping, authority-unclear, or validation-strategy-changing work.

## Coordination Workflow

1. Freeze the wave inventory and ownership before dispatch. For PR remediation, the main agent must first inventory all
   folded and open findings; delegation cannot omit outside-diff or nitpick rows.
2. Give every subagent the inherited governance source, task class, recovery source, objective, owned and forbidden
   scope, validation, expected output, commit authority, and verified `model_relation` with comparison evidence.
3. Require the worker model to be the same level as or lower than its direct parent. With inherited context, inherit the
   parent model/reasoning; with an explicit override, record comparison evidence. Never infer rank from names,
   availability, or reasoning effort; pause when the relation is unknown or higher-model approval is absent.
4. Once a write slice is delegated, keep ownership with that worker until final closeout, explicit block, owned-scope
   conflict, unsafe state, or actual retry exhaustion. Do not reclaim it after a quiet wait or missing visible diff.
5. While workers run, perform only non-overlapping coordination, review preparation, or validation planning.
6. Use bounded checkpoints only as health checks. A checkpoint is not closeout, stop, handoff, or permission for local
   takeover.
7. Verify every returned slice for inherited context, owned scope, implementation, direct validation, comment
   governance, commit evidence, and model relation.
8. Retry an unusable bounded slice once with a fresh worker and prior failure evidence. A second failure returns
   retry-exhaustion wave evidence and `required_context` to the caller or outer controller; it never terminates the
   topic loop.
9. Stop the wave when ownership begins to overlap, authority moves outside the inherited scope, validation changes
   strategy, or parallel execution becomes harder to review than local work.

Read [references/worker-contract.md](references/worker-contract.md) before dispatching a write worker, choosing wait
windows, requesting a checkpoint, validating response fields, or retrying.

## Worker Evidence

Each worker returns a final closeout or a checkpoint in the referenced shape. `suggested_follow_up` is advisory only.
The batch returns evidence upward to its caller or outer controller; it never emits controller fields or interprets
retry-exhaustion as topic completion.

## Boundaries

- Do not use this skill as a substitute for `graft-boot`.
- Do not delegate overlapping write scopes or unstated governance.
- Do not turn quiet workers into untracked main-agent implementation.
- Do not use delegation to bypass PR inventory closure, validation, comment review, or scoped commit rules.
- Do not treat batch completion, worker success/failure, commit success, recovery receipt, or retry exhaustion as a
  topic terminal signal.
- After changing the loop/task contract, run `python3 scripts/validate_loop_controller_contract.py`.
