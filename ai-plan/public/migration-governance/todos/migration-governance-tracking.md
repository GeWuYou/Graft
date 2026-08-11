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

- Current batch: migration lessons and semantic sidecar gate.
- Completed: repository audit, strategy decisions, topic bootstrap, catalog checker, bootstrap integration, local/CI wiring, integration-checkout runtime validation, and the migration lessons/sidecar rollout.
- Pending: PR CI execution and disposable PostgreSQL historical scenario expansion for MIG-001.
- Risk: static risk derivation is a conservative lower bound; historical data safety still requires the PostgreSQL scenario and deployment preflight evidence.

## Batch State

- completed_batches:
  - catalog-contract-and-bootstrap
  - local-and-ci-wiring
  - migration-lessons-and-semantic-gate
- pending_batches:
  - integration-runtime-validation
- next_batch: migration-historical-scenarios

## Checklist

- [x] Keep the reusable migration workflow aligned with the canonical SQL policy validator.
- [x] Pass the disposable schema report path through bootstrap and upload the same path as an artifact.
- [x] Preserve the Compose Smoke probe URL in the job summary without shell command substitution.
- [x] Keep the topic README Owned Scope aligned with the migration contract checker package.
- [x] Record this review batch without changing the existing Work Contract, Recovery Point, or Batch State.
- [x] Link the transient PostgreSQL image-pull retry to `pipeline-governance` while retaining this topic's bootstrap authority.
- [x] Add `MIG-001`, a migration-specific lessons file, and deterministic SHA-256 retrieval receipts in the sole YAML sidecar contract.
- [x] Add static sidecar/risk/receipt validation and the read-only `graft migrate preflight` operator command.

## Acceptance Conditions

- `migration-check.yml` runs `python3 scripts/validate_sql_migrations.py` before migration bootstrap validation.
- The bootstrap step writes `MIGRATION_SCHEMA_REPORT`, and the diagnostics artifact uploads that exact path.
- Compose Smoke summary URLs render as Markdown code spans and retain the resolved probe base URL.
- `README.md` lists `server/internal/migrationcontract/**` in Owned Scope.
- This tracking file contains an auditable checklist and acceptance conditions, while PR metadata remains unchanged.
- New or modified live migration SQL is checked for exactly one `.preflight.yaml`; stale governance/lesson revisions, inactive lesson IDs, missing safety evidence, risk understatement, and ungoverned historical rewrites fail the static gate.
- `graft migrate preflight --manifest` only runs declared read-only target-data checks and never invokes Atlas migration execution.
