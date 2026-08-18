---
name: graft-web-browser-agent
description: Repository-specific Playwright browser workflow for inspecting, authenticating into, and interacting with an explicitly approved Graft web target. Use when Codex needs browser screenshots, DOM text snapshots, login verification with private local credentials, or simple click/fill/wait checks before normal web validation.
---

# Graft Web Browser Agent

## Overview

Use this skill to give Codex an eyes-on-browser loop for Graft `web` work. It is an observation and interaction aid only; it does not replace `web/AGENTS.md` or the required `bun run check` validation for frontend changes.

Follow root `AGENTS.md` startup governance before using this skill. For frontend implementation tasks, also follow `web/AGENTS.md` and `graft-web-vibe-coding`; this skill only adds browser inspection capability after the normal frontend authority and design rules are in force.

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

1. Confirm explicit authorization and the browser-only question. From the approved developer-owned primary checkout,
   confirm that the selected service is already running and serves the branch and full `HEAD` under review. Do not
   infer a target from current WSL networking, an occupied port, or a previous run.
2. Bootstrap the project-local browser environment if `.ai/venv/bin/python` or Playwright is missing:

```bash
.agents/skills/graft-web-browser-agent/scripts/bootstrap.sh
```

If bootstrap reports missing Chromium system dependencies, do not claim browser inspection is available yet. Report the printed `playwright install-deps chromium` command to the user; installing those packages is an explicit machine-level action.

3. Initialize the fixed private target registry on first use:

```bash
.ai/venv/bin/python .agents/skills/graft-web-browser-agent/scripts/browser_agent.py --init-config
```

The command creates `.ai/private/graft-browser-targets.yaml` only when absent, refuses to overwrite it, and stops.
Normal invocation also creates the template and stops when the file is missing. Fill the private file before retrying.
It is ignored by Git and must never be printed, committed, or copied into browser artifacts.

Use this schema; add as many environments, instances, and services as the developer has approved:

```yaml
schema_version: 1
defaults:
  environment: local
  instance: primary
  service: web
environments:
  local:
    instances:
      primary:
        services:
          web:
            base_url: "http://127.0.0.1:3002"
            credentials:
              username: ""
              password: ""
```

`credentials` is optional unless `--login` is used. Runtime identity is deliberately absent: dynamically verify it
from the primary checkout on every run and pass the current branch and full `HEAD` in `--runtime-identity`.

4. Run `browser_agent.py` against the selected target. CLI selectors override only the corresponding config defaults.
   `--url` is an approved, non-login, one-run path override and is never persisted; its scheme, host, and port must
   exactly match the selected service `base_url`. Authenticated runs must use the registered `base_url` and cannot
   combine `--login` with `--url`:

```bash
.ai/venv/bin/python .agents/skills/graft-web-browser-agent/scripts/browser_agent.py \
  --environment local \
  --instance primary \
  --service web \
  --runtime-identity "primary-web <verified-branch> <verified-full-head>" \
  --session ui-inspection \
  --screenshot \
  --snapshot-text
```

5. For authenticated Graft admin screenshots, place the credentials only on the selected service entry in the private
   config and let the script verify login before capture:

```bash
.ai/venv/bin/python .agents/skills/graft-web-browser-agent/scripts/browser_agent.py \
  --environment local \
  --instance primary \
  --service web \
  --runtime-identity "primary-web <verified-branch> <verified-full-head>" \
  --login \
  --session auth-check \
  --screenshot \
  --snapshot-text
```

Replace both placeholders with the branch and full `HEAD` verified in the clean primary checkout before running the
command. The script rejects missing, mismatched, or unsafe labels before importing or launching Playwright.

The helper reads only `credentials.username` and `credentials.password` from the selected private service entry.
`summary.json` may record non-secret selectors, relative navigation paths, and authentication status, but never the
configured `base_url`, URL origin/query, credential fields, or credential values. Do not place secrets in `--url`,
`--fill`, `--runtime-identity`, session names, screenshots, or visible page text.

6. Use focused interactions when debugging UI behavior:

```bash
.ai/venv/bin/python .agents/skills/graft-web-browser-agent/scripts/browser_agent.py \
  --environment local --instance primary --service web \
  --url http://127.0.0.1:3002/audit/logs \
  --runtime-identity "primary-web <verified-branch> <verified-full-head>" \
  --session audit-filter-check \
  --click "text=Filter" \
  --fill "input[placeholder='Keyword']=admin" \
  --wait-ms 500 \
  --screenshot
```

7. Use the browser evidence to guide fixes, then run the normal repository validation required by the changed scope.
   Stop after the focused question is resolved; do not repeat unrelated interaction flows for generic UI confidence.

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
