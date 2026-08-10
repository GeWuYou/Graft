# Docker Agent Local Validation

## Status

In progress. The independent Docker Builder Agent OCI package and real Vault
dev-TLS fixture now exist, but local lifecycle acceptance remains blocked before
bootstrap because the isolated Backend topology has not yet supplied its
PostgreSQL/migration/runtime-target preparation and server-owned delivery driver.

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
| Agent container startup | partial | image entrypoint and `--version` run with read-only root filesystem; full bootstrap is pending Backend preparation |
| Vault AppRole login | pass | real Vault dev-TLS fixture initializes AppRole and verifies login |
| Vault PKI issuance | pending | Backend must establish its CA-verified Vault connection during a real bootstrap |
| mTLS reconnect | not run | server-side tests exist, but no deployable Agent exists |
| Ledger receipt | not run | no Agent runtime exists to pull and acknowledge a snapshot |
| Agent restart recovery | not run | no Agent state volume or container exists |

## Missing Items

1. Isolated PostgreSQL service, migration execution, and a Runtime Target created through its normal control-plane authority.
2. A server-owned non-public conformance driver that resolves registered Runtime Target authorities and records normal delivery evidence without direct SQL.
3. Backend Agent bootstrap and mTLS listeners with deployment-mounted TLS material.
4. The Agent config/secret handoff from the normal delivery authority into the named Agent volumes.
5. A full restart/reconnect run and the resulting durable ledger/receipt evidence.

## Fixture

The real Vault fixture is defined under `deployments/compose/docker-builder-agent/`.
It uses Vault dev-TLS mode, AppRole login, and PKI issuance configuration through
the Vault API. `tests/conformance/docker-builder-agent/run.sh` refuses to run
without `CONFORMANCE_DRIVER_CMD`, which must invoke the existing authenticated
Runtime Target/moduleapi workflow. The driver never writes the database and does
not provide fake certificate responses.

The fixture remains blocked until a server-owned command can invoke that normal
workflow, the isolated Backend topology has a migrated PostgreSQL database and
an existing Runtime Target, and the Agent bootstrap listener is available.
Current blockers:

`Runtime Target enrollment/delivery driver seam is not exposed as a runnable server-owned command; isolated Backend database/target preparation is absent; Vault revocation durable reconciliation pending.`

## Root Cause

The server-side Runtime Target trust boundary, deployable Agent package and real
Vault fixture are present. Existing focused tests and image startup cannot prove
container secret injection, Backend-to-Vault issuance, mTLS receipt, or restart
recovery until the normal server-owned delivery workflow is runnable against an
isolated migrated topology.

## Next Smallest Fix

Add the server-owned conformance driver and isolated Backend preparation through
the normal Runtime Target authority. Do not publish Agent or Operator OpenAPI
until all Phase 2 through Phase 5 gates pass against those real containers.
