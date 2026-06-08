---
name: graft-web-browser-agent
description: Repository-specific Playwright browser workflow for inspecting, authenticating into, and interacting with the local Graft web UI. Use when Codex needs Graft browser screenshots, DOM text snapshots, login/auth verification with temp credentials, or simple click/fill/wait checks before normal web validation.
---

# Graft Web Browser Agent

## Overview

Use this skill to give Codex an eyes-on-browser loop for Graft `web` work. It is an observation and interaction aid only; it does not replace `web/AGENTS.md` or the required `bun run check` validation for frontend changes.

Follow root `AGENTS.md` startup governance before using this skill. For frontend implementation tasks, also follow `web/AGENTS.md` and `graft-web-vibe-coding`; this skill only adds browser inspection capability after the normal frontend authority and design rules are in force.

Playwright MCP can be used as an optional exploration layer for this skill when it is available in `codex mcp list`.
Use MCP to discover page structure, accessible names, role selectors, and complex TDesign interactions quickly. Then
turn the stable path into a `browser_agent.py` command so the final evidence is reproducible and written under
`.ai/artifacts/browser/<session>`.

## Workflow

1. Confirm the local web app is running, usually with `cd web && bun run dev`.
   - Default local targets are backend `127.0.0.1:8080` through the Vite proxy and frontend `http://172.21.235.129:3002`.
   - If those ports are occupied, assume the user may already have the services running; do not start duplicate servers.
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

4. For authenticated Graft admin screenshots, use the temp credential file and let the script verify login before capture:

```bash
.ai/venv/bin/python .agents/skills/graft-web-browser-agent/scripts/browser_agent.py \
  --url http://172.21.235.129:3002 \
  --login \
  --credentials temp/username-passward.md \
  --session auth-check \
  --screenshot \
  --snapshot-text
```

The login helper accepts `username` / `account` / `user` and `password` / `passward` / `passwd` / `pwd` fields. It
writes only redacted auth status to `summary.json`; do not print or commit credential values, access tokens, or session
storage dumps.

5. Use focused interactions when debugging UI behavior:

```bash
.ai/venv/bin/python .agents/skills/graft-web-browser-agent/scripts/browser_agent.py \
  --url http://localhost:5173/audit/logs \
  --session audit-filter-check \
  --click "text=Filter" \
  --fill "input[placeholder='Keyword']=admin" \
  --wait-ms 500 \
  --screenshot
```

6. Use the browser evidence to guide fixes, then run the normal repository validation required by the changed scope.

## Playwright MCP Fast Path

Use Playwright MCP before `browser_agent.py` when any of these are true:

- the page structure, accessible names, or reliable selectors are unknown
- the target flow includes TDesign dialogs, drawers, dropdowns, tabs, pagination, or table filters
- the first task is exploratory triage rather than repeatable evidence capture
- a screenshot failed and the agent needs to inspect visible state before choosing the next stable action

Do not stop at MCP exploration. Once the selector path is known, rerun the same flow through `browser_agent.py` with
`--click`, `--fill`, `--wait-for`, `--screenshot`, and `--snapshot-text` as appropriate.

Recommended closeout evidence:

```text
Browser evidence:
- playwright_mcp_used: yes | no | unavailable
- browser_agent_used: yes | no
- session: <session-name>
- artifact_dir: .ai/artifacts/browser/<session-name>
- selectors_adopted: <stable selectors or not-applicable>
```

## Auth Failure Triage

When login fails:

- Verify the credential file can be parsed without printing secret values.
- Probe `http://172.21.235.129:3002/api/auth/login` through the frontend proxy before blaming the browser.
- Use `.ai/venv/bin/python`, not system `python3`; the system interpreter may not have Playwright installed.
- Inspect `.ai/artifacts/browser/<session>/summary.json` for `/api/auth/login` and `/api/auth/bootstrap` statuses.
- Treat a final `/login` URL after `--login` as an authentication failure even if a screenshot exists.

## Cleanup Rule

Browser artifacts live under `.ai/artifacts/browser/<session>` and are ignored by git. At task closeout, ask the user whether to clean or keep the session artifacts before the final handoff when this skill was used.

If the user chooses cleanup, run:

```bash
.agents/skills/graft-web-browser-agent/scripts/cleanup.sh --session <session>
```

If the user chooses to keep artifacts for the current conversation, report the retained directory in the handoff. Do not imply automatic cleanup after the Codex session ends; the reliable cleanup point is task closeout.

## Scripts

- `scripts/bootstrap.sh` creates `.ai/venv`, installs `.ai/browser/requirements.txt`, and installs Chromium into `.ai/ms-playwright`.
- `scripts/browser_agent.py` opens a URL, optionally authenticates with temp credentials, applies simple actions, waits, writes screenshots, and optionally writes visible page text.
- `scripts/cleanup.sh` removes one session, all browser artifacts, or artifacts older than a given age.

## Boundaries

- Do not add Playwright to `web/package.json` or create a second frontend test baseline for this skill.
- Do not treat Playwright MCP as the final browser artifact path; use it to discover stable actions and then capture
  evidence with `browser_agent.py`.
- Do not treat screenshots as acceptance by themselves; they are inspection evidence.
- Do not commit `.ai/venv`, `.ai/ms-playwright`, or `.ai/artifacts/browser`.
- Prefer `data-testid`, stable text, role selectors, or TDesign-visible labels for actions. Avoid brittle generated class selectors when a stable selector exists.
