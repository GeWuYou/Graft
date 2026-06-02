This directory is the module-owned migration boundary for `server/modules/audit`.

`audit_logs` and `audit_policy_rules` rebuild from this directory as a clean empty-database
baseline. The historical shared chain under `server/internal/ent/migrate/migrations` remains
available only for explicit/manual runs.

`202605190003_audit_module_schema.sql` is the canonical audit-module baseline on the default
migration path. It already contains the current table structure, indexes, comments, and seeded
policy rules, so no plugin-boundary or follow-up upgrade/comment migrations remain in this
directory.
