Continue the `validation-delivery-tooling` topic after completing root `AGENTS.md` startup preflight.

- governance source: root `AGENTS.md`
- task class: `docs/automation`
- recovery source: topic `ai-plan/public/validation-delivery-tooling/README.md`
- local execution truth: `ai-plan/AGENTS.md`
- authority: root `AGENTS.md`; `docs/development/validation.md`; `.github/workflows/pull-request-validation.yml`; `Justfile`
- AI skills: `$graft-validation-runner`, `$graft-commit`

Topic objective: keep scoped validation evidence reusable, grouped suites bounded, CI impact selection conservative, and pull-request validation fully verified, without weakening required gates or renaming required checks.

Work contract summary: `optimize`, `long-running`, active topic only; no design, roadmap, or ADR requirement.

Locked decisions:

1. Scoped validation never silently falls back to full gates.
2. Evidence reuse fails closed on head or digest mismatch.
3. CI impact selection stays advisory and conservative.
4. Required check names and the no-push-cancellation policy are preserved.

Current batch plan:

1. If the implementation branch is not yet integrated, verify the recorded commits and continue from the tracking checklist.
2. After the lead integrates and pushes, observe one pull-request run exercising the extended `Required checks passed` verification.

Validation expectations:

```bash
python3 -m unittest scripts/test_local_validation.py scripts/test_check_suite.py scripts/test_ci_impact.py scripts/test_governance_workflow.py
python3 -B scripts/check_suite.py --groups governance --dry-run
python3 -B scripts/ci_impact.py --json
actionlint .github/workflows/pull-request-validation.yml
python3 scripts/validate_ai_plan_structure.py
python3 scripts/validate_ai_governance.py
git diff --check
```
