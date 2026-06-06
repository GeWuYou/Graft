---
name: graft-multi-agent-batch
description: Repository-specific multi-agent coordination workflow for Graft. Use when the user explicitly wants subagent delegation, or when the current task cleanly splits into two or more disjoint slices across server, web, docs, or automation, and the main agent should keep ownership of review, validation, and final integration. `graft-boot` should assess whether this skill is justified before delegation starts.
---

# Graft Multi-Agent Batch

Use this skill when `Graft` work benefits from bounded parallel subagents.

Treat `AGENTS.md` as the source of truth. This skill expands the repository's subagent workflow; it does not replace it.

The main agent coordinates the batch wave. Delegated `worker` subagents keep implementation ownership for their
bounded slices until they emit a final closeout, return an explicit blocked state, report an owned-scope conflict, or
exhaust the retry policy.

## Use When

Use this skill only when all of the following are true:

* the task is large enough that parallel work materially shortens it
* the write sets can stay disjoint
* the current execution owner can keep its immediate blocking step local
* reviewability will still be acceptable after integration

## Coordination Workflow

1. Run the normal `graft-boot` grounding flow first and establish the startup receipt required by `AGENTS.md`.
2. Treat `graft-boot`'s multi-agent suitability check as the activation gate; do not delegate just because parallelism is available.
3. Identify the immediate blocking step and keep it local to the current execution owner.
   - when this batch runs inside one `graft-multi-agent-loop` round, the current execution owner is the delegated
     worker subagent, not the outer loop orchestrator
4. Split only non-blocking work into disjoint slices.
   - when this batch runs inside one `graft-multi-agent-loop` round, default sidecars to read-only `explorer`
     subagents; add write-capable `worker` sidecars only when the round remains reviewable and the current worker still
     owns final integration, validation, and closeout
5. Use `explorer` subagents for read-only discovery or comparison.
6. Use `worker` subagents only for bounded implementation slices with explicit ownership.
7. For every subagent, specify:
   - governance source
   - task class
   - recovery source
   - objective
   - owned files or subsystem
   - areas it must not touch
   - required validation
   - expected output format
8. Once a write-capable slice is delegated, keep implementation ownership with that same `worker`:
   - do not reclaim the slice locally just because one wait window elapsed
   - do not treat `no visible diff yet` as evidence of stall by itself
   - do not silently continue the worker's bounded slice in the main agent after a checkpoint reply
9. While subagents run, do only non-overlapping work locally:
   - review returned slices
   - prepare follow-up validation
   - refine the next safe slice
10. Treat `timeout != stalled` for active worker slices:
   - exceeding one wait window is not enough to declare a worker stalled
   - absence of visible repo-tracked changes is not, by itself, evidence of no progress
   - before judging stall, distinguish `no final closeout yet`, `no visible diff yet`, `no new visible output evidence`,
     and `closeout not started`
   - if observable changes in the worker's owned scope are gradually increasing, treat the worker as progressing and
     continue waiting instead of interrupting
   - if the current tool surface does not expose direct activity, do not rewrite that into `no activity`
11. Use bounded checkpoint requests instead of ad-hoc remote control:
   - default `checkpoint_budget=1` per active worker unless the batch budget explicitly raises it
   - use checkpoint only as a health check for possible `blocked`, architecture decision required, unsafe worktree,
     validation failure, or a long quiet window after soft timeout with no observable owned-scope changes
   - checkpoint requests must not change the task goal, broaden scope, or append implementation requirements
   - do not checkpoint, stop, or retry only because a final closeout is missing while owned-scope changes are still
     increasing
   - enforce cooldown before another interrupt against the same worker
   - a checkpoint response is not a closeout and must not be interpreted as permission for the main agent to finish the slice
12. After a usable checkpoint with `can_continue=true`, continue the same worker slice:
   - wait at least one post-checkpoint grace window sized from ETA or the batch default wait rule
   - if the worker was closed by interrupt handling, reopen it and resume the same goal, scope, and budget
   - if the worker later returns a valid final closeout, accept that closeout as the slice result
13. If a worker slice cannot produce a usable final closeout:
   - retry the same bounded slice once with a fresh worker
   - pass the retry worker the previous failure reason, partial owned-scope diff, and relevant validation evidence
   - if the second worker still fails, mark that slice blocked or stop the batch wave explicitly
   - do not recover the implementation locally and do not silently continue outside the declared batch contract
14. Stop the wave when ownership boundaries start to overlap, validation changes strategy, or the batch becomes harder to review than to implement locally.

## Acceptance Rules

Before accepting a subagent result, confirm:

* the subagent received the inherited startup context instead of only an objective
* the subagent stayed inside its ownership boundary
* the reported validation is enough for that slice
* the result still follows plugin, DI, and `menu + route + page + api + permission` boundaries
* any checkpoint response was treated as a health report, not a handoff or implicit stop signal
* any retry-exhausted slice was reported as blocked or wave-stop rather than being completed locally by the main agent

## Output Expectations

For every delegated `worker`, require one of these response shapes:

1. Final closeout for the bounded slice:
   - concise human-readable result
   - owned scope or changed files
   - validation performed
   - risks or blockers
   - explicit outcome such as complete, blocked, retry-needed, or owned-scope conflict
2. Checkpoint status for a still-running slice:
   - begin with `Checkpoint status:`
   - include `current_phase`, `changed_files`, `last_validation`, `next_action`, `can_continue`,
     `estimated_remaining_minutes`, `eta_confidence`, and `risks_or_blockers`
   - do not include `Next-session startup prompt:`
   - do not present the checkpoint as final closeout

## Boundaries

* do not use this skill as a substitute for `graft-boot`
* do not delegate overlapping write scopes
* do not let checkpoint interrupts turn the batch into real-time remote control of workers
* do not let an active delegated slice silently downgrade into untracked main-agent execution
* do not assume a subagent can inherit unstated governance; pass the inherited startup context explicitly
