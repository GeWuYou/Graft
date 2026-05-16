---
name: graft-boot
description: Repository-specific startup workflow for Graft. Use when the task starts from a short prompt such as "boot", "continue", "read AGENTS", or "next step", and Codex should first ground itself in AGENTS.md, the ai-plan/ documents, the current repo state, and the likely server/web/plugin boundary before implementation.
---

# Graft Boot

Use this skill to start or resume work in `Graft` with minimal prompting.

Treat `AGENTS.md` as the source of truth. This skill is a startup workflow, not a replacement for repository rules.

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
7. If the task is complex and splits into disjoint parallel slices, consider `graft-multi-agent-batch`.
8. Before edits, tell the user what you read, how you classified the task, and the first implementation step.

## Recovery Rules

* recovery follows startup preflight; it does not replace it
* prefer repository truth over assumptions
* if the repo still lacks a stable build or runtime contract, say so explicitly and keep validation expectations honest
* if docs and code diverge, update the docs first or in the same change
