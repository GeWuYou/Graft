---
name: graft-local-validation-bootstrap
description: Create or deliberately recreate Graft's isolated local three-version Docker Compose validation environment. Use when a developer needs the initial `.local` deployment for `beta`, `latest`, and fixed `v0.11.0-beta.39`, or needs a requested full rebuild after confirming its data-reset impact.
---

# Graft Local Validation Bootstrap

Create the local validation environment through the shared repository command:

```bash
python3 scripts/local_graft_validation.py
```

This skill owns first creation and explicit recreation only. Use
`graft-local-validation-maintenance` for an existing environment.

## Preconditions

1. Establish the root `AGENTS.md` startup receipt and read
   `.ai/environment/tools.ai.yaml` before selecting local tools.
2. Treat the official `compose.yml` and `compose.env.example` as the deployment
   contract. Do not create a second Compose topology or validation truth.
3. Inspect the current `.local/graft-validation/` state. Do not overwrite
   existing instance data, `.env` values, or local credential records.
4. Run the command's port preflight before materializing or starting anything.
   Resolve host listener and Docker published-port conflicts before proceeding.

## Bootstrap Workflow

1. Run `init` to materialize the three isolated production-mode instances:
   `beta`, `latest`, and `fixed` using `v0.11.0-beta.39`.
2. Confirm each instance has unique Compose identity, published port, data paths,
   and update-state volume. Preserve the official `.data/postgres` convention.
3. Generate or refresh `.local/README.md`; it is the local access record and may
   contain local-only credentials because `.local/` is Git-excluded. Do not move
   credentials or machine-specific paths into tracked repository files.
4. Pull each requested target before starting it. The pull operation synchronizes
   that instance's official Compose files from the pulled server image's OCI
   source revision; then start it. Before reporting readiness, run the tool's
   rendered-Compose, service-status, and endpoint checks, then record the result
   in the local README.

## Safety Rules

- Keep `GRAFT_APP_ENV=production`; local startup does not justify development
  mode or reduced functionality.
- Never automatically reset, replace, or delete an existing instance. Require an
  explicit request and the tool's explicit reset confirmation for a full rebuild.
- If fixed tag `v0.11.0-beta.39` is unavailable, retain its generated
  configuration and report the unavailable image. Never substitute `beta` or
  `latest`.
- Keep all compiled modules and normal runtime facilities enabled, including the
  deployment's backup, import, Docker-socket, and update capabilities.

## Handoff

Report the generated instance URLs, image tags, port-preflight outcome, and the
path to `.local/README.md`. Direct later upgrades, configuration changes,
status checks, logs, and README refreshes to
`graft-local-validation-maintenance`.
