# Docker Agent Local Validation

## Status

The isolated conformance topology and fixture-only server-owned driver have
completed local lifecycle acceptance. The fixture used supplied Backend, Agent,
and driver images plus an immutable Agent OCI digest; it did not substitute
synthetic identities or bypass the normal Backend lifecycle.

## Environment

- Date: 2026-08-10 (Asia/Shanghai)
- Platform: WSL Ubuntu 24.04.4
- Docker client/server: 29.2.1 / 29.2.1
- Docker Compose: v5.1.0
- Branch: `feature/agent-bootstrap-security`
- Worktree: clean before this report

Existing running containers include the local Graft server/web validation instances, PostgreSQL and Redis. No Vault or
Graft Agent container is running.

## Phase Results

| Gate | Result | Evidence |
| --- | --- | --- |
| Docker daemon | pass | `docker version` returned client/server `29.2.1` |
| Docker Compose | pass | `docker compose version` returned `v5.1.0` |
| Agent OCI image build | pass | `server/agents/docker-builder-agent/Dockerfile` builds `graft/docker-builder-agent:conformance` |
| Agent container startup | pass | two `--once` lifecycles ran against the fixture with the Agent state volume persisted between them |
| Vault AppRole login | pass | real Vault dev-TLS fixture initializes AppRole and verifies login |
| Vault PKI issuance | pass | first bootstrap submitted its CSR through Backend-owned Credential Vault AppRole integration and activated generation 1 |
| mTLS reconnect | pass | Agent reconnected through the dedicated mTLS listener with the Vault-issued certificate |
| Ledger receipt | pass | first mTLS lifecycle received and consumed a certificate-bound ledger snapshot receipt |
| Agent restart recovery | pass | second one-shot Agent reused the same identity/generation and increased the accepted receipt count from 1 to 2 |

## Missing Items

1. CI still needs to provide immutable Backend, Agent, and fixture-only driver images plus the Agent digest.
2. Retain the generated redacted service logs as CI lifecycle evidence.

## Fixture

The real Vault fixture is defined under `tests/conformance/docker-builder-agent/`.
It runs PostgreSQL, the normal migration command, Backend, Vault, the
build-tagged fixture driver, and the Agent in separate services. The driver uses
Runtime Target and moduleapi authorities and never writes lifecycle rows
directly or provides fake certificate responses.

## Runtime Evidence

`tests/conformance/docker-builder-agent/run.sh` completed all eight stages on
2026-08-10. The redacted service log is retained at
`/tmp/docker-agent-conformance.log` in this local environment. The proof uses
real containers, Backend TLS listeners, Vault AppRole/PKI, and a Docker daemon
through the production-owned lifecycle rather than a fixture database shortcut.

## Re-run

Run `tests/conformance/docker-builder-agent/run.sh` with the required images and
digest to repeat the full lifecycle gate.
