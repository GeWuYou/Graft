# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 implementation batch

- Long-running `optimize` work, `docs/automation`; active topic only. `kind: optimize` extends the ADR-003 v1 vocabulary; ADR owner to reconcile.
- Evidence reuse (SHA-bound plans, fail-closed comparisons), grouped check suite (requires, deadline, JSON summary), conservative CI impact mapping, and the `Required checks passed` run/attempt/SHA cross-check; push runs never cancelled.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c; corrections 8cb4d491c, 8bc145d60, d6b912342, 8977741a0, 56f7df274 (GET pin), ae83ae328 (PR head SHA binding). Detail in trace git history; findings `fixed`, stale re-post `stale`.

## 2026-11-20 PR 5 batch (preview lifecycle + deployment consistency + release candidates)

- Branch `chore/preview-release-consistency` (base 65bf4c6c0); task class `docs/automation`; recovery source this topic.
- `preview.py`: deterministic identity; registry/dirs/archives in the git common directory; port conflicts refused; `PREVIEW_MAX_RUNNING` cap; stop retains data; destroy needs exact `--confirm-instance`; status JSON binds URLs, health, logs, source digests.
- `preview_recovery.py`: read-only dry run; anomalies exit 1; suggestions gated on ownership proof (registered checkout, or identity+task claim only repository-bound, fail-closed); archive digests verified.
- `check_deployment_consistency.py`: HTML + all JS/CSS/modulepreload assets; "no acceptable 5xx" by default, 4xx never; Content-Type per kind; pages without entry assets fail; revision markers fail closed when expected.
- `release-candidate.yml`: pinned trivy; ghcr `candidate-v<version>-<sha>`; `release.json` binds run/attempt/SHA/version/channel/digests; summary replicates the PR #374 discipline and tags `rc-v<version>[-beta]` idempotently; candidate tags avoid `v*`.
- Justfile preview/release section; docs in `docs/development/preview-lifecycle.md`, `docs/operations/release-deployment-consistency.md`.
- Validation: 27+16+18 unit tests OK; actionlint clean; heredoc compiles; `git diff --check` clean.
- Commits: 25613647d, 69a9123d0, 30a0617d7, 3d8a622ab, 68dcd57c7, ff5ca9070.

## 2026-11-20 Corrections 1-2 (CI findings on PR #377; correction 1 budget 2, correction 2 final)

- Correction 1 (7 fixed, 1 partial accept): recovery suggestions use `--identity`; orphan baseline from the full registry before the identity filter; IDENTITY=TASK_ID claim format documented and parser-tested; restart re-checks `PREVIEW_MAX_RUNNING`; `--image-tag` bare-tag validation; status-accepted assets report `accepted_status: true` and skip content verification; release workflow adds a non-blocking unfixed scan whose artifact digests bind into `release.json` and are verified by the summary; docs aligned. Commits: 2cd8ed20d, 35b13d50a, e832f0f77, 476f49f1b, a93444916, 27bad57c1.
- Correction 2 (5 P1 threads, all real): candidate binary lands in the workspace release-binary context with a `-f` guard (4634415765); registry read-modify-write serialized by a real flock with allocation and cap reservation inside the lock plus unique temp names (4634415791); instance-scoped backup host path (4634415825); unhealthy marked-running instances restore via compose up (4634415848); destroy reclaims container-owned data via the stack postgres image and keeps the entry with a `cleanup_failed` marker on failure, retryable (4634415808). Commits: 4adf9e6d4, 09848a85b.
- Validation: 74 tests OK (three modules), actionlint exit 0, 5 heredocs compile, `git diff --check` clean, `just -n` OK.

## Next Steps

- After lead integration: observe a real pull-request run and one release-candidate dispatch.
- Primary-checkout follow-ups (commands in the docs): preview smoke, deployment consistency run with `--expect-revision` once web declares markers.
- Later batch: gating workflow jobs with `scripts/ci_impact.py` output; needs its own scope.
- ADR owner: reconcile the `kind: optimize` vocabulary.
