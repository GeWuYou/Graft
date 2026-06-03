---
name: graft-web-browser-agent
description: Repository-specific Playwright browser workflow for inspecting and interacting with the local Graft web UI. Use when Codex changes or reviews frontend UI, layout, navigation, dialogs, forms, or interaction behavior and needs browser screenshots, DOM text snapshots, or simple click/fill/wait checks before normal web validation.
---

# Graft Web Browser Agent

## Overview

Use this skill to give Codex an eyes-on-browser loop for Graft `web` work. It is an observation and interaction aid only; it does not replace `web/AGENTS.md` or the required `bun run check` validation for frontend changes.

Follow root `AGENTS.md` startup governance before using this skill. For frontend implementation tasks, also follow `web/AGENTS.md` and `graft-web-vibe-coding`; this skill only adds browser inspection capability after the normal frontend authority and design rules are in force.

## Workflow

1. Confirm the local web app is running, usually with `cd web && bun run dev`.
2. Bootstrap the project-local browser environment if `.ai/venv/bin/python` or Playwright is missing:

```bash
.agents/skills/graft-web-browser-agent/scripts/bootstrap.sh
```

If bootstrap reports missing Chromium system dependencies, do not claim browser inspection is available yet. Report the printed `playwright install-deps chromium` command to the user; installing those packages is an explicit machine-level action.

3. Run `browser_agent.py` against the target page. Use a stable `--session` name so later checks can reuse the same artifact directory.

```bash
.ai/venv/bin/python .agents/skills/graft-web-browser-agent/scripts/browser_agent.py \
  --url http://localhost:5173 \
  --session ui-inspection \
  --screenshot \
  --snapshot-text
```

4. Use focused interactions when debugging UI behavior:

```bash
.ai/venv/bin/python .agents/skills/graft-web-browser-agent/scripts/browser_agent.py \
  --url http://localhost:5173/audit/logs \
  --session audit-filter-check \
  --click "text=Filter" \
  --fill "input[placeholder='Keyword']=admin" \
  --wait-ms 500 \
  --screenshot
```

5. Use the browser evidence to guide fixes, then run the normal repository validation required by the changed scope.

## Cleanup Rule

Browser artifacts live under `.ai/artifacts/browser/<session>` and are ignored by git. At task closeout, ask the user whether to clean or keep the session artifacts before the final handoff when this skill was used.

If the user chooses cleanup, run:

```bash
.agents/skills/graft-web-browser-agent/scripts/cleanup.sh --session <session>
```

If the user chooses to keep artifacts for the current conversation, report the retained directory in the handoff. Do not imply automatic cleanup after the Codex session ends; the reliable cleanup point is task closeout.

## Scripts

- `scripts/bootstrap.sh` creates `.ai/venv`, installs `.ai/browser/requirements.txt`, and installs Chromium into `.ai/ms-playwright`.
- `scripts/browser_agent.py` opens a URL, applies simple actions, waits, writes screenshots, and optionally writes visible page text.
- `scripts/cleanup.sh` removes one session, all browser artifacts, or artifacts older than a given age.

## Boundaries

- Do not add Playwright to `web/package.json` or create a second frontend test baseline for this skill.
- Do not treat screenshots as acceptance by themselves; they are inspection evidence.
- Do not commit `.ai/venv`, `.ai/ms-playwright`, or `.ai/artifacts/browser`.
- Prefer `data-testid`, stable text, role selectors, or TDesign-visible labels for actions. Avoid brittle generated class selectors when a stable selector exists.
