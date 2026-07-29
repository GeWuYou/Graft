# Compose Update Policy Reset

## Current Status Summary

- Topic objective: reset the Beta self-update deployment contract around explicit Compose image references and a declared update policy.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: a Beta reliability repair needs coordinated Compose, runner, API, Web, and release-governance changes.
- Canonical authority: official `compose.yml` / `.env` deployment contract, verified GitHub Release manifests, and the platform-update lifecycle design.
- Completed so far: Work Contract bootstrap and ADR-007 are created.
- Not started yet: server/web contract, runner, observability, and UI implementation.

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

1. The deployment `.env` owns `GRAFT_UPDATE_POLICY` and complete server/web image references.
2. `stable`, `beta`, `fixed`, and `manual` are the only policy values; `manual` never automates image changes.
3. The policy reset intentionally has no old digest-key compatibility branch.

## Current Recovery Point

- First batch is resetting documented Compose authority and establishing the active topic.
- Next step: align runner write semantics, API contract, and Web policy selection with ADR-007.

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
