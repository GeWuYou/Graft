# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 implementation batch

- Long-running `optimize` work, `docs/automation`; active topic only. `kind: optimize` extends the ADR-003 v1 vocabulary; left for the ADR owner.
- Evidence reuse: plans bind head SHA + per-file SHA-256 digests; plan mode fails on empty selection, missing tests or gaps; `--compare-report` fail-closed; `--report-out` persists reports; `just check-local-verify` verifies then executes.
- Check suite: six groups; cross-group `requires=` with auto-include and cycle/unknown rejection; independent failures; process-level deadline; JSON summary; self-ignored logs.
- CI impact: conservative mapping; unknown/shared surfaces force FULL; hygiene and summary always included. Advisory only.
- Workflow: grouped jobs; `Required checks passed` needs every job and verifies every job of run/attempt/SHA via the Actions API (push-only Secret Scan skip); names unchanged; push runs never cancelled.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c; corrections 8cb4d491c, 8bc145d60, d6b912342, 8977741a0 (gh api GET, stdout purity, consumer-test binding), 56f7df274 (GET pin), ae83ae328 (PR head SHA binding). Full correction detail in this trace's git history; findings `fixed`, stale re-post `stale`.

## 2026-11-20 PR 5 batch (preview lifecycle + deployment consistency + release candidates)

- Branch `chore/preview-release-consistency` (base 65bf4c6c0); task class `docs/automation`; recovery source this topic.
- `preview.py`: deterministic identity (checkout path + task id, digest suffix); registry, instance dirs and archives in the git common directory; port conflicts (foreign or registered) refused, never silently skipped; `PREVIEW_MAX_RUNNING` cap; `stop` retains data; `destroy` requires exact `--confirm-instance` and archives SHA-256 cleanup evidence; status JSON binds URLs, health, logs and source digests.
- `preview_recovery.py`: read-only dry run; anomalies exit 1; destroy suggestions gated on ownership proof (registered checkout, or identity+task claim only on a repository-bound registry, fail-closed); archive verification covers manifest SHA-256, inner evidence digest, unlisted and missing records; healthy running instances never suggested.
- `check_deployment_consistency.py`: entry HTML + all JS/CSS/modulepreload assets; default contract "no acceptable 5xx (503 included)", explicit `--acceptable-status` opt-in, 4xx never; Content-Type bindings per kind; pages without entry assets fail; revision/source-digest markers fail closed when expectations exist (web entry does not yet declare markers — documented).
- `release-candidate.yml`: versioned server linking + web release build; pinned trivy v0.36.0 scan (HIGH,CRITICAL; unfixed recorded); ghcr `candidate-v<version>-<sha>` push; `release.json` bound to run/attempt/SHA/version/channel/digests; summary replicates the PR #374 discipline then verifies the record and creates `rc-v<version>[-beta]` idempotently (same SHA success, different SHA hard fail). Candidate tags avoid `v*` so publish.yml stays out.
- Justfile gained a separate preview/release section; docs added in `docs/development/preview-lifecycle.md` and `docs/operations/release-deployment-consistency.md` (new directory).
- Validation: 27+16+18 unit tests via `python3 -B -m unittest` OK (in-process HTTP fixtures, tree-snapshot read-only proof); actionlint clean; YAML parse clean; 4 workflow heredocs compile; `git diff --check` clean; `just -n` parses the whole file.
- Commits: 25613647d, 69a9123d0, 30a0617d7, 3d8a622ab, 68dcd57c7. The lead pushes and integrates.

## Next Steps

- After lead integration: observe a real pull-request run with the fixed cross-check and one release-candidate dispatch; record results here.
- Primary-checkout follow-ups (commands in the docs): preview smoke, deployment consistency run with `--expect-revision` once web declares markers.
- Later batch: gating workflow jobs with `scripts/ci_impact.py` output; needs its own scope.
- ADR owner: reconcile the `kind: optimize` vocabulary.
