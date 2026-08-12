# Migration Governance Trace

## 2026-07-23 Intake

- Classified as long-running `docs/automation with server impact` work.
- Confirmed no active topic owns SQL Schema Governance.
- Established `migration-governance` as the active recovery topic.
- Locked Phase 1 to catalog-enforced semantics and disposable PostgreSQL bootstrap; destructive changes remain review-only because Atlas lint requires Atlas Pro.
- Selected `graft-multi-agent-batch` because CLI, catalog-checker, and automation scopes can remain reviewable and non-overlapping.

## 2026-07-23 Phase 1 implementation

- Added registry-derived `graft migrate export` for Atlas CLI tooling without a second migration discovery path.
- Added `graft migrate check-schema`; it evaluates PostgreSQL catalog state and keeps naming/FK source-index findings report-only.
- Replaced the bootstrap script's task-specific assertion with the generic catalog contract check.
- Added a reusable migration CI workflow; the PR migration job now calls that workflow.
- Static tests passed in the numbered worktree. Disposable runtime validation remains intentionally deferred because agent worktrees may not start database containers.

## 2026-08-01 Pipeline governance linkage

- GitHub Actions history identified one transient Docker Hub reset while pulling the disposable PostgreSQL image.
- `pipeline-governance` owns the PR reliability repair and adds bounded image-pull retry in `scripts/check_migration_bootstrap.py`; this topic remains the authority for bootstrap semantics and reusable migration workflow behavior.

## 2026-08-01 PR review documentation alignment

- Aligned the Pipeline Governance recovery validation commands with its completed local-validation record: one `server` directory transition now runs the focused realtime test and both backend validation stages, and the startup prompt lists the complete target set.

## 2026-08-12 Migration lessons and semantic gate

- Moved migration-specific experience into `ai-plan/lessons/migrations.md` and introduced active `MIG-001` for existing-data uniqueness on `registry_connections` with `artifact_repositories` references.
- Established one `*.preflight.yaml` sidecar contract. The static gate verifies its path/version, receipt revisions, active lesson IDs, safety evidence, and SQL-derived minimum risk without taking ownership of PostgreSQL scenario execution.
- Added `graft migrate preflight --manifest` as a deployment-time read-only data check. It cannot apply migrations or alter Atlas revision state; the operator guide places it before the existing `graft migrate up` path.
