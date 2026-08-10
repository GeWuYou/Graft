# Docker Builder Agent Local Dependencies

This development-only topology starts isolated PostgreSQL, Redis, and Vault for
a host-debugged Docker Builder Agent. It is not part of the production Compose
topology and uses only the ignored `.data/docker-builder-agent-dev` directory.

`graft dev docker-builder-agent prepare` supplies generated values for the
required environment variables and starts `vault-init`. Do not run this
topology with production credentials, production Vault addresses, or an
existing production data directory.

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
created, run `graft dev docker-builder-agent reset`. It stops only this Compose
project, archives `.data/docker-builder-agent-dev` with a timestamp, removes
the generated ignored Agent config, and prepares a fresh local topology. This
is explicit because Runtime Target correctly permits only one live delivery
grant for a generation.
