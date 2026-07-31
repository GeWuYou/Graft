# Pipeline Governance

## Current Status Summary

- Topic objective: make pull-request validation predictable, bounded, and evidence-driven without weakening required quality gates.
- Current status: `active`
- Task class: `docs/automation with server impact`
- Intake summary: long-running refactor with an active topic, roadmap, and batch-oriented execution.
- Canonical authority: `.github/workflows/pull-request-validation.yml`, `.github/workflows/migration-check.yml`, and `scripts/check_migration_bootstrap.py`.
- Completed so far: historical Actions failure analysis, first reliability batch, and local validation.
- Not started yet: observe the changed workflow on GitHub and review the repository ruleset checklist with a maintainer.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `docs/automation`
- recovery source: parent topic `migration-governance` for disposable PostgreSQL bootstrap semantics
- authority summary: PR workflow orchestration owns cancellation and job limits; reusable migration workflow and its bootstrap helper own migration-gate reliability.

## Owned Scope

- `.github/workflows/pull-request-validation.yml`
- `.trufflehog-exclude-paths.txt`
- `scripts/check_migration_bootstrap.py`
- `scripts/test_check_migration_bootstrap.py`
- `server/internal/realtime/ws_handler_test.go`
- `ai-plan/public/pipeline-governance/**`

Out of scope:

- `Build Artifacts` implementation changes.
- `Web Check` implementation changes.
- GitHub repository or ruleset API writes.

## Locked Decisions

1. Required quality gates remain blocking; report-only jobs stay non-required in repository settings.
2. Cancellation applies only to obsolete pull-request runs, never main-branch push validation.
3. TruffleHog exclusions are limited to known fake fixtures and live in one repository file.

## Phase Plan

- Batch 1: stabilize PR workflow orchestration, migration bootstrap image retrieval, and SSE test delivery ordering.
- Batch 2: verify real GitHub Actions behavior and have a maintainer apply the documented repository-settings checklist.

## Current Recovery Point

- Batch 1 is committed after local validation.
- GitHub Actions history showed fake-fixture secret-scan findings, one Docker Hub pull reset, and an SSE delivery race.
- Next step: observe a PR run after the branch is pushed by a maintainer.

## Work Intake

- This topic was created through `Work Intake`.
- The full Work Contract is in `todos/pipeline-governance-tracking.md`.

## Pending Batch Direction

- Verify cancellation, timeout, secret scanning, and migration bootstrap behavior in GitHub Actions.
- Review and apply repository settings manually; this topic only records the checklist.

## Validation Targets

```bash
actionlint .github/workflows/*.yml
python3 -m unittest scripts/test_check_migration_bootstrap.py
cd server && go test ./internal/realtime -count=20
cd server && go run ./cmd/graft validate backend --stage lint
cd server && go run ./cmd/graft validate backend --stage buildtest
python3 scripts/validate_ai_plan_structure.py
python3 scripts/validate_ai_governance.py
git diff --check
```

## Loop Entry

- Preferred entry: `ai-plan/public/pipeline-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-batch`
