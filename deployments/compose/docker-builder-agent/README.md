# Docker Builder Agent local conformance

This fixture is intentionally separate from the production Compose topology. It
starts a real Vault dev-TLS server, a Graft backend, and the independently
built `graft-docker-builder-agent` image. Vault is configured by
`vault-init` through the Vault HTTP API; no certificate or AppRole response is
mocked.

The backend image and Agent image are supplied by environment variables:

```sh
export GRAFT_BACKEND_IMAGE=localhost/graft-server:conformance
export GRAFT_AGENT_IMAGE=localhost/graft/docker-builder-agent:conformance
docker compose -f compose.yml up -d
```

`vault-init` writes RoleID, SecretID, and the CA bundle to the named
`vault-secrets` volume. The backend receives only the RoleID and SecretID files;
the Agent never receives Vault credentials.

The conformance driver must be given a command that invokes the existing
authenticated Runtime Target workflow (for example the repository's normal
operator/automation command):

```sh
CONFORMANCE_DRIVER_CMD='...' \
  tests/conformance/docker-builder-agent/run.sh
```

An unset driver command is a hard failure. This prevents the fixture from
claiming conformance based on container health or direct database writes.
