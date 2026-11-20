# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 batch

- `optimize`, `docs/automation`; `kind: optimize` extends the ADR-003 v1 vocabulary (ADR owner to reconcile).
- Evidence reuse (SHA-bound, fail-closed), grouped suite, conservative CI impact, `Required checks passed` run/attempt/SHA cross-check with PR-head binding.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c; corrections 8cb4d491c, 8bc145d60, d6b912342, 8977741a0, 56f7df274, ae83ae328. Detail in git history and `archive/trace-history-corrections.md`.

## 2026-11-20 PR 5 batch (preview lifecycle + deployment consistency + release candidates)

- `preview.py`: identity, common-dir registry, port refusal, cap, guarded destroy. `preview_recovery.py`: read-only reconciliation. `check_deployment_consistency.py`: no-acceptable-5xx default. `release-candidate.yml`: pinned trivy, candidate tags, `release.json` binding, idempotent `rc-` tags avoiding `v*`. Justfile section; two docs.
- Commits: 25613647d, 69a9123d0, 30a0617d7, 3d8a622ab, 68dcd57c7, ff5ca9070.

## 2026-11-20 PR 4 batch (browser acceptance harness)

- `browser_acceptance.py` (tiered suites, zero retries, sanitized reports), `browser/serve_dist.py`, `tests/browser/` overlay, test-only seeds, docs.
- Commits: c1501a2b5, 77612a329, 9c335138c, 1c3b193f5, 997d43575, 16fe0722b.

## 2026-11-20/21 PR 5 corrections 1-7 (standing-authorized per D1)

- C1 (7 fixed, 1 partial): `--identity` suggestions, orphan baseline, IDENTITY=TASK_ID claims, restart cap, bare tag, `accepted_status`, unfixed-scan evidence, docs. Commits: 2cd8ed20d, 35b13d50a, e832f0f77, 476f49f1b, a93444916, 27bad57c1.
- C2 (5 P1 real): workspace binary path (4634415765); registry flock (4634415791); scoped backups (4634415825); unhealthy restore (4634415848); retryable `cleanup_failed` (4634415808). Commits: 4adf9e6d4, 09848a85b.
- C3 (3 P1 real): post-redirect final-URL assets; missing-files destroy by recorded project, retryable; recovery withholds suggestions for running-or-unknown stacks. Commits: 2fb131273, f562536fd, 2666dc4b7.
- C4 (1 P1 real, 4645472263): compose child env strips every key the instance `.env` owns plus file-selection controls; exported settings cannot override port, image or volumes.
- C5 (1 P1 real, live smoke repro): container-owned data removed inside the privileged container instead of re-rooting for the host; `status --all` reports unreadable dirs.
- C6 (2 P1 real): checkout read failures report `checkout_compose_unreadable` without touching the instance verdict; destroy chowns root-owned leftovers back to the invoking uid:gid.
- C7 (3 P1 real, threads 4646780242/4646780262/4646780255): checkout is_file probe inside its try (unsearchable dirs report `checkout_compose_unreadable`, status --all survives); missing dirs skip the image-dependent restore (label teardown); destroy tears down BEFORE the ownership restore, stack uids never flagged.
- Validation after C7: 96 tests OK, `git diff --check` clean.

## 2026-11-20 PR 4 corrections 1-3 (1 lead-verified + 2 standing-authorized per D1)

- C1 (Secret Scan + 15 findings, all real): invented-DSN fixture provenance in `archive/trace-history-corrections.md`; eda72736d streaming proxy, 39ab2e5f5 authenticated fixture session, 45 unit tests with in-process proxy servers.
- C2: teardown test contained in `TemporaryDirectory`; unused `browserEnv` import removed (CodeQL). Commit 915b507b0.
- C3: per-attempt session state files with regression-pinned harness↔TS contract; daemon suites assert live list state on real pages. Commit 501b0e6ed.

## Next Steps

- After integration: observe a real pull-request run and one release-candidate dispatch.
- Primary-checkout follow-ups (commands in docs): preview smoke, deployment consistency with `--expect-revision` once web declares markers, chromium local-tier browser run.
- Later batch: gating workflow jobs with `ci_impact.py` output (own scope).
- ADR owner: reconcile the `kind: optimize` vocabulary.
