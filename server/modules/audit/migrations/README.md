This directory is the module-owned migration boundary for `server/modules/audit`.

`audit_logs` and `audit_policy_rules` rebuild from this directory as a clean empty-database
baseline. The old shared Ent/manual replay chain has been removed and is no longer a fallback
authority.

`202605190003_audit_module_schema.sql` is the canonical audit-module baseline on the default
migration path for the original audit table structure, indexes, comments, and baseline policy
seed. Follow-up policy upgrades for already deployed environments also live in this directory.

`202606250001_audit_container_dangerous_action_policies.sql` extends the default seeded
`DOMAIN_EVENT` include rules so dangerous container operations are persisted by the audit module
without requiring a manual policy repair after upgrade.
