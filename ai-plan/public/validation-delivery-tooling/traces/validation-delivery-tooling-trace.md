# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 implementation batch

- Long-running `optimize` work, `docs/automation`; active topic only. `kind: optimize` extends the ADR-003 v1 vocabulary; ADR owner to reconcile.
- Evidence reuse (SHA-bound plans, fail-closed comparisons), grouped check suite (requires, deadline, JSON summary), conservative CI impact mapping, and the `Required checks passed` run/attempt/SHA cross-check; push runs never cancelled.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c; corrections 8cb4d491c, 8bc145d60, d6b912342, 8977741a0, 56f7df274 (GET pin), ae83ae328 (PR head SHA binding). Detail in trace git history; findings `fixed`, stale re-post `stale`.

## 2026-11-20 PR 5 batch (preview lifecycle + deployment consistency + release candidates)

- Branch `chore/preview-release-consistency` (base 65bf4c6c0); task class `docs/automation`; recovery source this topic.
- `preview.py`: deterministic identity (checkout + task id, digest suffix); registry/dirs/archives in the git common directory; port conflicts refused; `PREVIEW_MAX_RUNNING` cap; stop retains data; destroy needs exact `--confirm-instance` and archives SHA-256 evidence; status JSON binds URLs, health, logs, source digests.
- `preview_recovery.py`: read-only dry run; anomalies exit 1; suggestions gated on ownership proof (registered checkout, or identity+task claim only repository-bound, fail-closed); archive digests verified.
- `check_deployment_consistency.py`: HTML + all JS/CSS/modulepreload assets; "no acceptable 5xx" by default with explicit opt-in, 4xx never; Content-Type per kind; pages without entry assets fail; revision markers fail closed when expected (web does not yet declare markers).
- `release-candidate.yml`: pinned trivy; ghcr `candidate-v<version>-<sha>`; `release.json` binds run/attempt/SHA/version/channel/digests; summary replicates the PR #374 discipline and tags `rc-v<version>[-beta]` idempotently (same SHA success, different SHA hard fail); candidate tags avoid `v*`.
- Justfile preview/release section; docs in `docs/development/preview-lifecycle.md`, `docs/operations/release-deployment-consistency.md`.
- Validation: 27+16+18 unit tests OK; actionlint clean; heredoc compiles; `git diff --check` clean.
- Commits: 25613647d, 69a9123d0, 30a0617d7, 3d8a622ab, 68dcd57c7, ff5ca9070.

## 2026-11-20 Correction 1 (CI findings on PR #377, budget 2)

- 7 repairs + 1 partial accept, all with tests.
- Recovery: suggestions emit `destroy --identity <identity>` (a lost worktree cannot re-derive identity from `--task`); orphan baseline uses the full registry before the `--identity` filter; `--claim-task-id` parsed as IDENTITY=TASK_ID with the documented example covered by a parser test.
- Preview: restart re-checks `PREVIEW_MAX_RUNNING` excluding itself; `--image-tag` rejects `/` and `:`.
- Deployment consistency (partial accept): status-accepted assets keep ok-semantics but report `accepted_status: true` and are not content-verified.
- Release workflow: non-blocking unfixed scan (`ignore-unfixed: false`) uploads the `release-candidate-unfixed-scan` artifact; digests bound in `release.json` `scan.unfixed`; missing evidence blocks registration; summary verifies digests; runbook aligned. Docs: `GRAFT_PREVIEW_PORT_BASE`, IDENTITY=TASK_ID, bare image tags.
- Commits: 2cd8ed20d, 35b13d50a, e832f0f77, 476f49f1b, a93444916. Disposition: 7 `fixed`, 1 partial-accepted with explicit report field.
- Validation: 68 tests OK (three modules); actionlint exit 0; 5 heredocs compile; `git diff --check` clean; `just -n` OK.

## Next Steps

- After lead integration: observe a real pull-request run and one release-candidate dispatch; record results here.
- Primary-checkout follow-ups (commands in the docs): preview smoke, deployment consistency run with `--expect-revision` once web declares markers.
- Later batch: gating workflow jobs with `scripts/ci_impact.py` output; needs its own scope.
- ADR owner: reconcile the `kind: optimize` vocabulary.
