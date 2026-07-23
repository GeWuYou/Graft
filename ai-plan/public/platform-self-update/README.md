# Platform Self Update

## Current Status Summary

- Topic objective: deliver Graft self-update as a controlled platform lifecycle capability.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running feature using the default `topic-completion-loop`.
- Canonical authority:
  - `ai-plan/design/release/`
  - `compose.yml`
  - `server/modules/*` and `web/src/modules/*` module conventions
- Derived delivery automation: `.github/workflows/publish.yml` implements the release design; it does not define release policy.
- Completed so far: Work Intake, release-manifest authority, release design, delivery roadmap, Compose runner ADR, read-only update discovery, and Update Center UI.
- Next implementation batch: backup capability.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: release policy and artifacts, Compose deployment, server module/OpenAPI contracts, and web navigation.

## Owned Scope

- `ai-plan/design/release/**`, `ai-plan/design/decisions/**`, and topic recovery materials
- release workflow and manifest artifacts
- `server/modules/update/**`, `server/modules/backup/**`, related contracts, and update execution fixtures
- `web/src/modules/**` and shell navigation required by Platform Updates

Out of scope:

- unattended automatic installation, multi-node orchestration, and Kubernetes execution
- replacing a binary inside a running container or a persistent update-agent runtime

## Locked Decisions

1. `Platform -> Updates` is the global entry; the logo version affordance links to it.
2. GitHub Release metadata plus immutable GHCR image digests is the v1 release source.
3. Compose is the only executable MVP deployment type; binary installation is detected and guided, not automatically replaced.
4. Backup is an independent platform capability consumed by update execution.

## Phase Plan

- establish release manifest, design authority, roadmap, and runner trust-boundary ADR
- deliver read-only update discovery, installation profile, APIs, scheduler, and Update Center
- add independent backup capability, then Compose execution and recovery evidence

## Current Recovery Point

- Read-only update discovery and Update Center UI are complete and committed.
- The deployment-profile capability matrix must remain authoritative over a declared environment value alone.
- Next step: `backup-capability`.

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full `Work Contract` in the tracking file, not here.

## Pending Batch Direction

- `backup-capability`
- `compose-execution-and-recovery`
- `archive-readiness`

## Validation Targets

```bash
git diff --check
```

## Loop Entry

- Preferred entry: `ai-plan/public/platform-self-update/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
