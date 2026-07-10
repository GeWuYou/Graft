# auth migrations

`auth_credentials` and `auth_refresh_sessions` are owned by this directory.
`202607100003_auth_credential_session_schema.sql` creates the auth-owned
tables. `202607100004_copy_legacy_auth_data.sql` then copies the existing
user-owned credential and refresh-session state into those tables, including
legacy IDs, user IDs, token IDs, password-change state, timestamps, revocation,
and replacement data. It also advances the auth table identity sequences.

The current user-backed runtime remains the only live persistence
implementation until the later one-way store-switch batch; this migration does
not add fallback reads or dual writes.
