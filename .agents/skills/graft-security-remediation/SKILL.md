---
name: graft-security-remediation
description: Repository-specific workflow for triaging, remediating, validating, and closing Graft GitHub Code Scanning and Dependabot alerts, including branch hygiene, scoped fixes, commit/push/PR flow, and post-push security recheck.
---

# Graft Security Remediation

Use this skill when a task asks to inspect, classify, remediate, validate, or recheck:

- GitHub `security/code-scanning`
- GitHub `security/dependabot`
- related GitHub Advanced Security findings surfaced through those alert APIs

Treat root `AGENTS.md` as the startup, authority, validation, commit, and closeout source of truth. This skill adds a
security-specific workflow plus reusable inventory assets; it does not create a second release path.

## Workflow

This skill has two explicit stages:

- `inventory/review` is the default and is read-only. It may query alerts, inspect local code, classify findings, and
  propose validation, but it must not edit files, dismiss alerts, commit, push, create/update a PR, or post comments.
- `remediation/write` starts only when the user explicitly requests local fixes or a specific external write. Local
  remediation authority does not implicitly authorize GitHub writes, commits, or pushes; apply each owning workflow's
  authorization gate independently.

1. Complete the startup preflight from root `AGENTS.md` and classify the task as `server`, `web`, `cross-boundary`, or
   `docs/automation`.
2. Read the required subdomain `AGENTS.md` before edits, then inspect the current worktree with `git status --short`.
3. Build one full open-alert inventory before planning or fixing:

   ```bash
   python3 .agents/skills/graft-security-remediation/scripts/inventory_open_alerts.py \
     --repo GeWuYou/Graft \
     --kind all \
     --output - > /tmp/graft-security-alerts.json
   ```

   Read [references/inventory.md](references/inventory.md) when you need the normalized schema, `gh` examples, or the
   offline import path.
4. Classify every alert as exactly one of:
   - `real-vulnerability`
   - `false-positive`
   - `needs-investigation`
   - `already-remediated-locally`
   - `blocked`
5. Publish a concise remediation plan. If the user requested only inspection, stop after the inventory/review report:
   - in-scope alerts
   - classification evidence
   - intended authority-first repair
   - validation commands
   - branch / commit / recheck path
6. Enter `remediation/write` only under explicit authorization, then remediate with the smallest authority-correct fix:
   - verify `source -> sink` for code scanning alerts
   - verify direct vs indirect usage and patched version reality for dependency alerts
   - do not add suppressions, aliases, or compatibility layers unless root `AGENTS.md` exception rules are satisfied
7. Use `graft-validation-runner` and complete the full entrypoint for the touched runtime scope:
   - `server`: focused `go test` during iteration, then `graft validate backend`
   - `web`: `bun run check`
   - `cross-boundary`: validate both sides
   Focused tests and lint stages are supplemental; they do not replace a full completion entrypoint.
8. Recheck after local validation. Read [references/recheck.md](references/recheck.md) when you need the standard
   evidence checklist or post-push expectations.

## Guardrails

- Do not start with a partial subset of alerts when a full inventory is still missing.
- Do not claim `false-positive` or `already-remediated-locally` without code-level evidence from the current branch.
- Do not invent remote write steps as the default path. Push, PR creation, alert dismissal, and any GitHub write action
  require an explicit user request and the owning repository skill's authorization gate.
- Do not turn a read-only inventory request into local remediation because findings are actionable.
- Keep the owned scope limited to the confirmed remediation slice; do not bundle unrelated worktree changes.

## Closeout Evidence

```text
Security remediation evidence:
- inventory_source: gh-api | local-json | mixed
- alerts_reviewed: <count by kind>
- disposition_summary: real / false-positive / stale / blocked
- authority_repair: <canonical layer changed or no-change>
- validation: <commands and results>
- recheck_status: local-only | pushed-awaiting-github | github-confirmed
- stage: inventory/review | remediation/write
```
