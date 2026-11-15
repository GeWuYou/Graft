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

### Local Task execution

`compose.local.yml` is a standalone private-engine fixture for local execution. It runs PostgreSQL, Redis, a private Docker daemon, workspace initialization, migrations and the backend. Its server image is the sole application image required for registration, with an exact source-revision label verified by the same runner.

```sh
python3 scripts/validate_docker_isolated.py init \
  --execution-mode local \
  --directory .local/docker-isolated/local \
  --project graft-isolated-local \
  --source-revision <full-commit> \
  --backend-image <backend-image> \
  --port 28080
python3 scripts/validate_docker_isolated.py up --directory .local/docker-isolated/local
python3 scripts/validate_docker_isolated.py probe --directory .local/docker-isolated/local
python3 scripts/validate_docker_isolated.py stop --directory .local/docker-isolated/local
```

The local probe discovers the canonical Docker target through the public API, exercises the same container and Compose operations, checks import persistence, and kills the backend after an observable application side effect. It requires the restarted Task to enter `needs_attention` while preserving the container identity and start time. It also verifies read-only mode preflight and a failed image pull against a closed loopback registry port. Failed required assertions return a nonzero exit code and retain scoped evidence for diagnosis.

For the full fresh-install path, supply `--web-image <same-source-web-image>` during registration and run `official-install` after `up`. Registration freezes the official Compose file, environment template and fixture files directly from the declared Git commit, recording their SHA-256. The action imports the verified immutable server/web images into the private daemon, stages the official root there and runs its configuration check, migrations, root initialization and complete server/web topology. It then repeats the API and engine probes through the nested web service on `--port + 1`. Restricted synthetic environment files stay under the registered evidence directory. Repeated execution preserves the fixture's existing environment and data.

```sh
python3 scripts/validate_docker_isolated.py official-install --directory .local/docker-isolated/local
```

For the real Controller restart boundary, also register a same-source `--runner-image` and run `controller-smoke` after `official-install`. This action imports the Controller into the private daemon, publishes the actual image bytes to its task-private registry, obtains real repository digests and runs the production Controller with a test-only image-reference overlay. It requires a successful local-mode receipt bound to the operation and runner, backup hashes and nonempty dumps, a changed server container identity and matching durable state after replacement. Its evidence explicitly covers the Controller protocol with same-source image retags. Public Update API submission, release discovery and Task settlement have a separate end-to-end acceptance boundary.

```sh
python3 scripts/validate_docker_isolated.py controller-smoke --directory .local/docker-isolated/local
```

### Public Update API through server replacement

`update-api` uses a separate fresh local registration. Build source and target server images from the same frozen commit with build versions `0.0.0-beta.1` and `0.0.0-beta.2`, and record both binary hashes. Build `Dockerfile.release-source` from this directory with the same revision label as the server, web and Controller images. Register the source server, target server, web, Controller and release-fixture image IDs, then run `up`, `official-install` and `update-api`.

```sh
python3 scripts/validate_docker_isolated.py init \
  --execution-mode local \
  --directory .local/docker-isolated/update-api \
  --project graft-isolated-update-api \
  --source-revision <full-commit> \
  --backend-image <source-server-image> \
  --target-backend-image <target-server-image> \
  --web-image <web-image> \
  --runner-image <controller-image> \
  --release-fixture-image <release-source-image> \
  --port 28180
python3 scripts/validate_docker_isolated.py up --directory .local/docker-isolated/update-api
python3 scripts/validate_docker_isolated.py official-install --directory .local/docker-isolated/update-api
python3 scripts/validate_docker_isolated.py update-api --directory .local/docker-isolated/update-api
```

The fixture creates an ephemeral CA and HTTPS endpoints for the standard GitHub release URLs inside its private Docker network. A TLS registry inside that daemon serves the real official `ghcr.io` image names and repository digests. Only the registered daemon's hosts/certificate directory and nested server's standard `SSL_CERT_FILE` mount receive fixture trust. Production release parsing, manifest checksum verification, image digest validation, Update API, Task execution, Backup and Controller code run through their normal entrypoints.

