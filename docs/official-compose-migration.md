# Migrate to the Official Compose Deployment

The Update Center runs controlled platform upgrades only for the official
Compose deployment. This guide moves an existing Graft instance to that model;
it does not make the Update Center a generic host command runner.

## Before You Start

- Schedule a maintenance window and keep an independently verified database backup.
- Record current images, Compose files, mounted data directories, environment values, and exposed ports.
- Run these commands on the Docker daemon host. Do not use a container-only path, a Windows/WSL translated path, or a path from another Docker host.
- The controlled Compose runner currently supports Linux `amd64` hosts only.

## 1. Prepare an Official Compose Root

Choose an absolute Docker daemon host directory, for example `/opt/graft`. Its top level must contain the official `compose.yml` (or `compose.yaml`) and the Compose project's `.env` file.

```bash
# Replace v0.11.0-beta.21 with the exact fixed official tag already deployed.
# Use that same tag in GRAFT_IMAGE_TAG below; do not change version or channel.
git clone --branch v0.11.0-beta.21 --depth 1 https://github.com/GeWuYou/Graft.git /opt/graft
cd /opt/graft
cp compose.env.example .env
```

Do not merge an old custom Compose file into the official file. Clone the exact release already deployed, then carry forward only supported deployment values, such as credentials, ports, mount locations, and allowed origins. Do not change the version or release channel during migration. The official topology includes `server`, `web`, `bootstrap`, `postgres`, and `redis`; the server also needs `/var/run/docker.sock` to discover the project and launch the short-lived upgrade runner.

## 2. Preserve Existing Data

Set existing persistent host directories in `.env` before the first `docker compose up`. Replacing them creates an empty instance rather than migrating the current one. Review the PostgreSQL mount in `compose.yml` and retain any existing optional roots:

```dotenv
GRAFT_APPLICATION_ROOT_HOST_PATH=/opt/graft/apps
GRAFT_BACKUP_ARTIFACT_HOST_PATH=/opt/graft/backups
GRAFT_PROJECT_IMPORT_HOST_PATH=/opt/graft/imports
```

Keep database credentials and `GRAFT_AUTH_JWT_SECRET` unchanged for an existing instance. Changing them in the same migration can prevent the new services from using the old database or invalidate active sessions.

## 3. Declare the Update Strategy

Configure the official values in `.env`:

```dotenv
# Use the exact fixed official tag already deployed and cloned in step 1.
# Do not change this version or its release channel during migration.
GRAFT_IMAGE_TAG=v0.11.0-beta.21
GRAFT_DEPLOYMENT_RUNTIME=compose
```

`GRAFT_IMAGE_TAG` is the only image-version and update-strategy setting. It must be the same exact fixed tag used to clone the Compose release in step 1; do not add a second update-policy variable. In a later controlled upgrade, tracking tags such as `latest` and `beta` remain in `.env`, while the runner uses a manifest-derived release target only for the current upgrade. A fixed-tag upgrade atomically writes a newer verified fixed tag in the same channel.

Normally leave `GRAFT_DEPLOYMENT_COMPOSE_ROOT` unset. Deployment Runtime discovers the server's Compose project through Docker and requires an administrator to confirm an ambiguous candidate. If discovery cannot identify the project, set this value to the absolute root from step 1:

```dotenv
GRAFT_DEPLOYMENT_COMPOSE_ROOT=/opt/graft
```

An explicitly blank, relative, invalid, or stale root blocks controlled upgrades. Graft does not fall back to a container path, binary updater, or unrelated Compose project.

## 4. Start and Verify

```bash
cd /opt/graft
docker compose pull
docker compose up -d
docker compose ps
```

Confirm that `bootstrap` completed successfully and `server`, `web`, `postgres`, and `redis` are healthy or running as expected. Then sign in as an administrator and select **Platform > Updates > Check for updates**. A single high-confidence candidate is usable directly; multiple or low-confidence candidates require a choice during the upgrade flow. Displayed evidence is diagnostic only: do not manually alter Docker labels.

`docker compose pull` and `docker compose up -d` are required to start the same fixed release selected above. Do not change `GRAFT_IMAGE_TAG`, version, or channel between cloning the release and running these commands.

## Recover an Affected `0.11.0-beta.22` Update Center

`0.11.0-beta.22` can fail while starting an update operation because its server expects the historical `update_mode` column before that migration was included in the release. This is not a normal Compose ordering failure: official Compose runs `bootstrap` before server startup and only exposes web after server health succeeds. Do not retry the in-app operation on the affected image.

First make and verify a database backup. The published corrective release for this incident is `v0.11.0-beta.23`; its
release manifest declares the following immutable image identities:

```text
ghcr.io/gewuyou/graft-server@sha256:b791e3d46d956ef9b0026cc64f90cb93e495633949506b0fae2457b63abedcaa
ghcr.io/gewuyou/graft-web@sha256:b30ef5b0647b1449051646b1d53fcf2dde54556bf221d65f6b45b07607356a0c
```

Set `GRAFT_IMAGE_TAG=v0.11.0-beta.23` in `.env`, verify those digests against the
`release-manifest.json` asset for that GitHub Release, and run the following on the Docker daemon host from the Compose root.
Do not continue if the manifest or image digests differ:

```bash
docker compose pull --policy always bootstrap server web
docker compose stop server web
docker compose run --rm bootstrap
docker compose up -d --no-deps --force-recreate server web
docker compose ps
docker compose exec -T server curl --fail --silent http://127.0.0.1:8080/healthz
```

The published `v0.11.0-beta.23` bootstrap applies the immutable `202607300001_update_operation_mode.sql`, which repairs the
missing-column failure in `0.11.0-beta.22`. The `202607300002_rename_update_operation_deployment_strategy.sql` migration is
not part of a published release yet; do not invent or select an unreleased tag. After this recovery, use only a later official
release whose verified manifest explicitly contains `300002` before expecting the canonical `deployment_strategy` schema.
The contract intentionally does not retain an `update_mode` API or storage alias once that later release is applied.

## Troubleshooting

| Checklist result | What to correct |
| --- | --- |
| Official Compose deployment is not detected | Start from the repository `compose.yml`, set `GRAFT_DEPLOYMENT_RUNTIME=compose`, and recreate the instance from that root. |
| Compose project cannot be detected | Verify that the server mounts `/var/run/docker.sock`. If discovery remains unavailable, set `GRAFT_DEPLOYMENT_COMPOSE_ROOT` to the Docker daemon host's absolute Compose root. |
| Image strategy is invalid | Set `GRAFT_IMAGE_TAG` to `latest`, `beta`, or a compatible fixed release tag. |
| No release information is available | Check host access to the release endpoint, then check again. This does not mean an upgrade is available. |
| Upgrade permission is missing | Sign in with an account granted `platform-update.manage`. |

Do not add aliases, fallback image variables, or a custom runner to make a non-official deployment appear eligible. The path fails closed when it cannot prove the Compose root, topology, image strategy, and release identity.

## Rollback Boundary

Before database migration starts, the runner may restore the previous configuration snapshot and image references after a failed pre-migration step. After migration starts, a failed verification records `NEEDS_ATTENTION`; Graft does not automatically roll back or restore the database. Use the pre-migration backup and the incident process for post-migration recovery.
