---
name: graft-boot
description: Repository-specific startup workflow for Graft. Use when the task starts from a short prompt such as "boot", "continue", "read AGENTS", or "next step", and Codex should first ground itself in AGENTS.md, the ai-plan/ documents, the current repo state, assess whether multi-agent work is justified, and then enter implementation with the repository's closeout and commit workflow in place.
---

# Graft Boot

Use this skill to start or resume work in `Graft` with minimal prompting.

Treat `AGENTS.md` as the source of truth. This skill performs startup plus the mandatory workflow hooks that follow
startup; it does not replace repository rules.

## Startup Workflow

1. Run the startup preflight defined in `AGENTS.md` `4.1 Startup Governance`.
2. Emit the minimum startup receipt from `AGENTS.md` before substantive work:
   - `governance source`
   - `task class`
   - `recovery source`
3. If the current turn needs recovery context, read `ai-plan/public/README.md` only after preflight, then follow the
   mapped parent topic and relevant subtopic recovery files for the current task shape.
4. Read the relevant repository-wide design and roadmap truth needed by the task.
5. Inspect the current repository state before assuming toolchains or entrypoints exist.
6. Identify the first concrete boundary decision before editing.
7. If the task touches live schema or migration files, re-read the `server/AGENTS.md` migration/comment rules and
   treat table/column comment completeness as a mandatory implementation item, including core-owned handwritten
   migration directories such as `server/internal/httpx/migrations/**`.
8. Assess whether `graft-multi-agent-batch` is justified:
   - use it only when the task is large enough, write scopes stay disjoint, and the current slice owner can keep the
     immediate blocking step local
   - when the task explicitly uses `graft-multi-agent-loop`, the outer loop owner may instead delegate the whole
     bounded round to one worker subagent and keep only orchestration, budget tracking, acceptance, and stop
     conditions local
   - do not enable it for small, overlapping, or review-hostile slices
9. Before edits, tell the user what you read, how you classified the task, whether multi-agent work is justified, and
   the first implementation step.
10. When the current slice reaches a stop, completion, or handoff point, route the ending through `graft-task-closeout`
   instead of relying on an implicit wrap-up path.
11. `graft-task-closeout` must evaluate commit eligibility through `graft-commit` rules:
   - if validation and ownership allow a safe scoped commit, use `graft-commit`
   - if they do not, report the exact blocker and keep the handoff status honest
12. If the current turn ends by proposing a next task, include one explicit next-task startup prompt that restates the
    startup receipt fields needed by the next turn instead of assuming boot state carries across turns.

## Recovery Rules

* recovery follows startup preflight; it does not replace it
* prefer repository truth over assumptions
* if the repo still lacks a stable build or runtime contract, say so explicitly and keep validation expectations honest
* if docs and code diverge, update the docs first or in the same change
