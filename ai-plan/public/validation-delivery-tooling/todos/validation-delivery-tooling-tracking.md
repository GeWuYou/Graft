# Validation Delivery Tooling Tracking

## Topic

Validation Delivery Tooling

## Scope

SHA-bound validation evidence reuse, grouped suites with a deadline, conservative CI impact mapping, a verified pull-request summary gate, and the same discipline for preview lifecycle, deployment consistency and release candidates.

## Repository Truth

- `AGENTS.md`; `docs/development/validation.md`, `preview-lifecycle.md`; `docs/operations/release-deployment-consistency.md`
- `scripts/`: `local_validation.py`, `check_suite.py`, `ci_impact.py`, `preview.py`, `preview_recovery.py`, `check_deployment_consistency.py`
- `.github/workflows/pull-request-validation.yml`, `release-candidate.yml`; `Justfile`

## Work Contract

```yaml
version: 1
kind: optimize
scope: long-running
authority_summary: Validation tooling scripts own scoped evidence, suite, impact, preview lifecycle, recovery and deployment-consistency semantics; the two workflows own job orchestration, summary gates and tagging discipline.
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

- Current batch: PR 5 plus CI corrections 1-3 committed on `chore/preview-release-consistency`, awaiting lead integration; SHAs and thread dispositions in the trace.
- Completed: evidence reuse, suite + impact, workflow discipline, PR 3 corrections, PR 5 slices, corrections 1-3.
- Pending: observe a real pull-request run and one release-candidate dispatch; primary-checkout smoke and consistency runs.
- Risk: `kind: optimize` extends the ADR-003 v1 vocabulary; ADR owner reconciles later.

## Task Checklist

- [x] Evidence reuse, check suite, CI impact and workflow regroup (PR 3); PR 3 corrections (trace git history).
- [x] PR 5: `preview.py` identity, common-dir registry, port refusal, cap, guarded destroy with evidence archives.
- [x] PR 5: `preview_recovery.py` read-only reconciliation, ownership-proof-gated suggestions, evidence verification.
- [x] PR 5: `check_deployment_consistency.py` no-acceptable-5xx contract, fail-closed revision markers.
- [x] PR 5: release-candidate workflow with artifact binding and idempotent `rc-` tagging.
- [x] PR 5: Justfile preview/release section, preview and release docs.
- [x] Corrections 1-3 on PR #377: 16 threads repaired (ids in the trace).
- [ ] Observe a real pull-request run after integration; record evidence in the trace.
- [ ] Primary-checkout follow-ups: preview smoke, deployment consistency run, one release dispatch.
- [ ] Later: whether CI impact gates workflow job paths (separate batch).

## Acceptance Conditions

- Scoped validation and comparisons fail closed; suites resolve dependencies, aggregate failures and enforce a deadline; CI impact stays conservative and advisory.
- PR summary verifies every job of run/attempt/SHA except documented skips; check names unchanged.
- Preview: deterministic identity, common-dir registry, port refusal, cap enforcement including restarts, flock-serialized registry writes, instance-scoped backups, restore of unhealthy running instances, recorded-project teardown and retryable destroy on cleanup failure.
- Deployment check resolves assets against the post-redirect final URL; fails on asset 404, wrong types, 5xx without `--acceptable-status`, and undeclared revision expectations; status-accepted assets are flagged, not content-verified.
- Release-candidate summary verifies run/attempt/SHA/version/channel/digests; `rc-` tags idempotent on same SHA, hard-fail on different SHA; registration requires unfixed-scan evidence.
- actionlint passes on both workflows; script tests cover changed behavior.
