# Validation Delivery Tooling

## Current Status Summary

- Topic objective: make scoped local validation evidence reusable and make pull-request validation runs disciplined, grouped, and fully verified without weakening required gates.
- Current status: `active`
- Task class: `docs/automation`
- Intake summary: long-running optimize topic covering evidence reuse, a grouped check-suite runner, conservative CI impact selection, and pull-request workflow discipline.
- Canonical authority: `scripts/local_validation.py`, `scripts/check_suite.py`, `scripts/ci_impact.py`, `.github/workflows/pull-request-validation.yml`, `docs/development/validation.md`.
- Completed so far: PR 3 slices for evidence reuse, check suite + CI impact tooling, and workflow regroup are committed on branch `chore/validation-ci-discipline`.
- Not started yet: observe a real pull-request run with the new summary verification; decide whether `ci_impact` selection should later gate workflow jobs.

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

- Branch `chore/validation-ci-discipline` carries the PR 3 implementation commits; the lead integrates.
- Next step: after integration and push, observe one pull-request run that exercises the extended `Required checks passed` verification and record the result in the trace.
