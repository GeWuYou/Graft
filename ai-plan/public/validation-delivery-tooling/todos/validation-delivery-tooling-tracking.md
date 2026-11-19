# Validation Delivery Tooling Tracking

## Topic

Validation Delivery Tooling

## Scope

Make scoped local validation evidence reusable and SHA-bound, run grouped check suites with dependencies and a deadline, map changed paths to CI jobs conservatively, and discipline the pull-request validation workflow with grouping, timeouts, and a complete summary gate. Preserve required check names and existing blocking gates.

## Repository Truth

- `AGENTS.md`
- `docs/development/validation.md`
- `scripts/local_validation.py`
- `scripts/check_suite.py`
- `scripts/ci_impact.py`
- `.github/workflows/pull-request-validation.yml`
- `Justfile`

## Work Contract

```yaml
version: 1
kind: optimize
scope: long-running
authority_summary: Scoped validation tooling owns evidence reuse semantics; the grouped suite runner and CI impact mapper own their selection rules; the pull-request workflow owns job orchestration and the run summary gate.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-loop
bootstrap:
  targets:
    - ai-plan/public/validation-delivery-tooling/README.md
    - ai-plan/public/validation-delivery-tooling/startup-prompt.md
    - ai-plan/public/validation-delivery-tooling/todos/validation-delivery-tooling-tracking.md
    - ai-plan/public/validation-delivery-tooling/traces/validation-delivery-tooling-trace.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current batch: PR 3 implementation on branch `chore/validation-ci-discipline`, committed and awaiting lead integration.
- Completed: evidence reuse slice, check suite + CI impact slice, workflow discipline slice, documentation update.
- Pending: observe one real pull-request run with the extended summary verification after integration and push.
- Risk: `kind: optimize` extends the ADR-003 v1 documented kind vocabulary; the ADR owner should reconcile the vocabulary in a later governance batch.

## Task Checklist

- [x] Bind scoped validation plans to head SHA and selected file digests with per-file reasons and coverage.
- [x] Add `--compare-report` evidence verification and `--report-out` executed check reports; plan mode fails on gaps.
- [x] Add `scripts/check_suite.py` grouped runner with requires, deadline, JSON summary and artifact logs.
- [x] Add `scripts/ci_impact.py` conservative path-to-job mapping with FULL-forcing shared surfaces.
- [x] Regroup pull-request-validation jobs, add missing timeout, extend summary needs to every job, add the run/attempt/SHA API cross-check.
- [x] Update `docs/development/validation.md` for evidence reuse, suite, impact and summary semantics.
- [ ] Observe a real pull-request run after integration and record the evidence in the trace.
- [ ] Decide whether CI impact selection should later gate workflow job paths (separate batch, requires roadmap consideration).

## Acceptance Conditions

- `just check-local-plan` exits nonzero on empty selection, missing consumer tests, or coverage gaps, and never falls back to full gates.
- `--compare-report` fails with per-file reasons when the recorded head moved, selected files changed, or the current selection is uncovered.
- `scripts/check_suite.py` resolves groups and dependencies with cycle and unknown-name rejection, aggregates independent failures, kills timed-out checks at the process level, and exits nonzero on any failure or deadline overrun.
- `scripts/ci_impact.py` forces FULL for unknown paths and shared surfaces and always includes hygiene and summary jobs in subsets.
- The pull-request summary fails on any job that is not successful for the current run, attempt and SHA, except the documented push-event Secret Scan skip; required check names are unchanged.
- `actionlint` passes on the modified workflow and unit tests cover the changed script behavior.
