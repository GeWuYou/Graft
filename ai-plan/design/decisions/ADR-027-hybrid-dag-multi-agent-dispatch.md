# ADR-027: Hybrid DAG Multi-Agent Dispatch Authority

- Status: accepted
- Date: 2026-08-24
- Scope: repository-local multi-agent batch dispatch, worker contracts, loop frontier scheduling, and recovery-boundary planning
- Extends: ADR-004 task runtime state machine

## Semantic Review Layer

- `graft-platform-architecture-review`: applicable because this ADR assigns topology, scheduling, and recovery
  authority across the outer controller, batch wave, and worker boundaries; the review checks that no runtime or
  terminal authority leaks into workers.
- `graft-consistency-review`: applicable because the hybrid-DAG vocabulary and topology evidence are repeated across
  the batch, loop, task, and recovery contracts; the review keeps those terms and state ownership aligned.
- `graft-change-review`: applicable at implementation closeout to verify that the skills, validator, and recovery
  documents preserve this ADR's authority boundaries.
- `graft-ai-plan-governance`: applicable because the decision updates `ai-plan` recovery/governance language; it
  confirms that `Work Contract` remains owned by intake governance and that no second workflow authority is added.

No applicable Semantic Review Layer skill was unavailable for this decision. `Work Contract` rules remain owned by
the intake governance and are unchanged by this ADR.

## Context

`graft-multi-agent-batch` coordinates bounded worker slices, while the outer loop owns topic state, recovery, and
terminal decisions. Dispatch planning has historically been described as a manually frozen set of disjoint write
scopes, which leaves prerequisites implicit and can start a slice before an earlier authority or validation gate settles.

The repository needs dependency-aware dispatch without creating a second loop controller or a rigid waterfall. Runtime
evidence can change the best-known plan at a wave boundary, so only nodes that have not started may be replanned.

## Decision

1. Before the first dispatch, the outer main agent creates the best-known task DAG. Each node has an identifier,
   objective, dependencies, owned and forbidden scope, authority owner, validation, acceptance gate, and execution
   context. The graph has a monotonically increasing `topology_revision`.
2. A batch wave dispatches only the current ready frontier: planned nodes whose dependencies settled successfully. Nodes
   in one frontier may run in parallel only when their write scopes, authority owners, execution contexts, and acceptance
   gates are independent; otherwise the frontier is conservatively serialized.
3. After a wave settles, the outer controller recomputes the ready frontier. It may add bounded recovery nodes or
   reorder, split, or merge nodes that have not been dispatched, recording the revision, reason, evidence, affected
   nodes, dependency delta, authority impact, and validation impact.
4. Completed and dispatched nodes are immutable historical facts. Replanning cannot rewrite their semantics or bypass a
   failed node.
5. The topology is a scheduling constraint only. The outer main agent remains the sole owner of architecture decisions,
   acceptance, recovery transitions, batch state, and terminal topic state.
6. `graft-multi-agent-batch` returns node and frontier evidence upward; it never emits controller fields such as
   `pending_batches`, `next_batch`, `continue`, archive readiness, or terminal state. Workers return node-level evidence
   only.
7. Existing retry, checkpoint, commit, closeout, and recovery semantics remain authoritative. A failed node and its
   descendants remain unsettled until the outer controller performs the governed retry or recovery path.

## Authority Boundaries

| Concern | Canonical owner |
| --- | --- |
| Best-known DAG, topology revision, ready frontier, and local replanning | outer main agent / loop controller |
| One ready-frontier execution wave and dispatch-scope checks | `graft-multi-agent-batch` |
| Node implementation, direct validation, and node evidence | worker subagent |
| Topic batch state, recovery, archive readiness, and terminal state | `graft-multi-agent-loop` |
| Repository governance and validation entrypoints | root `AGENTS.md` and applicable governance skills |

## Consequences

- Prerequisites become explicit while independent ready nodes can still run concurrently.
- Runtime discoveries can update only the unstarted graph portion, preserving completed history and recovery evidence.
- The batch contract carries topology evidence, and the controller validator checks graph, frontier, and replanning
  invariants.
- No runtime scheduler, fresh-session runner, hidden recovery store, or worker-owned topic state is introduced.

## Rejected Alternatives

- Fixed total order: prevents safe parallelism and creates an unnecessary waterfall.
- Worker-owned topology changes: transfers architecture and recovery authority away from the outer controller.
- Batch-owned topic state machine: duplicates `graft-multi-agent-loop` and risks conflicting terminal decisions.
- Dispatching every initially discovered slice: permits unmet prerequisites and increases retry cost.
