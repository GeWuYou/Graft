# auth migrations

`auth_credentials` and `auth_refresh_sessions` are owned by this directory.
`202607100003_auth_credential_session_schema.sql` creates empty auth-owned
tables only; it deliberately does not copy legacy user-owned credential or
refresh-session data. The current user-backed runtime remains the only live
persistence implementation until the later store-switch and data-migration
batches complete.
