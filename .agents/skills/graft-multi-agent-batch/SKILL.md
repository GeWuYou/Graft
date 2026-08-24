---
name: graft-multi-agent-batch
description: Repository-specific hybrid-DAG multi-agent coordination workflow for Graft. Use when the user explicitly wants subagent delegation, or when the current task cleanly splits into two or more dependency-aware, reviewable slices across server, web, docs, or automation, and the main agent should keep ownership of review, validation, recovery, and final integration. `graft-boot` should assess whether this skill is justified before delegation starts.
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

## Hybrid DAG Dispatch

The outer main agent creates the best-known task DAG before the first dispatch. The graph is a scheduling plan, not a
second controller. Each node carries:

- `node_id`, `objective`, and `depends_on`
- `owned_scope` and `forbidden_scope`
- `authority_owner`
- `validation` and `acceptance_gate`
- `execution_context`

The graph is identified by a monotonically increasing `topology_revision`. Completed and dispatched nodes are immutable
history. Only nodes that have not been dispatched may be added, split, merged, or reordered at a wave boundary, and
every change records its reason, evidence, affected nodes, dependency delta, authority impact, and validation impact.

Before dispatch, the coordinator validates dependency completeness and acyclicity, then computes the ready frontier:
planned nodes whose dependencies have settled successfully. Nodes in one frontier may run concurrently only when their
write sets, authority owners, execution contexts, and acceptance gates are independent. Otherwise the frontier is
conservatively serialized. The topology constrains scheduling only; architecture, acceptance, recovery, batch state,
and terminal authority remain with the outer controller.

## Activation

Use a batch only when:

- the task is large enough for parallelism to matter
- at least two write sets are disjoint and reviewable
- the current execution owner keeps its immediate blocking decision local
- `graft-boot` or the explicit caller has established the startup receipt and delegation authority

Use read-only explorers for discovery. Use write-capable workers only for bounded implementation ownership. Do not
delegate small, overlapping, authority-unclear, or validation-strategy-changing work.

## Coordination Workflow

1. Freeze the initial topology, wave inventory, and ownership before dispatch. For PR remediation, the main agent must
   first inventory all folded and open findings; delegation cannot omit outside-diff or nitpick rows.
2. Validate the DAG, compute the ready frontier, and filter out nodes with overlapping write sets, authority owners,
   execution contexts, or acceptance gates.
3. Give every dispatched worker the inherited governance source, task class, recovery source, topology revision,
   `node_id`, dependencies, objective, owned and forbidden scope, validation, expected output, commit authority, and
   verified `model_relation` with comparison evidence.
4. Require the worker model to be the same level as or lower than its direct parent. With inherited context, inherit the
   parent model/reasoning; with an explicit override, record comparison evidence. Never infer rank from names,
   availability, or reasoning effort; pause when the relation is unknown or higher-model approval is absent.
5. Once a write slice is delegated, keep ownership with that worker until final closeout, explicit block, owned-scope
   conflict, unsafe state, or actual retry exhaustion. Do not reclaim it after a quiet wait or missing visible diff.
6. While workers run, perform only non-overlapping coordination, review preparation, or validation planning.
7. Use bounded checkpoints only as health checks. A checkpoint is not closeout, stop, handoff, or permission for local
   takeover.
8. Verify every returned slice for inherited context, topology node, owned scope, implementation, direct validation,
   comment governance, commit evidence, and model relation. A successful node unlocks only its direct dependents.
9. Retry an unusable bounded slice once with a fresh worker and prior failure evidence. A second failure returns
   retry-exhaustion wave evidence and `required_context` to the caller or outer controller; its descendants remain
   unsettled and it never terminates the topic loop.
10. At the wave boundary, let the outer controller recompute the ready frontier and, when justified by new evidence,
    add or reorder only undispatched nodes with a new `topology_revision`.
11. Stop the wave when ownership begins to overlap, authority moves outside the inherited scope, validation changes
    strategy, or parallel execution becomes harder to review than local work.

Read [references/worker-contract.md](references/worker-contract.md) before dispatching a write worker, choosing wait
windows, requesting a checkpoint, validating response fields, or retrying.

## Worker Evidence

Each worker returns a final closeout or a checkpoint in the referenced shape. `suggested_follow_up` is advisory only.
Final closeout includes node-level `topology_evidence` for the assigned `node_id`; frontier and replan results are batch
evidence for the outer controller. The batch returns evidence upward to its caller or outer controller; it never emits
controller fields or interprets retry-exhaustion as topic completion.

## Boundaries

- Do not use this skill as a substitute for `graft-boot`.
- Do not delegate overlapping write scopes or unstated governance.
- Do not turn quiet workers into untracked main-agent implementation.
- Do not use delegation to bypass PR inventory closure, validation, comment review, or scoped commit rules.
- Do not dispatch a node before its dependencies settle successfully, and do not let workers mutate the topology.
- Do not treat a topology revision, ready frontier, or replan evidence as `pending_batches`, `next_batch`, recovery, or
  terminal controller state.
- Do not treat batch completion, worker success/failure, commit success, recovery receipt, or retry exhaustion as a
  topic terminal signal.
- After changing the loop/task contract, run `python3 scripts/validate_loop_controller_contract.py`.
