# Docker Builder Agent conformance fixture

This fixture is isolated from the production Compose and runner topologies. It
runs PostgreSQL, the normal `graft migrate up` command, Backend, Vault, an
internal conformance driver, and the independently built Docker Builder Agent.
The driver image is fixture-only: it is not a `graft conformance` production CLI
surface and it never exposes a public HTTP contract.

The Backend owns Runtime Target discovery, enrollment, delivery, activation,
and ledger receipts. The driver invokes those existing Backend/module
authorities; it does not define a Docker target representation or lifecycle
policy. It loads the fixture Backend configuration, opens the fixture database,
and receives the Vault AppRole material required to drive that Backend-owned
scenario. Vault owns PKI issuance. The Agent has no database connection and
receives neither Vault address nor AppRole material.

## Required images and inputs

```sh
export GRAFT_BACKEND_IMAGE=localhost/graft-server:conformance
export GRAFT_AGENT_IMAGE=localhost/graft/docker-builder-agent:conformance
export GRAFT_CONFORMANCE_DRIVER_IMAGE=localhost/graft/docker-builder-agent-conformance:latest
export GRAFT_CONFORMANCE_POSTGRES_PASSWORD=fixture-only-password
export GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST=sha256:<built-agent-image-digest>
export VAULT_DEV_ROOT_TOKEN=fixture-only-root-token
docker build -f tests/conformance/docker-builder-agent/Dockerfile -t "$GRAFT_CONFORMANCE_DRIVER_IMAGE" server
tests/conformance/docker-builder-agent/run.sh
```

`vault-init` uses Vault's API to configure an AppRole and PKI. It produces
Backend-only Vault credentials plus a server certificate/key for the dedicated
bootstrap TLS and Agent mTLS listeners. The public CA is copied to the separate
`agent-trust` volume. A prepare command writes the generated Agent config and
one-time bootstrap secret to separate fixture volumes, mounted read-only by the
Agent. Do not print, retain, or commit their contents.

The driver writes the one-time token at
`/conformance/agent-bootstrap/bootstrap-token`; the generated Agent config names
the Agent-visible `/run/graft-bootstrap/bootstrap-token` path and the separate
`/run/graft-agent-trust/ca.pem` CA path. This keeps driver-only write paths out
of the Agent configuration.

## Lifecycle gate

The runner executes fixed, non-injectable phases:

1. `prepare` establishes the Backend-owned Runtime Target, enrollment, and
   delivery receipt, then asserts that receipt acceptance alone is not active.
2. The first `--once` Agent execution performs bootstrap, Vault PKI issuance,
   mTLS reconnect, and its first ledger receipt.
3. `verify-bootstrap` requires certificate issuance, active generation, and the
   first accepted ledger receipt.
4. A second `--once` execution reuses the persisted Agent state.
5. `verify-restart` requires the same identity, no second enrollment or
   issuance, and a new accepted ledger receipt.

The test requires Docker socket access for Backend and Agent. Teardown with the
same Compose project after inspection; volumes contain fixture state and should
be removed only when that evidence is no longer required.

The fixture Dockerfile compiles only `cmd/graft-docker-builder-conformance` with
the `conformance` build tag. It is not part of the Backend production image or
the public `graft` CLI. The runner keeps the driver's JSON evidence in temporary
files, validates it with `jq`, and passes only non-secret identity, generation,
and receipt-count fields to the restart phase.

The one-shot Agent remains root in this fixture because it must read the
driver-created `0600` bootstrap token. The conformance driver runs as a
dedicated non-root user and owns its writable fixture mounts. Neither grants the
Agent Vault credentials or a database connection.
