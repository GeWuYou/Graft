# Compose Update Policy Reset

## Current Status Summary

- Topic objective: reset the Beta self-update deployment contract around one shared Compose image tag and a declared update policy.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: a Beta reliability repair needs coordinated Compose, runner, API, Web, and release-governance changes.
- Canonical authority: official `compose.yml` / `.env` deployment contract, verified GitHub Release manifests, and the platform-update lifecycle design.
- Completed so far: Compose authority reset, ADR-007, runner policy and receipt reliability, server/OpenAPI policy and release catalog handling, and Web policy selection and progress rendering.
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

1. The deployment `.env` owns `GRAFT_UPDATE_POLICY` and the shared `GRAFT_IMAGE_TAG` for fixed official server/web repositories.
2. `stable`, `beta`, `fixed`, and `manual` are the only policy values; `manual` never automates image changes.
3. The policy reset intentionally has no old digest-key compatibility branch.

## Current Recovery Point

- The policy, runner, server/OpenAPI, and Web implementation batches are complete.
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
