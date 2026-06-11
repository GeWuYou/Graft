---
name: graft-table-design
description: Repository-specific workflow for Graft database table design and migration governance. Use when designing or changing DB tables, Ent schemas, Atlas migrations, audit fields, soft delete fields, deleted_by, indexes, partial unique indexes, store query semantics, or database table and column comments.
---

# Graft Table Design

Use this skill before designing, changing, or reviewing database table structures in `Graft`.

This skill is a governance workflow. It does not replace root `AGENTS.md`, `server/AGENTS.md`, backend validation, Ent
generation, Atlas migration generation, or module ownership rules.

`graft-table-design` owns modeling decisions: table owner, fields, audit fields, soft delete, store query semantics,
index plan, and compatibility notes. When actually creating or modifying migration SQL, also use
`graft-sql-migration`; SQL comment hard rules and migration SQL validation are owned there and must not be duplicated
with conflicting wording in this skill.

## Workflow

1. Complete the startup preflight from root `AGENTS.md`.
2. Classify the task:
   - `server` when changing Ent schema, migrations, store query code, or backend tests.
   - `docs/automation with server impact` when only changing governance docs or skills.
   - `cross-boundary` when table changes alter shared contracts, OpenAPI, web bootstrap semantics, or frontend-visible behavior.
3. Read:
   - `server/AGENTS.md`
   - `ai-plan/design/数据库表设计与迁移规范.md`
   - `ai-plan/design/契约治理与魔法值治理规范.md` when table semantics affect shared contracts, status values, permission codes, or API-visible fields
4. Identify the owner before proposing fields:
   - owner module or core owner
   - table name
   - lifecycle owner for writes, deletes, and store queries
   - whether any cross-module capability / stable DTO is required
5. Design the table against the required governance:
   - audit fields, including `created_at`, `updated_at`, and applicable `created_by`, `updated_by`, `deleted_by`
   - new-table soft delete as `deleted_at BIGINT NOT NULL DEFAULT 0`
   - live query semantics as `deleted_at = 0`
   - Chinese Ent schema comments and migration SQL comments
   - migration version prefix globally unique across all live module migration directories
   - indexes from real store query semantics
   - partial unique indexes using `WHERE deleted_at = 0` for soft-delete live-row uniqueness
6. Treat nullable timestamp soft delete as existing-table compatibility only. Before adding compatibility, record the
   canonical owner, why direct repair is not done now, affected consumers, cleanup trigger, and validation scope.
7. Validate according to the task class:
   - schema / migration changes: follow `server/AGENTS.md` Ent generate, migration, test, and backend validation rules.
   - migration SQL creation or edits: also follow `graft-sql-migration` and run `python3 scripts/validate_sql_migrations.py`.
   - governance-only changes: run the relevant docs / skill validation and explain why runtime validation is not applicable.

## Required Output

Report this before implementation or in closeout:

```text
Table design summary:
- owner_module:
- table_name:
- fields:
- audit_fields:
- soft_delete_model:
- store_query_semantics:
- index_plan:
- comments_plan:
- migration_version:
- validation:
- compatibility_notes:
```

## Guardrails

- Do not add new business schema truth under `server/internal/ent/**`.
- Do not let one module migration modify another module's table.
- Do not use nullable timestamp soft delete for new tables.
- Do not omit Chinese table or column comments from Ent schema or migration SQL.
- Do not claim a schema change is complete without matching Ent generation, migration, and direct validation evidence.
