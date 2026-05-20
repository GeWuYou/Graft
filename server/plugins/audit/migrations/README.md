This directory is the plugin-owned migration boundary for `server/plugins/audit`.

`audit_logs` now rebuilds from this directory as the default owner-aligned baseline. The
historical shared chain under `server/internal/ent/migrate/migrations` remains available only for
explicit/manual runs.

`202605190003_audit_plugin_boundary_baseline.sql` is the first forward-only version in this
directory. It establishes the plugin-owned baseline for the audit table without keeping the
historical shared chain on the default migration path.
