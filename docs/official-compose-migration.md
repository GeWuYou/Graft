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
GRAFT_UPDATE_DEPLOYMENT_MODE=compose
```

`GRAFT_IMAGE_TAG` is the only image-version and update-strategy setting. It must be the same exact fixed tag used to clone the Compose release in step 1; do not add a second update-policy variable. In a later controlled upgrade, tracking tags such as `latest` and `beta` remain in `.env`, while the runner uses a manifest-derived release target only for the current upgrade. A fixed-tag upgrade atomically writes a newer verified fixed tag in the same channel.

Normally leave `GRAFT_UPDATE_COMPOSE_ROOT` unset. The server discovers its own Compose project through Docker and requires an administrator to confirm an ambiguous candidate. If discovery cannot identify the project, set this value to the absolute root from step 1:

```dotenv
GRAFT_UPDATE_COMPOSE_ROOT=/opt/graft
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

## Troubleshooting

| Checklist result | What to correct |
| --- | --- |
| Official Compose deployment is not detected | Start from the repository `compose.yml`, set `GRAFT_UPDATE_DEPLOYMENT_MODE=compose`, and recreate the instance from that root. |
| Compose project cannot be detected | Verify that the server mounts `/var/run/docker.sock`. If discovery remains unavailable, set `GRAFT_UPDATE_COMPOSE_ROOT` to the Docker daemon host's absolute Compose root. |
| Image strategy is invalid | Set `GRAFT_IMAGE_TAG` to `latest`, `beta`, or a compatible fixed release tag. |
| No release information is available | Check host access to the release endpoint, then check again. This does not mean an upgrade is available. |
| Upgrade permission is missing | Sign in with an account granted `platform-update.manage`. |

Do not add aliases, fallback image variables, or a custom runner to make a non-official deployment appear eligible. The path fails closed when it cannot prove the Compose root, topology, image strategy, and release identity.

## Rollback Boundary

Before database migration starts, the runner may restore the previous configuration snapshot and image references after a failed pre-migration step. After migration starts, a failed verification records `NEEDS_ATTENTION`; Graft does not automatically roll back or restore the database. Use the pre-migration backup and the incident process for post-migration recovery.