The source image version and original image tag are required before any fixture trust is installed. A missing deployment-environment snapshot is then injected and restored to verify a failed Backup and Task with the server retained. The normal update must settle both launch and final Task stages, retain attempt-bound receipts, replace the server with the registered target image, report the target build version and expose persisted backup bytes matching the receipt hashes and sizes. Evidence under `release-api/update-api-evidence.json` explicitly identifies the synthetic release source and its source/target binary hashes, CA, overlay and manifest fingerprints. Run guarded outer `destroy` with the registered project identity after inspection to remove every nested resource.

Local fixture stop pauses all registered services. Guarded destroy removes only that instance's project and named volumes. Agent-mode registrations keep the existing Vault-retention lifecycle described below. Build publication and full platform self-update have separate acceptance boundaries, because this API fixture's backend runs outside its private daemon's managed topology. A full self-update exercise must install the official Compose topology inside that daemon and provide verified release images and controller artifacts from the tested commit.

### Agent execution

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

The runner verifies the rendered socket boundary before startup, performs normal migrations and Backend-owned enrollment, starts the regular polling Agent, verifies bootstrap through bounded driver retries, stops and restarts that service, and verifies the persisted identity and receipt advancement. This startup lifecycle uses the regular polling service. The probe separately stops that service, runs a task-named `--once` container with a 60-second limit, requires successful process exit together with the same persisted identity and an advanced ledger receipt, removes any leftover one-shot container, and restarts the polling service. The API probe changes the disposable administrator password, enables the fixture's container actions, checks list/logs and start/stop/restart Tasks against actual engine state, verifies a Shell command over the actual WebSocket bridge, creates and deploys a Compose application with a relative bind mount, imports a workspace, verifies its display name, runtime target, Compose name and name source through the API and a read-only PostgreSQL query, requires a second import to return conflict with the persisted fields unchanged, and records missing Compose material, execution interruption, Backend socket permission denial and Agent-offline behavior. The interruption probe kills the Agent only after observing its application container in the private engine, then compares Task settlement and container identity after reconnect.

Public import rejects an already registered workspace before the repository upsert. The repeat-import scenario verifies that public conflict contract. Direct repository regression covers the upsert update branch.

`instance.json` and command diagnostics contain synthetic secrets and remain restricted local files. Share `trust-evidence.json` and `api-evidence.json` after inspecting them. Each API scenario carries its actual verification scope. The fixture Agent runs as root for the existing bootstrap-secret contract. The socket permission test compares fresh curl connections under the Backend UID before and after chmod, while recording the existing API connection separately. An already-open Unix connection can remain usable after socket file permissions change.

`stop` pauses the registered workload and dependency services while retaining the running development Vault service. Vault dev keeps its PKI and AppRole authority in process memory, so this retained service preserves the trust represented by the fixture's delivered files and `.initialized` marker. `status` continues to show that task-owned Vault service. `up` verifies a fresh AppRole login and checks the live PKI CA against the delivered CA before migrations or Backend startup, saving the non-secret result in `vault-authority-evidence.json`.

For a bounded resume check, run `up`, save the authority fingerprint, run `stop`, confirm Vault remains running and the workload services are stopped, then run `up` again and compare the authority fingerprint and Agent restart evidence. If Vault was stopped or recreated independently, its dev authority is lost and `up` rejects the stale material. Destroy that disposable fixture and initialize a fresh task instance.

After verified delivery, remove only the registered fixture resources with `destroy --confirm-project <same-project>`. Its named volumes contain disposable data. The command never invokes global Docker pruning. The Backend's deployment/self-update discovery sees the private engine, while the Backend container itself runs in the fixture's outer project. Self-update acceptance therefore needs a separate full deployment inside the private engine.
