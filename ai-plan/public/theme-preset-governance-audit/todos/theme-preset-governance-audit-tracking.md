# Theme Preset Governance Audit Tracking

## Topic

Theme Preset Governance Audit

## Scope

Audit every entry in `THEME_PRESET_DEFINITIONS` and, after explicit authorization, remove identified runtime preset authority through writable settings state and shared editable token definitions.

## Repository Truth

- `AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/AGENTS.md`
- `DESIGN.md`
- `ai-plan/design/governance/frontend/前端视觉设计规范.md`
- `web/src/config/theme-workbench.ts`

## Work Contract

```yaml
version: 1
kind: audit
scope: long-running
authority_summary: "Writable theme authority state, editable token definitions, and shared derived CSS rules own rendered visuals; preset definitions may only materialize initial values into those owners."
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: direct-specialized-skill
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/public/theme-preset-governance-audit/README.md
    - ai-plan/public/theme-preset-governance-audit/startup-prompt.md
    - ai-plan/public/theme-preset-governance-audit/todos/theme-preset-governance-audit-tracking.md
    - ai-plan/public/theme-preset-governance-audit/traces/theme-preset-governance-audit-trace.md
closeout:
  archive: false
  lessons_review: true
```

## Current Recovery Point

- Completed batches: `preset-authority-audit`, `persist-audit-report`, and `authority-first-remediation`.
- Authority repair: `selectedThemePresetId` remains catalog/diff metadata only. Applying a preset or resetting materializes the authority/style/token snapshot; `buildThemeModeSnapshot()` consumes editable inputs only.
- Token repair: all Chart and Material values identified by the audit, including shared highlight/noise values, are registered in `THEME_TOKEN_DEFINITIONS` and exposed by the Advanced editor.
- Validation clarification: Vue SFC suites must run through `bun run test:run` (Vitest), not Bun's native `bun test` runner. The focused Vitest suites pass.
- Completion validation: `bun run check` passes. Next step: assess archive readiness; the required audit baseline remains in `reports/theme-preset-governance-audit.md`.

## Task Checklist

- [x] Build the complete 21-preset mapping and token reachability inventory.
- [x] Audit manual Workbench control coverage and derived style rules.
- [x] Audit preset identity, DOM/CSS, and runtime hidden-authority branches.
- [x] Persist final Chinese report and recommendations.
- [x] Materialize preset authority, regular token, and Material token values into editable state.
- [x] Remove preset identity and persisted application scope from runtime token composition.
- [x] Add shared editable Chart and Material token definitions and Workbench groups.
- [x] Establish focused Vitest evidence for store and Workbench catalog/panel behavior.
- [x] Run the web completion entrypoint.
- [ ] Reassess archive readiness.

## Acceptance Conditions

- Every preset has a mapping through its rendered visual effects and an explicit manual reproducibility result.
- Runtime rendering depends only on editable theme state and shared derived rules, not a preset identity.
- Every preset-introduced token is writable through the Advanced editor.
- Store and component regression evidence cover both application scopes and ID-independence.

## Loop Batch State

```json
{
  "loop_mode": "direct-specialized-skill",
  "completed_batches": ["preset-authority-audit", "persist-audit-report", "authority-first-remediation"],
  "pending_batches": ["archive-readiness-check"],
  "current_batch": "archive-readiness-check",
  "next_batch": "archive-or-retain-topic",
  "closeout_status": "remediation-and-web-validation-passed"
}
```
