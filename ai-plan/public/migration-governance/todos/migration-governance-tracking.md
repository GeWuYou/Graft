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

## Checklist

- [x] Keep the reusable migration workflow aligned with the canonical SQL policy validator.
- [x] Pass the disposable schema report path through bootstrap and upload the same path as an artifact.
- [x] Preserve the Compose Smoke probe URL in the job summary without shell command substitution.
- [x] Keep the topic README Owned Scope aligned with the migration contract checker package.
- [x] Record this review batch without changing the existing Work Contract, Recovery Point, or Batch State.
- [x] Link the transient PostgreSQL image-pull retry to `pipeline-governance` while retaining this topic's bootstrap authority.

## Acceptance Conditions

- `migration-check.yml` runs `python3 scripts/validate_sql_migrations.py` before migration bootstrap validation.
- The bootstrap step writes `MIGRATION_SCHEMA_REPORT`, and the diagnostics artifact uploads that exact path.
- Compose Smoke summary URLs render as Markdown code spans and retain the resolved probe base URL.
- `README.md` lists `server/internal/migrationcontract/**` in Owned Scope.
- This tracking file contains an auditable checklist and acceptance conditions, while PR metadata remains unchanged.
