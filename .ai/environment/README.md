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

## Rules

- Do not hand-maintain `tools.raw.yaml` or `tools.ai.yaml`.
- Refresh both files when repository toolchain expectations or environment guidance change.
- Keep secrets, machine-specific credentials, and private URLs out of the inventory.
