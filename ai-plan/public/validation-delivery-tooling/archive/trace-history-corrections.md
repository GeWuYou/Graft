# Validation Delivery Tooling Trace History (archived 2026-11-20)

Archived from traces/validation-delivery-tooling-trace.md to keep the active
trace within the ai-plan size budget. Content preserved verbatim from the
correction entries recorded on 2026-11-20.

## Corrections 1-2 (cross-check hardening)

- Correction 1: cross-check rewritten from a hard-coded `Authorization` header onto `gh api` with `GITHUB_TOKEN`; restored the emptied `check-governance` recipe; bound derived python/node consumer tests into digests/selection; fixed `--json` stdout purity and external `--artifacts-dir` resolution. Commits: 8cb4d491c, 8bc145d60, d6b912342, 8977741a0.
- Correction 2: `gh api` with field flags defaults to POST and the jobs-list endpoint only supports GET, so the cross-check failed with HTTP 404 on run 44589471488; pinned `--method GET` with `-f` fields, semantics unchanged; regression test pins GET and forbids `-F`. Commit 56f7df274. Re-posted check-governance finding disposed `stale`.
- Severity split, tooling notes and validation evidence: summarized in the tracking file, full text in the trace's git history.

## Correction 3 (PR head SHA binding)

- Confirmed by CI: on pull_request events `github.sha` is the merge commit (3652e9e33) while every job's `head_sha` in the Actions jobs-list is the PR branch head (763b05fdd), so the `job.head_sha != HEAD_SHA` check failed for all 13 jobs (run 44592526857).
- Fixed: `HEAD_SHA: ${{ github.event.pull_request.head.sha || github.sha }}` — PR head on pull_request events, `github.sha` otherwise; GET with `-f` fields, conclusion==success, push-only Secret Scan skip, fail-closed behavior and the summary-job exclusion unchanged. Regression test pins the binding and the fallback.
- Disposition: fixed. Commit: ae83ae328 (fix). Validation: nine-module bundle 100 tests OK; actionlint clean; heredoc compile check clean.
