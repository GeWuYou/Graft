# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 batch

- Long-running `optimize` work, `docs/automation`; `kind: optimize` extends the ADR-003 v1 vocabulary; ADR owner to reconcile.
- Evidence reuse (SHA-bound, fail-closed), grouped suite, conservative CI impact, `Required checks passed` run/attempt/SHA cross-check with PR-head binding; push runs never cancelled.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c; corrections 8cb4d491c, 8bc145d60, d6b912342, 8977741a0, 56f7df274, ae83ae328. Detail in trace git history and `archive/trace-history-corrections.md`.

## 2026-11-20 PR 5 batch (preview lifecycle + deployment consistency + release candidates)

- `preview.py`: deterministic identity; common-dir registry; port refusal; cap; guarded destroy with evidence archives. `preview_recovery.py`: read-only reconciliation, ownership-proof gating, digest verification. `check_deployment_consistency.py`: no-acceptable-5xx default, fail-closed revision markers. `release-candidate.yml`: pinned trivy, ghcr candidate tags, `release.json` binding, idempotent `rc-` tags avoiding `v*`. Justfile section; two docs.
- Commits: 25613647d, 69a9123d0, 30a0617d7, 3d8a622ab, 68dcd57c7, ff5ca9070.

## 2026-11-20 PR 4 batch (browser acceptance harness)

- `browser_acceptance.py` (suite/engine/tier registry, zero retries, sanitized per-attempt reports), `browser/serve_dist.py`, `tests/browser/` overlay + specs, `TestSeedBrowserFixture` test-only seeds, docs, Justfile section.
- Commits: c1501a2b5, 77612a329, 9c335138c, 1c3b193f5, 997d43575, 16fe0722b.

## 2026-11-20 PR 5 corrections 1-3 (1-2 lead-authorized, 3 standing-authorized per D1)

- C1 (7 fixed, 1 partial): identity suggestions, orphan baseline, IDENTITY=TASK_ID claims, restart cap, bare tag, `accepted_status`, non-blocking unfixed scan bound into `release.json`, docs. Commits: 2cd8ed20d, 35b13d50a, e832f0f77, 476f49f1b, a93444916, 27bad57c1.
- C2 (5 P1 real): workspace release-binary path (4634415765); registry flock + in-lock allocation/cap (4634415791); instance-scoped backups (4634415825); unhealthy running restored (4634415848); destroy reclaims container-owned data, retryable `cleanup_failed` (4634415808). Commits: 4adf9e6d4, 09848a85b.
- C3 (3 P1 real): assets resolve against the final URL after redirects; missing-files destroy tears down by recorded project, retryable; recovery withholds suggestions for running-or-unknown stacks. Commits: 2fb131273, f562536fd, 2666dc4b7.
- Validation after C3: 85 tests OK, actionlint 0, heredoc compile OK.

## 2026-11-20 PR 4 corrections 1-3 (1 lead-verified + 2 standing-authorized per D1)

- C1 (Secret Scan + 15 Greptile findings, all real): proof anchors in `archive/trace-history-corrections.md`. Secret Scan = invented DSN in the sanitization fixture (c1501a2b5, raw_sha256 f70d79c0…acf58), registered per synthetic-fixture policy in 7ac2f3c8c; current test composes the credential at runtime. eda72736d streaming proxy + per-run project + suite parsing; 39ab2e5f5 authenticated fixture session + envelope-correct specs; 45 unit tests incl. in-process proxy servers.
- C2: teardown test contained in `TemporaryDirectory`; unused `browserEnv` import removed (CodeQL). Commit 915b507b0.
- C3: per-attempt session state files with regression-pinned harness↔TS contract; daemon suites assert live list state on real pages. Commit 501b0e6ed.

## Next Steps

- After integration: observe a real pull-request run and one release-candidate dispatch.
- Primary-checkout follow-ups (commands in docs): preview smoke, deployment consistency with `--expect-revision` once web declares markers, chromium local-tier browser run.
- Later batch: gating workflow jobs with `ci_impact.py` output (own scope).
- ADR owner: reconcile the `kind: optimize` vocabulary.
