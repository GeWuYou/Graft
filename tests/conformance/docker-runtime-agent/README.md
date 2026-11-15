# Docker Runtime Agent conformance fixture

This fixture is isolated from the production Compose and runner topologies. It
runs PostgreSQL, the normal `graft migrate up` command, Backend, Vault, an
internal conformance driver, and the independently built Docker Runtime Agent.
The driver image is fixture-only: it is not a `graft conformance` production CLI
surface and it never exposes a public HTTP contract.

The Backend owns Runtime Target discovery, enrollment, delivery, activation,
and ledger receipts. The driver invokes those existing Backend/module
authorities; it does not define a Docker target representation or lifecycle
policy. It loads the fixture Backend configuration, opens the fixture database,
and receives the Vault AppRole material required to drive that Backend-owned
scenario. Vault owns PKI issuance. The Agent has no database connection and
receives neither Vault address nor AppRole material.

## Batch 5 Build SDK boundary

Build operations use the same Task-owned external execution lease as the other finite side effects. The authenticated
Agent must claim the `docker/oci-build` binding at capability version `docker/v1`, resolve fence-bound
`build-execution-material/v1` only after claim, and submit `build-execution-result/v1` before the terminal receipt. The
fixture must assert that Build receives only normalized artifact facts, Task persistence retains only the result digest,
and exact result replay is idempotent while a changed replay is rejected.

The fixture Agent has no inbound port. Backend and Agent share the named `build-snapshots` volume at
`/tmp/graft-build-snapshots`; material, Registry credentials, endpoints, host paths and commands remain transient and
must not appear in Task rows, logs, receipts or Agent journal. The Backend Docker socket remains only for explicitly
unmigrated Runtime Target discovery/summary, Container snapshot/resource/log/stream/event/stats and interactive reads,
Deployment Runtime Docker facts, and Update Controller observation/recovery; it is not a Build execution path.

## Required images and inputs

```sh
export GRAFT_BACKEND_IMAGE=localhost/graft-server:conformance
export GRAFT_AGENT_IMAGE=localhost/graft/docker-runtime-agent:conformance
export GRAFT_CONFORMANCE_DRIVER_IMAGE=localhost/graft/docker-runtime-agent-conformance:latest
export GRAFT_CONFORMANCE_POSTGRES_PASSWORD=fixture-only-password
export GRAFT_CONFORMANCE_AGENT_IMAGE_DIGEST=sha256:<built-agent-image-digest>
export VAULT_DEV_ROOT_TOKEN=fixture-only-root-token
docker build -f tests/conformance/docker-runtime-agent/Dockerfile -t "$GRAFT_CONFORMANCE_DRIVER_IMAGE" server
tests/conformance/docker-runtime-agent/run.sh
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

## Update rollout contract

The official bootstrap keeps `graft update cutover-v1` as the one-time authority
for legacy update-state cleanup and runs it before `graft migrate up`. A normal
Update Controller rollout is ordered and must not be shortened: update
`server`/`web`, verify the new server health, replace the Runtime Agent last,
then verify the new Agent's mTLS identity, active generation, and declared
capability readiness. The Backend socket in this fixture is retained only for
Runtime Target discovery/summary, Container snapshot/resource/log/stream/event/
stats and interactive reads, Deployment Runtime Docker facts, and bounded
Update observation/recovery; it is not an Update launch or Build execution path.

The fixture Dockerfile compiles only `cmd/graft-docker-runtime-conformance` with
the `conformance` build tag. It is not part of the Backend production image or
the public `graft` CLI. The runner keeps the driver's JSON evidence in temporary
files, validates it with `jq`, and passes only non-secret identity, generation,
and receipt-count fields to the restart phase.

The one-shot Agent remains root in this fixture because it must read the
driver-created `0600` bootstrap token. The conformance driver runs as a
dedicated non-root user and owns its writable fixture mounts. Neither grants the
Agent Vault credentials or a database connection.

## Dedicated Docker daemon verification

`compose.isolated.yml` overlays this fixture with a private Docker Engine. The Backend, Agent and fixture driver share only its named socket volume mounted at canonical `/run`. Alpine resolves `/var/run` through a symlink, so mounting `/run` lets the runtime order the parent volume before nested secret mounts. This private parent volume remains writable so the OCI runtime can create nested mountpoints, while the nested certificate and bootstrap volumes remain read-only. The daemon and application consumers use the same `/opt/graft/apps` volume path, so relative Compose bind mounts resolve against the daemon's actual filesystem. The Backend receives the private socket group explicitly.

`python3 scripts/validate_docker_isolated.py --help` exposes `init`, `up`, `probe`, `status`, `stop` and guarded `destroy`. Run orchestration from the stable checkout with Docker available. Build the three fixture images from the same exact commit and label each image with `org.opencontainers.image.revision=<commit>`. `init` verifies that label and saves immutable local image IDs. Supply an ignored task evidence directory, a unique `graft-isolated-<task>` project, and three unused loopback ports starting at `--port`.

```sh
python3 scripts/validate_docker_isolated.py init \
  --directory .local/docker-isolated/baseline \
  --project graft-isolated-baseline \
  --source-revision <full-commit> \
  --backend-image <backend-image> \
  --agent-image <agent-image> \
  --driver-image <fixture-driver-image> \
  --port 28080
python3 scripts/validate_docker_isolated.py up --directory .local/docker-isolated/baseline
python3 scripts/validate_docker_isolated.py probe --directory .local/docker-isolated/baseline
python3 scripts/validate_docker_isolated.py stop --directory .local/docker-isolated/baseline
```

The runner verifies the rendered socket boundary before startup, performs normal migrations and Backend-owned enrollment, verifies the persisted Agent identity across restart, then starts the regular polling Agent. The API probe changes the disposable administrator password, enables the fixture's container actions, checks list/logs and start/stop/restart Tasks against actual engine state, verifies a Shell command over the actual WebSocket bridge, creates and deploys a Compose application with a relative bind mount, imports a workspace, and records missing Compose material, execution interruption, Backend socket permission denial and Agent-offline behavior. The interruption probe kills the Agent only after observing its application container in the private engine, then compares Task settlement and container identity after reconnect.

`instance.json` and command diagnostics contain synthetic secrets and remain restricted local files. Share `trust-evidence.json` and `api-evidence.json` after inspecting them. Each API scenario carries its actual verification scope. The fixture Agent runs as root for the existing bootstrap-secret contract, so the socket permission test proves the non-root Backend read boundary.

After verified delivery, remove only the registered fixture resources with `destroy --confirm-project <same-project>`. Its named volumes contain disposable data. The command never invokes global Docker pruning. The Backend's deployment/self-update discovery sees the private engine, while the Backend container itself runs in the fixture's outer project. Self-update acceptance therefore needs a separate full deployment inside the private engine.
