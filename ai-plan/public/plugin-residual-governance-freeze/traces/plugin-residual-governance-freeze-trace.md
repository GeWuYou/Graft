# Plugin Residual Governance Freeze Trace

## Summary

- Re-ran startup preflight from root `AGENTS.md`.
- Kept the task classified as `cross-boundary`.
- Read root `AGENTS.md`, `.ai/environment/tools.ai.yaml`, `server/AGENTS.md`, `web/AGENTS.md`, active design truth, and active recovery truth before edits.
- Confirmed the repository decision: current-authority cleanup only; archive topics and archived names remain read-only.
- Added a plugin residual checker and allowlist so accepted residuals stay explicit and new drift fails closed.
- Repaired active governance, roadmap, and design wording where `plugin` still acted as the current canonical term for modules.

## Validation

- `python3 scripts/plugin_residual/test_check_plugin_residuals.py`
- `python3 scripts/plugin_residual/check_plugin_residuals.py`
- `git diff --check`

## Immediate Next Step

- If the checker later finds uncategorized non-archive hits, repair them at the true authority owner or extend the allowlist only when the hit clearly belongs to an accepted residual class.
