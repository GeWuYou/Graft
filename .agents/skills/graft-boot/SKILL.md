---
name: graft-boot
description: Repository-specific startup workflow for Graft. Use when the task starts from a short prompt such as "boot", "continue", "read AGENTS", or "next step", and Codex should first ground itself in AGENTS.md, the ai-plan/ documents, the current repo state, and the likely server/web/plugin boundary before implementation.
---

# Graft Boot

Use this skill to start or resume work in `Graft` with minimal prompting.

Treat `AGENTS.md` as the source of truth. This skill is a startup workflow, not a replacement for repository rules.

## Startup Workflow

1. Read `AGENTS.md`.
2. Read `.ai/environment/tools.ai.yaml` if it exists. Use `.ai/environment/tools.raw.yaml` only when the AI-facing
   inventory is missing or insufficient.
3. Read the relevant repository-wide documents in `ai-plan/`, starting with:
   - `ai-plan/design/项目设计.md`
   - `ai-plan/design/插件与依赖注入设计.md`
   - `ai-plan/design/前端架构设计.md`
   - `ai-plan/roadmap/MVP实施计划.md`
4. Read `ai-plan/public/README.md` and any active topic tracking or trace files mapped to the current branch or
   worktree.
5. Inspect the current repository state before assuming toolchains or entrypoints exist.
6. Classify the task into one of:
   - `server/core`
   - `server plugin`
   - `web module`
   - `cross-boundary`
   - `docs or automation`
7. Identify the first concrete boundary decision before editing:
   - core or plugin
   - public service interface or internal-only code
   - menu, route, page, API, permission linkage
   - required validation scope
8. If the task is complex and splits into disjoint parallel slices, consider `graft-multi-agent-batch`.
9. Before edits, tell the user what you read, how you classified the task, and the first implementation step.

## Recovery Rules

* prefer repository truth over assumptions
* if the repo still lacks a stable build or runtime contract, say so explicitly and keep validation expectations honest
* if docs and code diverge, update the docs first or in the same change
