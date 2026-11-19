# Validation Delivery Tooling Tracking

## Topic

Validation Delivery Tooling

## Scope

Scoped validation evidence reuse (SHA-bound), grouped suites with a deadline, conservative CI impact mapping, a fully verified pull-request summary gate, and the same discipline for preview lifecycle, deployment consistency and release candidates.

## Repository Truth

- `AGENTS.md`; `docs/development/validation.md`, `preview-lifecycle.md`; `docs/operations/release-deployment-consistency.md`
- `scripts/`: `local_validation.py`, `check_suite.py`, `ci_impact.py`, `preview.py`, `preview_recovery.py`, `check_deployment_consistency.py`
- `.github/workflows/pull-request-validation.yml`, `release-candidate.yml`; `Justfile`

## Work Contract

```yaml
version: 1
kind: optimize
scope: long-running
authority_summary: Validation tooling scripts own scoped evidence, suite, impact, preview lifecycle, recovery and deployment-consistency semantics; the pull-request and release-candidate workflows own their job orchestration, summary gates and tagging discipline.
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

- Current batch: PR 5 (preview lifecycle + deployment consistency + release candidate flow) committed on `chore/preview-release-consistency`, awaiting lead integration. Commits: 25613647d, 69a9123d0, 30a0617d7, 3d8a622ab, 68dcd57c7.
- Completed: evidence reuse, suite + impact, workflow discipline, corrections 1-3, PR 5 slices.
- Pending: observe a real pull-request run and one release-candidate dispatch after integration; primary-checkout preview smoke and deployment consistency runs (commands in the docs).
- Risk: `kind: optimize` extends the ADR-003 v1 kind vocabulary; ADR owner reconciles later.

## Task Checklist

- [x] Evidence reuse, check suite, CI impact and workflow regroup (PR 3); corrections 1-3 (trace git history).
- [x] PR 5: `preview.py` identity, common-dir registry, port refusal, `PREVIEW_MAX_RUNNING`, guarded destroy with SHA-256 evidence archives.
- [x] PR 5: `preview_recovery.py` read-only reconciliation with ownership-proof-gated suggestions and evidence verification.
- [x] PR 5: `check_deployment_consistency.py` no-acceptable-5xx contract, fail-closed revision markers.
- [x] PR 5: release-candidate workflow with artifact binding and idempotent `rc-` tagging.
- [x] PR 5: Justfile preview/release section, preview and release docs.
- [ ] Observe a real pull-request run after integration; record evidence in the trace.
- [ ] Primary-checkout follow-ups: preview smoke, deployment consistency run, one release dispatch.
- [ ] Later: whether CI impact gates workflow job paths (separate batch).

## Acceptance Conditions

- Scoped validation and comparisons fail closed; check suite resolves dependencies, aggregates failures and enforces a process-level deadline; CI impact stays conservative and advisory.
- PR summary verifies every job of the current run/attempt/SHA except documented skips; required check names unchanged.
- Preview: deterministic identity, common-dir registry, port refusal, cap enforcement, guarded destroy with evidence archives, read-only recovery suggesting destroy only with proven ownership.
- Deployment check fails on asset 404, wrong content type, 5xx without `--acceptable-status`, and revision expectations without declared markers.
- Release-candidate summary verifies run/attempt/SHA/version/channel/digests; `rc-` tags idempotent on same SHA, hard-fail on a different SHA; `v*` stays semantic-release-owned.
- actionlint passes on both workflows; script unit tests cover the changed behavior.
