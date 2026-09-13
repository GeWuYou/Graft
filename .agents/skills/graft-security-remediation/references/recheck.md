# Recheck Reference

Use this reference when the remediation slice is ready for local closeout or for an explicit post-push GitHub recheck.

## Local Recheck

Before any remote step:

1. reuse passing validation for unchanged scope, or rerun relevant commands when a fix or new evidence invalidates it
2. rerun the inventory helper if classification depended on current open-alert counts
3. confirm each fixed alert now maps to one of:
   - `already-remediated-locally`
   - `false-positive`
   - `blocked`

Recommended local evidence:

- normalized inventory JSON before the fix
- normalized inventory JSON after the fix if the fix changed advisory interpretation
- code references for every `false-positive` or `already-remediated-locally` claim
- exact validation commands and results

## GitHub Recheck

Do not make GitHub write actions the default flow. When the user explicitly requests push or PR follow-up, or another
repository skill owns that path, capture:

- branch name used for the remediation slice
- commit SHA that contains the validated fix
- whether push / PR creation was requested explicitly
- whether GitHub still shows the alert as open, fixed, or awaiting analysis

If GitHub still shows the alert after a verified local fix, record one of:

- `analysis-lag`
- `wrong-alert-scope`
- `needs-manual-dismissal`
- `still-open-real-finding`

## Closeout Notes

A concise closeout should answer:

- what inventory source was used
- how many alerts were reviewed per kind
- which alerts were true positives vs false positives vs blocked
- which validation directly covered the changed scope
- whether the outcome is local-only or GitHub-confirmed
