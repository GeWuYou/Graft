---
name: graft-local-validation-maintenance
description: Safely update, inspect, or refresh an existing local Graft three-version validation deployment. Use for a selected instance's image tag, Compose/config refresh, port change, status, logs, health checks, or `.local/README.md` access record while retaining its data by default.
---

# Graft Local Validation Maintenance

Maintain an existing local validation deployment through the shared repository
command:

```bash
python3 scripts/local_graft_validation.py
```

This skill is incremental. Use `graft-local-validation-bootstrap` for first
creation or explicitly requested full recreation.

## Preconditions

1. Establish the root `AGENTS.md` startup receipt and read
   `.ai/environment/tools.ai.yaml` before selecting local tools.
2. Treat official `compose.yml` and `compose.env.example` as the deployment
   contract. This skill may refresh local generated files but must not define a
   second startup or validation path.
3. Identify the target instance and inspect its current tag, Compose status,
   port, data paths, and `.local/README.md` record before changing it.
4. Run port preflight and rendered-Compose validation before applying a port,
   Compose, or configuration update.

## Incremental Workflow

1. Use the command's targeted operations to update only the requested instance:
   refresh tracking tags, set an explicit fixed tag, or update supported local
   configuration. After pulling, synchronize the official Compose contract from
   the pulled server image's OCI source revision.
2. Preserve the selected instance's database, application, backup, import, and
   update-state data by default. Run any available backup step before a
   potentially disruptive configuration change.
3. Pull/restart only when required by the requested change, then validate the
   rendered configuration, container status, and configured health endpoints.
4. Refresh `.local/README.md` with the active tag, URL, state, access method,
   and local-only credential record. Do not copy credential values or
   machine-specific paths into tracked files.

## Access and Diagnostics

Use the shared command to expose the current environment access methods:

- `status` for instance, Compose service, image-tag, and health state.
- `logs` for a selected instance or service.
- `doctor` for port, rendered-Compose, and endpoint diagnostics.
- `up`, `down`, and `restart` for explicit lifecycle actions.
- `.local/README.md` for the current URLs, initial/admin access notes, and
  operator-recorded password after it changes.

## Safety Rules

- Keep `GRAFT_APP_ENV=production` and retain the complete deployment features.
- Never reset data as an incidental maintenance step. Require an explicit reset
  request, target one instance, and require the tool's confirmation.
- Treat fixed-tag image availability as an explicit result. Do not fall back to
  `beta` or `latest` when the configured fixed image is unavailable.
- Validate before and after a configuration, tag, port, or Compose update; report
  validation limits honestly when Docker or an image is unavailable.
