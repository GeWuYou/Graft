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
- Recheck Compose project: `graft-agent-recheck-20260810`
- Recheck ports: Backend `28080`, bootstrap TLS `28443`, Agent mTLS `28444`

Existing running containers include the local Graft server/web validation instances, PostgreSQL and Redis. No Vault or
Graft Agent container is running.

## Phase Results

| Gate | Result | Evidence |
| --- | --- | --- |
| Docker daemon | pass | `docker version` returned client/server `29.2.1` |
| Docker Compose | pass | `docker compose version` returned `v5.1.0` |
| Agent OCI image build | pass | `server/agents/docker-runtime-agent/Dockerfile` builds `graft/docker-runtime-agent:conformance` |
| Agent container startup | pass | two `--once` lifecycles ran against the fixture with the Agent state volume persisted between them |
| Vault AppRole login | pass | real Vault dev-TLS fixture initializes AppRole and verifies login |
| Vault PKI issuance | pass | first bootstrap submitted its CSR through Backend-owned Credential Vault AppRole integration and activated generation 1 |
| mTLS reconnect | pass | Agent reconnected through the dedicated mTLS listener with the Vault-issued certificate |
| Ledger receipt | pass | first mTLS lifecycle received and consumed a certificate-bound ledger snapshot receipt |
| Agent restart recovery | pass | second one-shot Agent reused the same identity/generation and increased the accepted receipt count from 1 to 2 |

Agent image digest observed during the recheck:

`sha256:bc0b70fdf84072bcee1e0384889a7e692d1c4150948a6be7517da63af0bc5690`

## Missing Items

1. CI still needs to provide immutable Backend, Agent, and fixture-only driver images plus the Agent digest.
2. A real Vault revoke request has not been added to this lifecycle runner; the existing revocation handler and durable event retry remain covered by server conformance tests.
3. Retain generated unredacted service logs only as restricted CI secret
   artifacts; do not commit or share them as ordinary build output.

## Fixture

The real Vault fixture is defined under `tests/conformance/docker-runtime-agent/`.
It runs PostgreSQL, the normal migration command, Backend, Vault, the
build-tagged fixture driver, and the Agent in separate services. The driver uses
Runtime Target and moduleapi authorities and never writes lifecycle rows
directly or provides fake certificate responses.

## Runtime Evidence

`tests/conformance/docker-runtime-agent/run.sh` completed all eight stages on
2026-08-10 under Compose project `graft-agent-recheck-20260810`. The runner log
is retained at `/tmp/docker-agent-conformance-rerun-isolated.log`; the
unredacted service log at `/tmp/docker-agent-conformance-services.log` is mode
`0600`, contains fixture secrets, and must not be committed or shared. The proof
uses
real containers, Backend TLS listeners, Vault AppRole/PKI, and a Docker daemon
through the production-owned lifecycle rather than a fixture database shortcut.

## Re-run

Run `tests/conformance/docker-runtime-agent/run.sh` with the required images and
digest to repeat the full lifecycle gate.

## Deterministic Build SDK evidence

The Docker Runtime Agent package now has an executable lease-to-provider seam
test in `server/agents/docker-runtime-agent/internal/build_provider_test.go`.
It verifies that `build-execution-material/v1` is fetched with the lease fence
token before the Moby SDK is called, that the Dockerfile context is sent through
the SDK, and that the returned value is the provider-neutral
`build-execution-result/v1` payload. The adjacent execution tests cover result
submission before terminal receipt, transient result transport recovery, and
no provider replay during digest/result replay.

Run the deterministic gate with:

```sh
cd server
go test ./agents/docker-runtime-agent/...
```

This is a local executable proof of the Agent boundary; it is not a substitute
for the real Compose fixture gate below.

## Batch 5 follow-up scope

The deterministic Build SDK gate is now executable locally through the Task-owned lease and the
`docker-runtime-agent` Moby/OCI SDK path. It verifies transient material/result handling and result-digest replay. The
remaining acceptance gap is the real Compose fixture
gate, which still requires immutable Backend, Agent, and fixture-only driver images, the Agent digest, and the Vault/
registry credentials described above. The Backend/Agent fixture must mount the named `build-snapshots` volume at
`/tmp/graft-build-snapshots`; the Agent still publishes no inbound port. Runtime Target discovery/summary, Container
read/stream/interactive and Update Controller remain explicit server-socket consumers.
