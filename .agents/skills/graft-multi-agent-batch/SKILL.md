---
name: graft-multi-agent-batch
description: Repository-specific multi-agent coordination workflow for Graft. Use when the user explicitly wants subagent delegation, or when the current task cleanly splits into two or more disjoint slices across server, web, docs, or automation, and the main agent should keep ownership of review, validation, and final integration.
---

# Graft Multi-Agent Batch

Use this skill when `Graft` work benefits from bounded parallel subagents.

Treat `AGENTS.md` as the source of truth. This skill expands the repository's subagent workflow; it does not replace it.

## Use When

Use this skill only when all of the following are true:

* the task is large enough that parallel work materially shortens it
* the write sets can stay disjoint
* the main agent can keep the critical path local
* reviewability will still be acceptable after integration

## Coordination Workflow

1. Run the normal `graft-boot` grounding flow first.
2. Identify the immediate blocking step and keep it local.
3. Split only non-blocking work into disjoint slices.
4. Use `explorer` subagents for read-only discovery or comparison.
5. Use `worker` subagents only for bounded implementation slices with explicit ownership.
6. For every subagent, specify:
   - objective
   - owned files or subsystem
   - areas it must not touch
   - required validation
   - expected output format
7. While subagents run, do only non-overlapping work locally:
   - review returned slices
   - prepare follow-up validation
   - refine the next safe slice
8. Stop the wave when ownership boundaries start to overlap, validation changes strategy, or the batch becomes harder to review than to implement locally.

## Acceptance Rules

Before accepting a subagent result, confirm:

* the subagent stayed inside its ownership boundary
* the reported validation is enough for that slice
* the result still follows plugin, DI, and `menu + route + page + api + permission` boundaries
