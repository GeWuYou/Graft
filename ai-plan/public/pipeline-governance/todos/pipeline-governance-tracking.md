# Pipeline Governance Tracking

## Topic

Pipeline Governance

## Scope

Improve pull-request cancellation, timeout containment, known-fixture secret scanning, disposable migration bootstrap reliability, and the verified realtime SSE test race. Preserve `Build Artifacts` and `Web Check` implementations.

## Repository Truth

- `AGENTS.md`
- `.github/workflows/pull-request-validation.yml`
- `.github/workflows/migration-check.yml`
- `scripts/check_migration_bootstrap.py`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Pull-request workflow orchestration owns CI run coordination; the reusable migration workflow and bootstrap helper own disposable PostgreSQL gate reliability.
requires:
  design: false
  topic: true
  roadmap: true
  adr: false
execution:
  engine: graft-multi-agent-batch
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/public/pipeline-governance/README.md
    - ai-plan/public/pipeline-governance/startup-prompt.md
    - ai-plan/public/pipeline-governance/todos/pipeline-governance-tracking.md
    - ai-plan/public/pipeline-governance/traces/pipeline-governance-trace.md
    - ai-plan/public/pipeline-governance/roadmap/pipeline-governance.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current batch: GitHub Actions observation and settings review.
- Completed: GitHub Actions history analysis; scope and settings decisions; topic bootstrap; first reliability implementation; and local validation.
- Pending: branch push by a maintainer and GitHub Actions observation.
- Risk: GitHub ruleset changes require a repository administrator and are intentionally checklist-only.

## Task Checklist

- [x] Add PR-only stale-run cancellation and bounded non-web job timeouts.
- [x] Centralize known fake secret-fixture exclusions.
- [x] Retry transient PostgreSQL image pulls before disposable bootstrap.
- [x] Synchronize the SSE test with subscription registration.
- [x] Complete local validation.
- [ ] Observe a live pull-request run after push.
- [ ] Review the repository settings checklist with a maintainer.

## Acceptance Conditions

- A new PR commit cancels only prior runs for the same PR; main push validation is never canceled by this policy.
- Expected quality gates remain blocking, including `Server Build/Test Stage`.
- Known fake update diagnostics fixtures do not produce TruffleHog findings, while new paths remain scanned.
- Migration bootstrap retries an image-pull failure with bounded delay and still emits final diagnostics.
- SSE test delivery waits for an actual subscriber and remains stable across repeated execution.
- GitHub settings are documented but not mutated by this change.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "first-reliability-implementation"
  ],
  "pending_batches": [
    "github-actions-observation-and-settings-review"
  ],
  "current_batch": "github-actions-observation-and-settings-review",
  "next_batch": "github-actions-observation-and-settings-review",
  "closeout_status": "awaiting-external-observation"
}
```
