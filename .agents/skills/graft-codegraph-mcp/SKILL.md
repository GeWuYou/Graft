---
name: graft-codegraph-mcp
description: Repository-specific workflow for installing, registering, initializing, validating, or troubleshooting CodeGraph MCP for Graft. Use when the task mentions CodeGraph, codegraph init, codegraph serve --mcp, MCP navigation, /mcp missing codegraph, or turning CodeGraph setup into a repeatable repository workflow.
---

# Graft CodeGraph MCP

Use this skill to set up or verify CodeGraph MCP as a developer-local AI navigation aid for `Graft`.

Treat root `AGENTS.md` and `ai-plan/design/CodeGraph-MCP-辅助开发规范.md` as the source of truth. This skill does not
replace startup governance, authority discovery, direct source reads, or repository validation entrypoints.

## Startup

1. Run the startup preflight from root `AGENTS.md` before repository conclusions or edits.
2. Classify CodeGraph setup and troubleshooting as `docs/automation` unless the task also changes `server` or `web`.
3. Read:
   - root `AGENTS.md`
   - `.ai/environment/tools.ai.yaml`
   - `ai-plan/design/CodeGraph-MCP-辅助开发规范.md`
4. Inspect `git status --short` before edits. If other agents have changes, keep this skill's scope limited to
   `.agents/skills/graft-codegraph-mcp/**`, CodeGraph governance docs, `.gitignore`, and confirmed CodeGraph-only
   hunks.

## Setup Workflow

1. Check whether the CLI exists:

   ```bash
   command -v codegraph
   codegraph --version
   ```

2. If missing, install outside project dependencies:

   ```bash
   bunx @colbymchenry/codegraph
   ```

   or, for a PATH-installed CLI:

   ```bash
   bun add -g @colbymchenry/codegraph
   ```

   Never run `bun add @colbymchenry/codegraph` inside `web`, and never add CodeGraph to `web/package.json`,
   `web/bun.lock`, `server/go.mod`, CI, hooks, or runtime scripts.

3. Register Codex MCP when the user wants Codex integration:

   ```bash
   codex mcp add codegraph -- codegraph serve --mcp
   ```

4. Verify Codex configuration:

   ```bash
   codex mcp list
   codex mcp get codegraph
   ```

   If `/mcp` does not show CodeGraph after registration, ask the user to restart the Codex session. A configured MCP
   server usually is not exposed to the already-running session's tool list until restart.

5. Initialize the current worktree from the repository root:

   ```bash
   codegraph init -i
   codegraph status
   ```

6. Confirm local index hygiene:

   ```bash
   git check-ignore .codegraph/codegraph.db
   git status --short
   ```

   `.codegraph/` must remain ignored and uncommitted.

## Troubleshooting

- `codegraph` missing from `/mcp`:
  - run `codex mcp list`
  - if no `codegraph` entry exists, run the `codex mcp add` command
  - if the entry exists, restart Codex
- `codegraph` command not found:
  - check `bun --version`
  - install with `bunx @colbymchenry/codegraph` or `bun add -g @colbymchenry/codegraph`
- `.codegraph/` appears in `git status`:
  - add or restore `.codegraph/` in `.gitignore`
  - use `git rm --cached` only if it was accidentally tracked, and keep the removal scoped
- CodeGraph query disagrees with code:
  - trust direct source reads and repository authority documents
  - rerun `codegraph init -i` or `codegraph status` to refresh local indexing evidence

## Usage Rules

- Use CodeGraph as navigation evidence for symbols, call chains, dependencies, module entrypoints, route discovery, and
  impact analysis.
- Do not use CodeGraph as canonical authority for module contracts, menu, permission, OpenAPI, bootstrap metadata,
  validation, or commit scope.
- Always read real files before modifying code or making a final technical claim.
- When a task touches TDesign Vue Next components, still use TDesign MCP according to
  `ai-plan/design/TDesign-MCP-辅助开发规范.md`; CodeGraph does not replace component docs.

## Validation

For this skill's own setup or docs changes, use the strongest honest docs/automation checks:

```bash
git diff --check -- .agents/skills/graft-codegraph-mcp AGENTS.md ai-plan/design/CodeGraph-MCP-辅助开发规范.md .gitignore
command -v codegraph
codegraph --version
codex mcp list
codex mcp get codegraph
codegraph status
git check-ignore .codegraph/codegraph.db
```

Do not run `graft validate backend` or `bun run check` for CodeGraph-only docs/automation changes unless the task also
changes `server` or `web` code.

## Closeout Evidence

Include a concise CodeGraph setup record when relevant:

```text
CodeGraph MCP setup:
- cli: installed | missing | not_checked
- cli_version: <version or not-applicable>
- codex_mcp: registered | missing | not_checked
- project_index: initialized | missing | not_checked
- git_ignore: passed | failed | not_checked
- note: <restart needed / fallback used / no runtime dependency added>
```
