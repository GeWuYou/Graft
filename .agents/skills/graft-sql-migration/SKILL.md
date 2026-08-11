---
name: graft-sql-migration
description: Repository-specific workflow for creating or modifying Graft live Atlas/PostgreSQL migration SQL with Chinese database comments, globally unique versions, live/legacy boundaries, and deterministic validation.
---

# Graft SQL Migration

Use this skill whenever a task creates or modifies migration SQL in `Graft`.

This skill owns SQL comment and migration validation workflow details. It does not replace root `AGENTS.md`,
`server/AGENTS.md`, `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`, `graft-table-design`, Ent generation, Atlas migration
generation, or backend validation.

## Trigger

Use this skill when the task touches any of:

- New database tables.
- Table structure changes.
- Added, deleted, or renamed columns.
- Atlas / PostgreSQL migration SQL.
- New module migration directories.
- Indexes, unique constraints, or foreign key constraints.
- Soft-delete fields.
- Database table or column comment fixes.

## Required Reading

1. Complete the startup preflight from root `AGENTS.md`.
2. Read `server/AGENTS.md`.
3. Read `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`.
4. Search `ai-plan/lessons/migrations.md` by operation, risk, and affected tables, then report the matched `MIG-###` IDs before migration design. No match is a valid result for unrelated L0 work.
5. Read the current module's related migration files.
5. If the task also requires table design decisions, use `graft-table-design` first.

## SQL Comment Hard Rules

- `CREATE TABLE` must have Chinese `COMMENT ON TABLE` in the same migration file.
- Every `CREATE TABLE` column must have Chinese `COMMENT ON COLUMN` in the same migration file.
- Every `ALTER TABLE ... ADD COLUMN` column must have Chinese `COMMENT ON COLUMN` in the same migration file.
- Common fields such as `id`, `created_at`, `updated_at`, and `deleted_at` are not exempt.
- `JSON` / `JSONB`, enum, status, boolean, and foreign key fields must explain business semantics.
- Comments must not use English, pinyin, `TODO`, `TBD`, `placeholder`, `待补充`, or `临时说明`.
- Comments must not merely restate the table or column identifier.
- Design docs do not replace database comments; the migration SQL must carry the comments.

## Version And Scope Rules

- Live migration versions must be globally unique across the full default live migration chain.
- Before adding a migration, search existing live migration versions.
- Default live migrations are discovered through core live dirs plus module registry migration dirs.
- The old `server/internal/ent/migrate/migrations/**` replay chain has been removed and is no longer a fallback
  authority.
- Do not recreate legacy migration files just to pass the current live SQL gate.
- If adding a live migration directory, update `scripts/validate_sql_migrations.py` discovery when registry/core-dir
  discovery would not include it.
- Every newly added or modified live migration SQL has exactly one sibling `<migration>.preflight.yaml`. It records the migration path/version, owner, derived risk level, affected and referenced tables, operation categories, historical assumptions, upgrade order, safety strategy, validation scenarios, and retrieval receipt.
- The receipt records the canonical governance and migration-lessons paths, their tooling-generated SHA-256 content revisions, and the actual matching `MIG-###` IDs. Do not use a self-attestation field such as `lessons_read: true`.
- The static gate derives a minimum risk: L0 additive schema, L1 metadata/backfill-safe, L2 constraint/index, L3 data transformation, L4 deletion/reconciliation, L5 irreversible or production-sensitive. A sidecar cannot understate it.
- Historical SQL is forward-only. A modified historical migration is rejected unless the sidecar carries governed unpublished-exception evidence; normal repair uses a higher version.

## Validation

After modifying migration SQL, run:

```bash
python3 scripts/validate_sql_migrations.py --changed --base-ref "$(git merge-base origin/main HEAD)"
```

For a deliberately scoped migration file, use `--paths <file>` instead. The validator rejects an unscoped invocation so a changed live migration cannot bypass the sidecar requirement.

If the migration directory contents changed, also recompute the touched directory checksum immediately:

```bash
cd server/modules/<module>/migrations && atlas migrate hash --dir file://.
```

If that migration directory is embedded through `server/internal/moduleregistry/generated.go`, regenerate it in the
same slice:

```bash
cd server && go generate ./internal/moduleregistry
```

Treat `validate migration dir ...: checksum mismatch` as a task failure, not as a deferred follow-up.

Also run the appropriate direct validation for the changed slice, such as:

```bash
python3 scripts/check_migration_versions.py --mode all
python3 scripts/validate_ai_governance.py
python3 -m unittest discover -s scripts -p 'test_*.py'
```

Do not claim completion while `scripts/validate_sql_migrations.py` fails. If a stronger backend validation is expected
but cannot run, report the exact command and blocker.

## Closeout Output

Final closeout for migration SQL work must include:

- Startup receipt.
- Modified files.
- Added or changed tables.
- Added or changed columns.
- `COMMENT ON TABLE` / `COMMENT ON COLUMN` coverage.
- Validation commands and result summary.
- Remaining uncovered boundary, if any.
- Whether legacy migration files were touched; if yes, why.
