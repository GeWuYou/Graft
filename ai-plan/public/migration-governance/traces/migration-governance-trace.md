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
