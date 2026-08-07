# Theme Preset Governance Audit Trace

## 2026-08-07 Intake and evidence dispatch

- Classified the requested durable report as a long-running `web` audit topic because it needs a recovery record and a bounded multi-agent evidence wave.
- Confirmed that no active topic owns theme preset authority or Personalization Workbench reproducibility.
- Established writable theme state, token definitions, and shared derived CSS as the intended visual authority; `THEME_PRESET_DEFINITIONS` must be evaluated as an initializer only.
- Selected `$graft-multi-agent-batch` for three read-only, non-overlapping evidence slices: preset/token mapping, Workbench control reachability, and hidden runtime/CSS authority.

## 2026-08-07 Preset authority audit completed

- Audited all 21 preset definitions and persisted the matrix in `reports/theme-preset-governance-audit.md`.
- Found no application-rendering selector or branch keyed by a concrete preset ID; catalog-only preset identifiers are not visual runtime authority.
- Found that `selectedThemePresetId` remains a runtime token-composition input, and `preserveThemePersonalization` changes Material application after preset selection.
- Found 4 chart and 9 preset Material token keys missing from `THEME_TOKEN_DEFINITIONS`; Material base highlight/noise controls are also missing.
- Limited the result to governance recommendations. No web runtime, tests, locales, or styles were changed by this audit.

## Locked Decisions

- The report will not authorize implementation or preserve preset-specific compatibility logic.
- A direct preset read in runtime rendering is an authority finding even when the preset's values are otherwise editable.

## 2026-08-07 Authorized authority-first remediation

- Replaced runtime preset-object composition with one-time materialization into `ThemeAuthorityState`. `selectedThemePresetId` remains catalog and attribution metadata and no longer selects rendered token values.
- Replaced persisted `preserveThemePersonalization` with a Workbench application-scope preference: palette selection preserves existing editable appearance/style values, while complete selection writes the entire authority/style/token snapshot. The preference is persisted for the visible control but is excluded from drafts and runtime token composition.
- Made reset-to-default a complete default-preset materialization.
- Registered the four Chart tokens, nine preset Material tokens, and shared glass highlight/noise tokens in `THEME_TOKEN_DEFINITIONS`; the Advanced editor exposes shared Chart and Material groups.
- Added regression coverage for state equivalence after preset-ID replacement and both preset application scopes. `bun test src/store/modules/setting.test.ts` passed 49 tests.
- The apparent component-test failures were caused by running `bun test`, which uses Bun's native runner rather than the repository's Vue SFC runner. `bun run test:run -- <focused files>` ran Vitest and passed 57 store/catalog/panel tests.
- `bun run check` passed the web completion chain, including formatting, type checks, i18n governance, lint/style checks, Vitest, and release build.

## 2026-08-07 Application Scope Persistence Repair

- The application-scope switch was initially component-local and reset to `palette` whenever the Workbench reopened.
- Moved it to persisted setting config while excluding it from `WORKBENCH_STYLE_CONFIG_KEYS`; closing or canceling a visual draft no longer discards the user's next-application preference.
- Focused Vitest coverage now verifies the persisted scope is outside visual draft rollback. It never becomes runtime visual authority.

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
