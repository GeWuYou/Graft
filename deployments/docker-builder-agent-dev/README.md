# Docker Builder Agent Local Dependencies

This development-only topology starts isolated Redis and Vault for a
host-debugged Docker Builder Agent. It supports two explicit Server database
modes and uses only the ignored `.data/docker-builder-agent-dev` directory for
Agent-local state:

- `shared` is the default. The local Server uses the development database
  configured in `server/.env`.
- `isolated` starts the Compose PostgreSQL service and gives the local Server a
  disposable database at `127.0.0.1:15432`, stored under `.data`.

`graft dev docker-builder-agent prepare` defaults to `shared`: it reads
`GRAFT_DATABASE_URL` from `server/.env`, supplies the remaining generated
environment values, and starts `vault-init`. Use
`graft dev docker-builder-agent prepare --database-mode isolated` for a
disposable database. Use the same `--database-mode` for `deliver` and `reset`.
Do not point `server/.env` at production credentials or a production database
for this topology.

`vault-init` creates disposable Vault PKI/AppRole material, the local Server
TLS certificate and key, and the enrollment pepper below `.data`:

- `server/credential-vault/`: Backend-only Vault CA, AppRole files, and pepper.
- `server/agent-server-tls/`: Backend-only server certificate and private key.
- `agent-trust/ca.pem`: the only generated trust material intended for Agent
  delivery.
- `agent-bootstrap/`, `agent-config/`, and `agent-state/`: owned by the later
  Runtime Target local-delivery preparation flow.

The Docker Builder Agent must never receive `server/credential-vault`,
`server/agent-server-tls`, the Vault root token, or a database connection.

If a local Vault restart occurs after a pending Agent delivery has been
created, run `graft dev docker-builder-agent reset` with the matching database
mode. It stops only this Compose project, archives Agent-local
`.data/docker-builder-agent-dev` state with a timestamp, removes the generated
ignored Agent config, and prepares a fresh local topology. In `shared` mode it
never resets, migrates, or archives the development database referenced by
`server/.env`. This is explicit because Runtime Target correctly permits only
one live delivery grant for a generation.
