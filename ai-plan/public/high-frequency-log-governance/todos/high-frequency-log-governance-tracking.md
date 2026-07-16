# High Frequency Log Governance Tracking

## Work Contract

```yaml
version: v1
kind: refactor
scope: long-running
authority_summary: server/internal/logger owns runtime logging; server/internal/config owns startup configuration; App Log, OpenAPI, and web are downstream consumers.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - ai-plan/public/high-frequency-log-governance/README.md
    - ai-plan/public/high-frequency-log-governance/startup-prompt.md
    - ai-plan/public/high-frequency-log-governance/todos/high-frequency-log-governance-tracking.md
    - ai-plan/public/high-frequency-log-governance/traces/high-frequency-log-governance-trace.md
closeout:
  archive: true
  lessons_review: true
```

## Loop State

- loop_mode: `topic-completion-loop`
- completed_batches:
  - `batch-1-logger-category-foundation`
- current_batch: `batch-2-high-frequency-migration-and-static-governance`
- pending_batches:
  - `batch-2-high-frequency-migration-and-static-governance`
  - `batch-3-app-log-category-contract`
- next_batch: `batch-2-high-frequency-migration-and-static-governance`

## Acceptance Conditions

- TRACE is available without changing existing `*zap.Logger` injection.
- Categories are typed constants in a logger-owned registry and config accepts no business-code string literals.
- Disabled Category prevents lazy field creation, encoding, serialization, and durable persistence.
- High-frequency normal diagnostics use TRACE; periodic failures remain visible and bounded.
- The final contract, persistence, and web impact are validated only when Batch 3 determines they are required.

## Current Risks

- TRACE requires a custom zap level and encoder handling; compatibility must be covered by observer tests.
- App Log persistence currently has no category field and is intentionally not modified before its authority batch.
- Category literal static checking must remain bounded to production server code and must not become a whole-repository linter.
