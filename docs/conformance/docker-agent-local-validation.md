# Docker Agent Local Validation

## Status

Blocked at Phase 2. No Docker Agent OCI package exists in the current checkout, so a real Docker + Vault + Agent
lifecycle cannot be started without first adding the explicitly missing Agent runtime.

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
| Agent OCI image build | blocked | no `cmd/graft-agent`, Agent binary, or Agent Dockerfile in `server/**` |
| Agent container startup | not run | no image can be built |
| Vault AppRole login | not run | no Vault service/configuration available |
| Vault PKI issuance | not run | no Agent client or real Vault fixture available |
| mTLS reconnect | not run | server-side tests exist, but no deployable Agent exists |
| Ledger receipt | not run | no Agent runtime exists to pull and acknowledge a snapshot |
| Agent restart recovery | not run | no Agent state volume or container exists |

## Missing Items

1. Dedicated `graft-agent` command and binary.
2. Agent image with `/app/graft-agent`, `/etc/graft/config`, and `/var/lib/graft-agent/state`.
3. Agent-side bootstrap, local private-key/state persistence, TLS 1.3 mTLS pull/ack, and reconnect logic.
4. Docker Compose Vault dev/server fixture with mounted AppRole RoleID/SecretID and PKI setup.
5. Docker-only conformance harness and image digest/version evidence.

## Root Cause

The server-side Runtime Target trust boundary is present, but the PR3 deployable Agent package and real Vault fixture
are absent from this checkout. Existing unit and HTTP tests cannot prove container startup, Docker-secret injection,
Vault AppRole authentication, or restart/reconnect behavior.

## Next Smallest Fix

Implement the dedicated Agent command/image and a derived local Vault Compose fixture. Do not publish Agent or Operator
OpenAPI until all Phase 2 through Phase 5 gates pass against those real containers.
