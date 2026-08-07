# Theme Preset Governance Audit Tracking

## Topic

Theme Preset Governance Audit

## Scope

Audit every entry in `THEME_PRESET_DEFINITIONS` from preset definition through setting state, Workbench controls, token editor, derived root attributes, CSS, and rendered effect. Publish a durable Chinese report and governance-only recommendations; do not implement the recommendations.

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

- Completed batch: `preset-authority-audit` and `persist-audit-report`.
- Authority finding: runtime token composition still reads `selectedThemePresetId` and preset material token overrides directly.
- Next step: await an explicitly authorized governance-remediation scope; the required evidence is in `reports/theme-preset-governance-audit.md`.

## Task Checklist

- [x] Build the complete 21-preset mapping and token reachability inventory.
- [x] Audit manual Workbench control coverage and derived style rules.
- [x] Audit preset identity, DOM/CSS, and runtime hidden-authority branches.
- [x] Persist final Chinese report and recommendations.

## Acceptance Conditions

- Every preset has a mapping through its rendered visual effects and an explicit manual reproducibility result.
- Every preset-introduced token is classified as editable, missing, duplicated, or preset-only.
- The report identifies all hidden authority and gives only authority-first governance refactors.

## Loop Batch State

```json
{
  "loop_mode": "direct-specialized-skill",
  "completed_batches": ["preset-authority-audit", "persist-audit-report"],
  "pending_batches": [],
  "current_batch": "governance-remediation-decision",
  "next_batch": "await-authorized-remediation-scope",
  "closeout_status": "report-persisted"
}
```
