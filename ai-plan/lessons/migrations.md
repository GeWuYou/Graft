# Migration Lessons

Migration lessons record durable historical-data and rollout experience. Each entry uses `MIG-###` and includes status, source, risk level, problem, why it happened, bad pattern, required pattern, detection rule, agent rule, enforcement, and related migrations or tables.

## MIG-001: Existing-data uniqueness must reconcile or abort before the index

- Status: active
- Source:
  - `server/modules/registry/migrations/202608110003_registry_connection_management.sql`
- Risk level: L4
- Problem:
  Adding a live uniqueness rule to `registry_connections(provider, endpoint)` can fail when historical duplicates exist, or silently orphan active `artifact_repositories` when duplicate connections are retired without checking references.
- Why it happened:
  Schema-level uniqueness describes the target invariant but does not establish that existing rows satisfy it or that dependent facts can be safely reconciled.
- Bad pattern:
  Create the unique index first, delete or soft-delete duplicate rows without inspecting live references, or allow the migration to choose a survivor while active artifact repositories still point to a duplicate.
- Required pattern:
  Inspect duplicate groups and active references first. Abort explicitly when a duplicate has a live artifact-repository reference; reconcile only unreferenced duplicates; create the unique index after reconciliation; then verify the live uniqueness invariant.
- Detection rule:
  An L2-or-higher uniqueness sidecar must declare duplicate and live-reference scans, reconcile-or-abort behavior, and the post-migration invariant. Static validation rejects missing evidence and risk understatement.
- Agent rule:
  Before authoring a live migration, retrieve this file by operation, risk, and affected concepts. Record matching `MIG-###` IDs and immutable content revisions in the single `.preflight.yaml` sidecar; never replace retrieval evidence with a self-attestation flag.
- Enforcement:
  `scripts/validate_sql_migrations.py` validates the sidecar receipt and safety evidence. Historical PostgreSQL scenarios remain in the disposable migration bootstrap path; deployment preflight runs only read-only duplicate and reference checks before `graft migrate up`.
- Related migrations/tables:
  - `server/modules/registry/migrations/202608110003_registry_connection_management.sql`
  - `registry_connections`
  - `artifact_repositories`

## MIG-002: Executed Atlas versions must receive forward-only repairs

- Status: active
- Historical stable ID: `LESSON-BACKEND-MIGRATION-VERSION-001` (moved from `backend.md`; preserve this cross-reference for older traces and links).
- Source:
  - Scheduler repair after `202606050001_scheduler_task_runs.sql` had already reached Atlas revision history.
  - The later access-log WebSocket correction that replaced an old-file rewrite with a higher-version migration.
- Risk level: L3
- Problem:
  Appending DDL to an already executed version makes Atlas report no pending work while deployed schemas lack the later statements.
- Why it happened:
  A local unexecuted database was treated as proof that a shared version could be replayed everywhere.
- Bad pattern:
  Rewrite a shared versioned SQL file, update `atlas.sum`, and expect existing databases to rerun it.
- Required pattern:
  Treat committed, shared, or CI-visible versions as executed unless governed evidence proves every affected environment is unpublished and unexecuted; otherwise create a higher globally unique migration.
- Detection rule:
  The static migration gate checks changed live SQL and rejects historical modification unless its sole sidecar carries explicit unpublished exception reason and evidence.
- Agent rule:
  Retrieve this lesson before changing a live SQL file. Do not use sidecar metadata to turn a normal historical repair into a compatibility path.
- Enforcement:
  `scripts/validate_sql_migrations.py --changed` and the staged hook enforce the forward-only default; Atlas checksum validation remains required after each directory change.
- Related migrations/tables:
  - `server/modules/scheduler/migrations/202606050002_scheduler_scheduled_tasks.sql`
  - `server/internal/httpx/migrations/202607150002_access_log_connection_type_backfill_fix.sql`
