# Validation Delivery Tooling

## Current Status Summary

- Topic objective: make scoped local validation evidence reusable and make pull-request validation runs disciplined, grouped, and fully verified without weakening required gates; extend the same discipline to preview instance lifecycle, deployment consistency and the release candidate flow.
- Current status: `active`
- Task class: `docs/automation`
- Intake summary: long-running optimize topic covering evidence reuse, a grouped check-suite runner, conservative CI impact selection, pull-request workflow discipline, and the preview/release consistency batch (PR 5).
- Canonical authority: `scripts/local_validation.py`, `scripts/check_suite.py`, `scripts/ci_impact.py`, `scripts/preview.py`, `scripts/preview_recovery.py`, `scripts/check_deployment_consistency.py`, `.github/workflows/pull-request-validation.yml`, `.github/workflows/release-candidate.yml`, `docs/development/validation.md`, `docs/development/preview-lifecycle.md`, `docs/operations/release-deployment-consistency.md`.
- Completed so far: PR 3 slices committed on `chore/validation-ci-discipline` (merged as PR #374); PR 5 slices committed on `chore/preview-release-consistency`.
- Not started yet: observe a real pull-request run and one release-candidate dispatch; primary-checkout preview smoke and deployment consistency runs; decide whether `ci_impact` selection should later gate workflow jobs.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `docs/automation`
- recovery source: none (new topic)
- authority summary: root AGENTS.md owns validation phases; the validation tooling scripts own scoped evidence semantics; the pull-request workflow owns CI job orchestration and the run summary gate.

## Owned Scope

- `scripts/local_validation.py` and `scripts/test_local_validation.py`
- `scripts/check_suite.py`, `scripts/ci_impact.py` and their unit tests
- `.github/workflows/pull-request-validation.yml`
- `Justfile` recipes for the validation entrypoints
- `docs/development/validation.md` validation-related sections
- `ai-plan/public/validation-delivery-tooling/**`

Out of scope: `server/`, `web/src/`, `openapi/`, `scripts/validate_docker_isolated.py`, other workflow files, skill or agent changes, and remote GitHub settings writes.

## Locked Decisions

1. Scoped validation never silently falls back to full gates; empty selection, missing tests and coverage gaps stay explicit failures.
2. Evidence reuse is SHA-bound: recorded head SHA plus selected file digests must match or the comparison fails with per-file reasons.
3. CI impact selection is advisory and conservative by construction; CI always executes the workflow as defined.
4. Job names in pull-request validation remain unchanged so branch protection keeps working; push validation on main is never cancelled.

## Current Recovery Point

- Branch `chore/preview-release-consistency` carries the PR 5 implementation commits (preview lifecycle, recovery, deployment consistency, release-candidate workflow, docs); the lead integrates.
- Next step: after integration and push, observe one pull-request run with the fixed cross-check and one release-candidate dispatch, then run the documented primary-checkout follow-ups; record results in the trace.
