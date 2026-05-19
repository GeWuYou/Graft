---
name: graft-multi-agent-batch
description: Repository-specific multi-agent coordination workflow for Graft. Use when the user explicitly wants subagent delegation, or when the current task cleanly splits into two or more disjoint slices across server, web, docs, or automation, and the main agent should keep ownership of review, validation, and final integration. `graft-boot` should assess whether this skill is justified before delegation starts.
---

# Graft Multi-Agent Batch

Use this skill when `Graft` work benefits from bounded parallel subagents.

Treat `AGENTS.md` as the source of truth. This skill expands the repository's subagent workflow; it does not replace it.

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
8. While subagents run, do only non-overlapping work locally:
   - review returned slices
   - prepare follow-up validation
   - refine the next safe slice
9. Stop the wave when ownership boundaries start to overlap, validation changes strategy, or the batch becomes harder to review than to implement locally.

## Acceptance Rules

Before accepting a subagent result, confirm:

* the subagent received the inherited startup context instead of only an objective
* the subagent stayed inside its ownership boundary
* the reported validation is enough for that slice
* the result still follows plugin, DI, and `menu + route + page + api + permission` boundaries
