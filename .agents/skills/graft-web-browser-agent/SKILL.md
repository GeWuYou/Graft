---
name: graft-web-browser-agent
description: "Inspect an explicitly approved Graft web runtime from the primary checkout and capture reproducible browser evidence."
---

# Graft Web Browser Agent

## Overview

Use this skill to give Codex an eyes-on-browser loop for Graft `web` work. It is an observation and interaction aid only; it does not replace [web/AGENTS.md](../../../web/AGENTS.md) or the required `bun run check` validation for frontend changes.

Follow root [AGENTS.md](../../../AGENTS.md) startup governance before using this skill. For frontend implementation tasks, also follow [web/AGENTS.md](../../../web/AGENTS.md) and `graft-web-vibe-coding`; this skill only adds browser inspection capability after the normal frontend authority and design rules are in force.

Before entering this skill, record the task's verification classification. Local browser use is not the default for UI
work: use it only after a task-local user or developer authorization for an explicit browser request, a browser-only
defect, or a real-browser environment investigation. Existing CI browser tests may be automated in CI, but never grant
local browser authority. Do not use this skill to replace an outstanding human acceptance flow.

Browser QA is a developer-owned primary-checkout activity. Do not use this skill from a numbered agent worktree, and
do not start services, invoke Playwright, capture screenshots, or run browser interactions there. Before browser
interaction, obtain user or developer approval, first confirm the runtime identity, and then confirm it serves the
primary checkout branch and HEAD under review; a different branch, dirty checkout, or error response is not valid
evidence. Pass the confirmed, non-secret runtime label with `--runtime-identity`; `summary.json` records that label
alongside the primary checkout repository, branch, and full HEAD. A worktree closeout records this as a
primary-checkout follow-up and does not block a validated scoped implementation commit.

Playwright MCP can be used as an optional exploration layer for this skill when it is available in `codex mcp list`.
Use MCP to discover page structure, accessible names, role selectors, and complex TDesign interactions quickly. Then
turn the stable path into a `browser_agent.py` command so the final evidence is reproducible and written under
`.ai/artifacts/browser/<session>`.

## Workflow

1. Confirm existing task-local browser authorization, a clean primary checkout, and independently proven runtime identity for the current branch and full HEAD. Never run from numbered agent worktrees.
2. Read [setup, private target schema, and commands](references/commands-and-targets.md) when preparing an actual browser run. The helper must keep credentials private, reject mismatched runtime identity, and capture only the approved target.
3. Stop when the focused question is resolved, run the normal affected validation, and report material evidence or gaps. Screenshots cannot settle human acceptance.

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

- Verify that the selected service has non-empty credential fields without printing values.
- Probe `<selected-base-url>/api/auth/login` through the same approved frontend proxy before blaming the browser.
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
- `scripts/browser_agent.py` resolves an approved private target, optionally authenticates with its service-local credentials, applies simple actions, waits, writes screenshots, and optionally writes visible page text.
- `scripts/cleanup.sh` removes one session, all browser artifacts, or artifacts older than a given age.

## Boundaries

- Do not add Playwright to `web/package.json` or create a second frontend test baseline for this skill.
- Do not treat Playwright MCP as the final browser artifact path; use it to discover stable actions and then capture
  evidence with `browser_agent.py`.
- Do not treat screenshots as acceptance by themselves; they are inspection evidence.
- Do not commit `.ai/venv`, `.ai/ms-playwright`, or `.ai/artifacts/browser`.
- Do not commit, print, or copy `.ai/private/graft-browser-targets.yaml`; never store runtime identity as trusted config.
- Prefer `data-testid`, stable text, role selectors, or TDesign-visible labels for actions. Avoid brittle generated class selectors when a stable selector exists.
