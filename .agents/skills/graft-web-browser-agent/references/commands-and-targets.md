# Browser Setup and Commands

Read only for an authorized primary-checkout run. Paths and commands are relative to the repository root.

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
