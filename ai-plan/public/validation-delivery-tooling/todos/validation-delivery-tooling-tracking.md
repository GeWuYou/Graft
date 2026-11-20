# Validation Delivery Tooling Tracking

## Topic

Validation Delivery Tooling

## Scope

SHA-bound validation evidence reuse, grouped suites with a deadline, conservative CI impact mapping, a verified pull-request summary gate, the browser acceptance harness, and the same discipline for preview lifecycle, deployment consistency and release candidates.

## Repository Truth

- `AGENTS.md`; `docs/development/validation.md`, `preview-lifecycle.md`, `browser-acceptance.md`; `docs/operations/release-deployment-consistency.md`
- `scripts/`: `local_validation.py`, `check_suite.py`, `ci_impact.py`, `preview.py`, `preview_recovery.py`, `check_deployment_consistency.py`, `browser_acceptance.py`
- `tests/browser/**`; test-only Go seeds in `server/modules/*` `browser_seed_test.go`
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

- Current batch: PR 3 (evidence reuse + suite + impact + workflow), PR 4 (browser acceptance harness) and PR 5 (preview lifecycle + release consistency) delivered into main; per-PR corrections recorded in the trace with thread dispositions (full detail in git history).
- Pending: observe a real pull-request run and one release-candidate dispatch; primary-checkout follow-ups (preview smoke, deployment consistency, chromium local-tier browser run); CI impact gating decision.
- Risk: `kind: optimize` extends the ADR-003 v1 vocabulary; ADR owner reconciles later.

## Task Checklist

- [x] PR 3: evidence reuse, check-suite groups, ci-impact selection, workflow regroup + summary cross-check, corrections 1-3.
- [x] PR 4: browser acceptance harness, tests/browser fixtures + specs, test-only seeds, docs, corrections 1-3.
- [x] PR 5: preview lifecycle + recovery + deployment consistency + release-candidate workflow, docs, corrections 1-5.
- [ ] Observe a real pull-request run and one release dispatch; record evidence in the trace.
- [ ] Primary-checkout follow-ups: preview smoke, deployment consistency, chromium local-tier browser run.
- [ ] Later: CI impact gating workflow job paths (separate batch).

## Acceptance Conditions

- Scoped validation and comparisons fail closed; suites resolve dependencies, aggregate failures and enforce a deadline; CI impact stays conservative and advisory.
- PR summary verifies every job of run/attempt/SHA except documented skips; check names unchanged.
- Browser acceptance: risk-tiered suites x engines, zero retries, sanitized per-attempt reports, per-attempt session state and daemon page assertions; primary-checkout only.
- Preview/deployment/release semantics per the delivered docs: deterministic identity and guarded destroy with in-container data removal, flock-serialized registry, post-redirect asset resolution, compose child env stripped of reserved keys, idempotent `rc-` tags with unfixed-scan evidence; status stays readable over cleanup-failed leftovers.
- actionlint passes on both workflows; script tests cover changed behavior.
