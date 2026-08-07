# Theme Preset Governance Audit

## Current Status Summary

- Topic objective: verify every built-in theme preset is a fully reproducible writable initial configuration, publish the audit, and complete the authorized authority-first remediation.
- Current status: `active`
- Task class: `web`
- Intake summary: long-running audit because the requested durable topic and multi-agent evidence wave need a recovery record.
- Canonical authority:
  - `web/src/config/theme-workbench.ts`
  - `web/src/store/modules/setting.ts` and `web/src/store/modules/setting-theme-*.ts`
  - `web/src/layouts/components/theme-workbench/**` and `web/src/style/**`
- Completed so far: intake, full 21-preset audit, and the authorized remediation that materializes preset data into editable state.
- Current verification: focused Vitest regression coverage passes for store and Workbench catalog/panel behavior.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `web`
- recovery source: `parent topic`
- authority summary: editable setting state and its derived token/runtime outputs must own theme visuals; preset metadata may only initialize that state.

## Owned Scope

- `web/src/config/theme-workbench.ts`
- `web/src/store/modules/setting*`
- `web/src/layouts/components/theme-workbench/**`
- `web/src/style/**`
- related locales and tests
- `ai-plan/public/theme-preset-governance-audit/**`

Out of scope:

- Adding preset-specific CSS selectors, compatibility branches, or new theme capabilities.

## Locked Decisions

1. The report evaluates visual authority, not the aesthetic quality of individual preset palettes.
2. A preset ID, name, or category may support catalog presentation and attribution only; it cannot affect rendered visual output after application.
3. The final report is the durable baseline evidence; remediation records its disposition without rewriting the historical findings.

## Current Recovery Point

- Completed batches: `preset-authority-audit`, `persist-audit-report`, and `authority-first-remediation`.
- Repair result: preset token and Material values are materialized at application/reset time; runtime token resolution no longer reads a preset object. The visible preset-application scope is persisted as a command preference only and never participates in token composition.
- Validation note: `bun test` invokes Bun's native runner and is not suitable for Vue SFC tests in this project. The project `test:run` script uses Vitest and passes the focused suites.
- Latest validation: `bun run check` passes, including Vitest and the release build.
- Next step: perform archive-readiness evaluation or retain the topic for any future preset additions.

## Work Intake

- This topic was created through `Work Intake`.
- The full `Work Contract` is in `todos/theme-preset-governance-audit-tracking.md`.

## Pending Batch Direction

- Rerun the web completion entrypoint and perform archive-readiness evaluation.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
cd web && bun run check
```

## Loop Entry

- Preferred entry: `ai-plan/public/theme-preset-governance-audit/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-batch`
