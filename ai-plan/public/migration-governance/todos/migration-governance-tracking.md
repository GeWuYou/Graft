# Migration Governance Tracking

## Work Contract

```yaml
version: v1
kind: feature
scope: long-running
authority_summary: server/internal/moduleregistry owns default-chain discovery; Atlas and PostgreSQL catalog checks consume that authority.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-batch
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/public/migration-governance/README.md
    - ai-plan/public/migration-governance/startup-prompt.md
    - ai-plan/public/migration-governance/todos/migration-governance-tracking.md
    - ai-plan/public/migration-governance/traces/migration-governance-trace.md
closeout:
  archive: true
  lessons_review: true
```

## Recovery Point

- Current batch: Phase 1 implementation.
- Completed: repository audit, strategy decisions, topic bootstrap, catalog checker, bootstrap integration, local/CI wiring, and integration-checkout runtime validation.
- Pending: PR CI execution.
- Risk: destructive migration detection remains review-only until a PostgreSQL-aware automated analyzer is available.

## Batch State

- completed_batches:
  - catalog-contract-and-bootstrap
  - local-and-ci-wiring
- pending_batches:
  - integration-runtime-validation
- next_batch: integration-runtime-validation
