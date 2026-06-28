---
name: graft-ai-governance-audit
description: Repository-specific audit workflow for Graft AI governance. Use when evaluating or changing AI tooling, MCP adoption, repository skills, ai-plan governance, Python helper scripts, environment inventory, or drift between AGENTS.md and AI workflow documents.
---

# Graft AI Governance Audit

Use this skill to evaluate or maintain `Graft` AI governance across MCP, skills, `ai-plan`, Python helpers, and
environment inventory.

Treat root `AGENTS.md` as the startup and execution-governance source of truth. This skill is a docs/automation audit
workflow; it does not replace repository startup, validation, closeout, or commit rules.

## Workflow

1. Complete the startup preflight from root `AGENTS.md`.
2. Classify the task as `docs/automation` unless the requested change also modifies `server`, `web`, OpenAPI, or shared
   runtime contracts.
3. Read:
   - `.ai/environment/tools.ai.yaml`
   - `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`
   - `ai-plan/design/governance/ai/CodeGraph-MCP-辅助开发规范.md` when CodeGraph is involved
   - `ai-plan/design/governance/frontend/TDesign-MCP-辅助开发规范.md` when TDesign or frontend component generation is involved
4. Inspect concurrent work before edits:
   - `git status --short`
   - keep ownership limited to AI governance docs, `.agents/skills/**` governance files, and `scripts/**` audit helpers
   - do not stage unrelated `server`, `web`, OpenAPI, dashboard, topic recovery, or generated artifact changes
5. Check live local tool state when relevant:
   - `codex mcp list`
   - `codex mcp get <name>`
   - `codex mcp get context7`
   - `codex mcp get github`
   - `codex mcp get playwright`
   - `codex mcp get headroom` when Headroom context compression or MCP tooling is involved
   - `codegraph status` only when CodeGraph index health matters
6. Run the structure audit:

```bash
python3 scripts/validate_ai_governance.py
```

7. If Python helper tests changed, run the focused tests:

```bash
python3 -m unittest discover -s scripts -p 'test_*.py'
```

8. Report:
   - startup receipt
   - MCP adoption status
   - skill coverage or drift
   - Python helper status
   - validation commands and results
   - any rejected tool and concrete reason

## Guardrails

- Do not add MCP servers to `server/go.mod`, `web/package.json`, CI, hooks, or runtime scripts.
- Do not commit personal MCP client configuration, tokens, database DSNs, `.codegraph/**`, `.ai/venv/**`, or browser
  artifacts.
- Do not introduce `memory` MCP or another hidden recovery store; use `ai-plan/public/**`.
- Do not use GitHub or database MCP write capabilities unless a repository skill explicitly owns the permission,
  workflow, and closeout evidence.
- Keep Context7, GitHub MCP, and Playwright MCP in user-level Codex MCP config only; do not commit client config.
- Use Context7 for current external library documentation, GitHub MCP for read-only PR/Actions context, and Playwright
  MCP as browser exploration before `graft-web-browser-agent` captures reproducible evidence.
- Keep new skills concise and prefer scripts for deterministic checks.

## Closeout Evidence

```text
AI governance audit:
- task_class: docs/automation
- owned_scope: ai-plan/design/**, .agents/skills/**, scripts/**
- tools_checked: codex mcp list / validate_ai_governance / unittest
- mcp_changes: adopted | pilot | rejected | none
- validation: <commands and results>
- commit_scope: <confirmed owned paths>
```
