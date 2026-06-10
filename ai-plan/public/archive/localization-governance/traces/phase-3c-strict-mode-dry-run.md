# Phase 3C Strict Mode Dry Run

## Scope

- Batch: `Phase 3C Strict Mode Dry Run`
- Baseline commit: `d0ca10cd`
- Task class: `cross-boundary`
- Owned scope:
  - `web/scripts/check-i18n-governance.ts`
  - `web/scripts/i18n-governance/**`
  - `web/src/**/locales/**`
  - governance reports/docs needed for strict-mode evidence

## Strict Mode Configuration

- Flag: `STRICT_I18N_KEY_FIRST=true`
- Direct validation command: `cd web && STRICT_I18N_KEY_FIRST=true bun run lint:i18n`
- Strict behavior under dry run:
  - fallback-only server key-first registry findings are promoted from warnings to blockers
  - key + fallback pairs remain allowed because fallback copy is compatibility-only
  - no compatibility alias or rule weakening was introduced

## Result

- Status: passed
- Failures recorded: none
- Genuine governance violations fixed: none required
- Scope expansion: none

Validation output summary:

```text
$ bun run scripts/check-i18n-governance.ts
No hard-coded UI text or locale governance issues found.
```

## Follow-up

- Phase 3D may enable strict governance in CI, PR validation, local check workflow, and contributor guidance.
- Phase 3D should keep `bun run check` as the web validation source of truth and avoid creating a second i18n CI contract.
