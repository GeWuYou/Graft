# Compose Image Tag Strategy Reset

## Current Status Summary

- Topic objective: reset the Beta self-update deployment contract around `GRAFT_IMAGE_TAG` as the one shared Compose image tag and update strategy.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: a Beta reliability repair needs coordinated Compose, runner, API, Web, and release-governance changes.
- Canonical authority: official `compose.yml` / `.env` deployment contract, verified GitHub Release manifests, and the platform-update lifecycle design.
- Completed so far: Compose authority reset, ADR-007, runner tag-strategy and receipt reliability, server/OpenAPI tag and release catalog handling, Web strategy rendering and progress behavior, and the `update_mode` to `deployment_strategy` operation-snapshot convergence.
- Remaining: cross-boundary validation, PR-review remediation, and archive-readiness review.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: official Compose deployment configuration is the canonical owner; no legacy digest configuration compatibility is retained.

## Owned Scope

- `compose.yml`, `compose.env.example`, release Compose smoke fixtures
- platform-update server, Web, OpenAPI, and release-governance authority documents

Out of scope:

- `nightly` channel support
- automatic database rollback, systemd/binary mutation, or a persistent Update Agent

## Locked Decisions

1. The deployment `.env` owns `GRAFT_IMAGE_TAG` as the shared official server/web image tag and the only update strategy: `latest` tracks stable, `beta` tracks Beta, and SemVer tags are fixed releases.
2. The resolved manifest release and digest are runtime state, not a second configuration value. Tracking tags remain in `.env` after a successful update; a fixed upgrade writes only the selected higher same-channel fixed tag.
3. The tag-strategy reset intentionally has no `GRAFT_UPDATE_POLICY`, old digest-key, alias, dual-read, or fallback compatibility branch.
4. `update_operations.deployment_strategy` is the only operation-snapshot name across persistence, API, and audit metadata. The released `300001` migration remains immutable; `300002` converges its historical `update_mode` column and constraint to the canonical name without an alias.

## Current Recovery Point

- The policy, runner, server/OpenAPI, Web, and deployment-strategy snapshot implementation batches are complete.
- The affected `0.11.0-beta.22` release omitted the migration required by its server write path. Normal official Compose startup still runs `bootstrap` before server traffic; recovery is an operator bootstrap of the corrected release, not an application-level fallback.
- Next step: finish cross-boundary validation and archive-readiness review, including verified PR-review remediation.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is in `todos/compose-update-policy-reset-tracking.md`.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/compose-update-policy-reset/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
