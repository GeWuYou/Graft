# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 implementation batch

- Long-running `optimize` work, `docs/automation`; active topic only (no design, roadmap, or ADR). `kind: optimize` extends the ADR-003 v1 vocabulary; left for the ADR owner to reconcile.
- Evidence reuse: plans bind head SHA and SHA-256 digests with per-file reasons and coverage; plan mode fails on empty selection, missing tests, or coverage gaps; `--compare-report` verifies previous reports fail-closed; `--report-out` persists executed reports; `just check-local-verify` verifies then executes.
- Check suite: six groups from Justfile/workflow reality; cross-group `requires=` with auto-include and cycle/unknown rejection; independent failures; process-level deadline with process-group kill; JSON + terminal summary; self-ignored logs under `.artifacts/checks/<group>/`.
- CI impact: conservative path-to-job mapping; unknown/workflow/script/toolchain/image surfaces force FULL; subsets always include hygiene and summary jobs. Advisory only.
- Workflow: grouped jobs; `web-check` timeout; `Required checks passed` needs every other job and verifies every job of the current run/attempt/SHA via the Actions API with only the push-event Secret Scan skip allowed; job names unchanged; push runs never cancelled.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c. The lead pushes and integrates.

## 2026-11-20 Correction 1 (contract governance P0 + Greptile findings)

- P0: cross-check step hard-coded an `Authorization` header literal; rewritten onto `gh api` with `GITHUB_TOKEN` from the environment. Branch-scoped scan (`GITHUB_BASE_REF=main`) reports scanned_files=7 with no active findings; tests forbid `Authorization`/`Bearer` literals and pin the gh path.
- P1: restored the `check-governance` body an earlier edit had left empty, so `just check` keeps the validators, Python tests, and Turbo tests; list extended with the two new tool test modules.
- P1: derived python/node consumer tests were omitted from digests/selection; now bound, with round-trip tests.
- P2: `--json` keeps stdout as pure report JSON (progress/summary on stderr); external absolute `--artifacts-dir` no longer crashes report-path resolution.
- Tooling note: just 1.57.0 swallows bare `var=...` lines in `[positional-arguments]` recipes; recipes use `"$1"` and `"${@:2}"`.
- Commits: 8cb4d491c, 8bc145d60, d6b912342, 8977741a0. Validation: 100 tests across nine script modules OK; actionlint clean; scanner clean; plan/verify round trip green; `check-governance` runs its gates and stops only at this worktree's missing `node_modules` (`just setup`).

## 2026-11-20 Correction 2 (gh api method pin; stale finding disposition)

- Confirmed by CI: `gh api` switches to POST when field flags are present, and the Actions jobs-list endpoint only supports GET, so the cross-check failed with `gh: Not Found (HTTP 404)` on run 44589471488. Fixed by pinning `--method GET` with `-f` query fields; pagination, run/attempt/SHA binding, the push-event Secret Scan skip, and fail-closed behavior unchanged. Verified locally against the failed run: the old shape returns HTTP 404, the fixed shape returns the run's 14 jobs including the summary job with the correct head SHA. Regression test pins `--method GET`, forbids `-F` on this call, and asserts the query fields. Commit 56f7df274.
- Disposition: `stale` — Greptile P1 re-posted "check-governance now has no commands" was already fixed in correction 1 (8bc145d60): `just -n check-governance` on HEAD shows the full body (three validators, the extended Python test bundle, and the Turbo tests), and the lead verified the restored recipe. No repair made; the Justfile is untouched by this correction.

## Next Steps

- Observe a real pull-request run with the fixed cross-check after the lead pushes.
- Later batch: evaluate gating workflow jobs with `scripts/ci_impact.py` output; that changes CI behavior and needs its own scope and acceptance.
