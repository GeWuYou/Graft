# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 implementation batch

- Long-running `optimize` work, `docs/automation`; active topic only (no design, roadmap, or ADR). `kind: optimize` extends the ADR-003 v1 vocabulary; left for the ADR owner to reconcile.
- Evidence reuse: plans bind head SHA and SHA-256 digests with per-file reasons and coverage; plan mode fails on empty selection, missing consumer tests, or coverage gaps; `--compare-report` verifies previous reports fail-closed; `--report-out` persists executed reports; `just check-local-verify` verifies then executes. Always-zero plan exit was a documented behavior change.
- Check suite: six groups derived from Justfile/workflow reality; cross-group `requires=` with auto-include and cycle/unknown rejection; independent failures; process-level deadline with process-group kill; JSON + terminal summary; self-ignored per-check logs under `.artifacts/checks/<group>/`.
- CI impact: conservative path-to-job mapping; unknown/workflow/script/toolchain/image surfaces force FULL with reasons; subsets always include hygiene and summary jobs. Advisory only; CI executes the workflow as defined.
- Workflow: grouped jobs; `web-check` timeout; `Required checks passed` needs every other job and verifies every job of the current run/attempt/SHA via the Actions API with only the push-event Secret Scan skip allowed; job names unchanged for branch protection; push runs never cancelled (pipeline-governance locked decision preserved).
- Tooling note: just 1.57.0 swallows bare `var=...` lines in `[positional-arguments]` recipes; recipes use `"$1"` and `"${@:2}"`.
- Commits (base 270c2fb3c): 2edcdc3f6 evidence reuse, 7902543de check suite + CI impact, 2e0cc9734 workflow discipline, c621ce40c docs and topic files. The lead pushes and integrates.

## 2026-11-20 Correction 1 (contract governance P0 + Greptile findings)

- P0: the cross-check step hard-coded an `Authorization` header literal; rewritten onto `gh api` with `GITHUB_TOKEN` from the environment, keeping pagination, run/attempt/SHA binding, the push-event Secret Scan skip, and fail-closed behavior. Branch-scoped scan (`GITHUB_BASE_REF=main`) reports scanned_files=7 with no active findings; workflow tests now forbid `Authorization`/`Bearer` literals and pin the gh api path.
- P1: restored the `check-governance` body an earlier edit had left empty (its fix commit replaced the recipe body instead of inserting before it), so `just check` keeps the validators, Python tests, and Turbo tests; the list now also covers the two new tool test modules, matching the governance suite group and the CI test command.
- P1: derived python/node consumer tests were reported in plan tests but omitted from digests/selection, so a report from selecting only `scripts/check_suite.py` rejected its own derived test as uncovered. Derived tests now bind into digests and selection, with round-trip tests for python, node, toolchain, and a full execute/compare cycle.
- P2: suite progress and summary route through a progress stream (`--json` keeps stdout as pure report JSON); an external absolute `--artifacts-dir` no longer crashes report-path resolution.
- Commits: 8cb4d491c, 8bc145d60, d6b912342, a5dc31e7f (this trace).
- Validation: 100 tests across nine script test modules OK; actionlint exit 0; governance scan clean; real-selection plan/verify round trip passes and uncovered selections still fail with reasons; `check-governance` runs its gates and stops only at the worktree's missing `node_modules` prerequisite (`just setup`); cross-check heredoc compile-checked.

## Next Steps

- Observe one real pull-request run with the extended `Required checks passed` verification after the lead integrates and pushes.
- Later batch: evaluate gating workflow jobs with `scripts/ci_impact.py` output; that changes CI behavior and needs its own scope and acceptance.
