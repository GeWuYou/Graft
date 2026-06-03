# AI Environment Inventory

`.ai/environment/` stores generated environment truth for `Graft`.

## Files

- `tools.raw.yaml`
  - Raw, repository-relevant environment facts collected from the current machine.
- `tools.ai.yaml`
  - AI-facing summary derived from `tools.raw.yaml`.
  - Prefer reading this file first during startup and task planning.

## Refresh Commands

```bash
bash scripts/collect-dev-environment.sh --check
bash scripts/collect-dev-environment.sh --write
python3 scripts/generate-ai-environment.py
```

## Project-Local Browser Agent

`graft-web-browser-agent` uses a project-local Python environment for AI-assisted frontend screenshots and simple
browser interactions:

```bash
.agents/skills/graft-web-browser-agent/scripts/bootstrap.sh
```

Generated runtime files are intentionally ignored:

- `.ai/venv/`
- `.ai/ms-playwright/`
- `.ai/artifacts/browser/`

Use the skill cleanup script at task closeout when the user chooses to remove browser artifacts:

```bash
.agents/skills/graft-web-browser-agent/scripts/cleanup.sh --session <session>
```

## Rules

- Do not hand-maintain `tools.raw.yaml` or `tools.ai.yaml`.
- Refresh both files when repository toolchain expectations or environment guidance change.
- Keep secrets, machine-specific credentials, and private URLs out of the inventory.
- Read `tools.ai.yaml` first during repository startup; use `tools.raw.yaml` only when the AI-facing summary is missing
  or insufficient.
- Keep the generated inventory aligned with the repository's current local toolchain so docs and automation can reference one fact source instead of restating divergent command matrices.
- The inventory is environment truth, not startup or validation governance: root `AGENTS.md` remains the only startup
  governance source, and repository entrypoints such as `graft validate backend` / `bun run check` remain validation
  truth.
