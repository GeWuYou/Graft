# Validation Delivery Tooling Trace History (archived 2026-11-20)

Archived from traces/validation-delivery-tooling-trace.md to keep the active
trace within the ai-plan size budget. Content preserved verbatim from the
entries recorded on 2026-11-20.

## PR 3 implementation batch detail

- Long-running `optimize` work, `docs/automation`; active topic only (no design, roadmap, or ADR). `kind: optimize` extends the ADR-003 v1 vocabulary; left for the ADR owner to reconcile.
- Evidence reuse: plans bind head SHA and SHA-256 digests with per-file reasons and coverage; plan mode fails on empty selection, missing tests, or coverage gaps; `--compare-report` verifies previous reports fail-closed; `--report-out` persists executed reports; `just check-local-verify` verifies then executes.
- Check suite: six groups from Justfile/workflow reality; cross-group `requires=` with auto-include and cycle/unknown rejection; independent failures; process-level deadline with process-group kill; JSON + terminal summary; self-ignored logs under `.artifacts/checks/<group>/`.
- CI impact: conservative path-to-job mapping; unknown/workflow/script/toolchain/image surfaces force FULL; subsets always include hygiene and summary jobs. Advisory only.
- Workflow: grouped jobs; `web-check` timeout; `Required checks passed` needs every other job and verifies every job of the current run/attempt/SHA via the Actions API (push-event Secret Scan skip only); job names unchanged; push runs never cancelled.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c. The lead pushes and integrates.

## Corrections 1-2 (cross-check hardening)

- Correction 1: cross-check rewritten from a hard-coded `Authorization` header onto `gh api` with `GITHUB_TOKEN`; restored the emptied `check-governance` recipe; bound derived python/node consumer tests into digests/selection; fixed `--json` stdout purity and external `--artifacts-dir` resolution. Commits: 8cb4d491c, 8bc145d60, d6b912342, 8977741a0.
- Correction 2: `gh api` with field flags defaults to POST and the jobs-list endpoint only supports GET, so the cross-check failed with HTTP 404 on run 44589471488; pinned `--method GET` with `-f` fields, semantics unchanged; regression test pins GET and forbids `-F`. Commit 56f7df274. Re-posted check-governance finding disposed `stale`.
- Severity split, tooling notes and validation evidence: summarized in the tracking file, full text in the trace's git history.

## Correction 3 (PR head SHA binding)

- Confirmed by CI: on pull_request events `github.sha` is the merge commit (3652e9e33) while every job's `head_sha` in the Actions jobs-list is the PR branch head (763b05fdd), so the `job.head_sha != HEAD_SHA` check failed for all 13 jobs (run 44592526857).
- Fixed: `HEAD_SHA: ${{ github.event.pull_request.head.sha || github.sha }}` — PR head on pull_request events, `github.sha` otherwise; GET with `-f` fields, conclusion==success, push-only Secret Scan skip, fail-closed behavior and the summary-job exclusion unchanged. Regression test pins the binding and the fallback.
- Disposition: fixed. Commit: ae83ae328 (fix). Validation: nine-module bundle 100 tests OK; actionlint clean; heredoc compile check clean.
