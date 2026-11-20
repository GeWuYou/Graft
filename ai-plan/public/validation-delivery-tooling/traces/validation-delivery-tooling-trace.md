# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 implementation batch

- Long-running `optimize` work, `docs/automation`; active topic only. `kind: optimize` extends the ADR-003 v1 vocabulary; ADR owner to reconcile.
- Evidence reuse (SHA-bound, fail-closed), grouped suite (requires, deadline, JSON), conservative CI impact, `Required checks passed` run/attempt/SHA cross-check; push runs never cancelled.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c; corrections 8cb4d491c, 8bc145d60, d6b912342, 8977741a0, 56f7df274, ae83ae328. Detail in trace git history; findings `fixed`, stale re-post `stale`.

## 2026-11-20 PR 5 batch (preview lifecycle + deployment consistency + release candidates)

- Branch `chore/preview-release-consistency` (base 65bf4c6c0); task class `docs/automation`; recovery source this topic.
- `preview.py`: deterministic identity; registry/dirs/archives in the git common directory; port refusal; `PREVIEW_MAX_RUNNING` cap; stop retains data; destroy needs exact `--confirm-instance`; status JSON binds URLs, health, logs, digests.
- `preview_recovery.py`: read-only dry run; anomalies exit 1; suggestions gated on ownership proof and stack state; archive digests verified.
- `check_deployment_consistency.py`: HTML + all JS/CSS/modulepreload assets; "no acceptable 5xx" default, 4xx never; pages without entry assets fail; revision markers fail closed when expected.
- `release-candidate.yml`: pinned trivy; ghcr `candidate-v<version>-<sha>`; `release.json` binds run/attempt/SHA/version/channel/digests; summary replicates the PR #374 discipline and tags `rc-v<version>[-beta]` idempotently; candidate tags avoid `v*`.
- Justfile preview/release section; docs in `docs/development/preview-lifecycle.md`, `docs/operations/release-deployment-consistency.md`.
- Validation: 27+16+18 unit tests OK; actionlint clean; heredoc compiles; `git diff --check` clean.
- Commits: 25613647d, 69a9123d0, 30a0617d7, 3d8a622ab, 68dcd57c7, ff5ca9070.

## 2026-11-20 Corrections 1-3 on PR #377 (1-2 lead-authorized, 3 standing-authorized per D1)

- Correction 1 (7 fixed, 1 partial accept): `--identity` suggestions; full-registry orphan baseline; IDENTITY=TASK_ID claims; restart cap re-check; bare `--image-tag`; `accepted_status` reporting; non-blocking unfixed scan with digests bound into `release.json` and verified by the summary; docs aligned. Commits: 2cd8ed20d, 35b13d50a, e832f0f77, 476f49f1b, a93444916, 27bad57c1.
- Correction 2 (5 P1, all real): binary lands in the workspace release-binary context with `-f` guard (4634415765); registry flock serialization with in-lock allocation/cap reservation and unique temp names (4634415791); instance-scoped backups (4634415825); unhealthy running restored via compose (4634415848); destroy reclaims container-owned data and stays retryable via `cleanup_failed` (4634415808). Commits: 4adf9e6d4, 09848a85b.
- Correction 3 (3 P1 on head 483f00a3b, all real; worktree fast-forwarded to the lead merge first): assets resolve against the final URL after redirects with a redirecting-fixture test; destroy with missing files tears down by the recorded project name and never drops the entry while the stack may be live (teardown failure keeps `cleanup_failed`, retryable); recovery probes container state by project label and withholds suggestions for running-or-unknown stacks, requiring confirmed-dead with evidence. Commits: 2fb131273, f562536fd, 2666dc4b7.
- Validation after correction 3: 85 tests OK (three modules), actionlint exit 0, heredoc compile OK, `git diff --check` clean, `just -n` OK.

## Next Steps

- After lead integration: observe a real pull-request run and one release-candidate dispatch.
- Primary-checkout follow-ups (commands in the docs): preview smoke, deployment consistency run with `--expect-revision` once web declares markers.
- Later batch: gating workflow jobs with `scripts/ci_impact.py` output; needs its own scope.
- ADR owner: reconcile the `kind: optimize` vocabulary.
