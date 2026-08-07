# Theme Preset Governance Audit

## Current Status Summary

- Topic objective: audit whether every built-in theme preset is a fully reproducible writable initial configuration, then publish authority-first governance recommendations.
- Current status: `active`
- Task class: `web`
- Intake summary: long-running audit because the requested durable topic and multi-agent evidence wave need a recovery record.
- Canonical authority:
  - `web/src/config/theme-workbench.ts`
  - `web/src/store/modules/setting.ts` and `web/src/store/modules/setting-theme-*.ts`
  - `web/src/layouts/components/theme-workbench/**` and `web/src/style/**`
- Completed so far: intake, startup preflight, and initial authority discovery.
- Not started yet: complete preset-by-preset evidence collection and report synthesis.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `web`
- recovery source: `none`
- authority summary: editable setting state and its derived token/runtime outputs must own theme visuals; preset metadata may only initialize that state.

## Owned Scope

- `web/src/config/theme-workbench.ts`
- `web/src/store/modules/setting*`
- `web/src/layouts/components/theme-workbench/**`
- `web/src/style/**`
- related locales and tests
- `ai-plan/public/theme-preset-governance-audit/**`

Out of scope:

- Implementing the identified governance refactors.
- Adding preset-specific CSS selectors, compatibility branches, or new theme capabilities.

## Locked Decisions

1. The report evaluates visual authority, not the aesthetic quality of individual preset palettes.
2. A preset ID, name, or category may support catalog presentation and attribution only; it cannot affect rendered visual output after application.
3. The final report is the durable result for this topic and records only governance refactors.

## Current Recovery Point

- Completed batch: `preset-authority-audit` and `persist-audit-report`.
- Risk: the worktree still contains user-owned or in-progress theme implementation changes; this audit did not modify them.
- Next step: await an explicitly authorized governance-remediation scope.

## Work Intake

- This topic was created through `Work Intake`.
- The full `Work Contract` is in `todos/theme-preset-governance-audit-tracking.md`.

## Pending Batch Direction

- Complete the evidence matrix and governance findings.
- Persist the final Chinese report under `reports/theme-preset-governance-audit.md`, then update tracking and trace evidence.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/theme-preset-governance-audit/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-batch`
