# Platform Self Update

## Final Status Summary

- Topic objective: deliver Graft self-update as a controlled platform lifecycle capability.
- Current status: `archive-ready`
- Task class: `cross-boundary`
- Intake summary: long-running feature using the default `topic-completion-loop`.
- Canonical authority:
  - `ai-plan/design/release/`
  - `compose.yml`
  - `server/modules/*` and `web/src/modules/*` module conventions
- Completed scope: release-manifest authority, verified discovery cache, Update Center, independent backup and Task
  capabilities, manifest-pinned one-shot Compose runner, durable Update history, protected execution APIs, and Compose
  smoke evidence.
- Archive-ready verdict: `confirmed` after the final cross-boundary validation on 2026-07-23.

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

## Final Closeout

- The Compose executor consumes Task and Backup capabilities; it does not own their facts or persistence.
- Official execution requires a fresh verified catalog, exact manual version confirmation, and manifest-pinned server,
  web, and runner digests. Binary deployments remain verified manual guidance only.
- The runner is non-root, one-shot, no-HTTP, and receipt-writing. Post-migration failure is `NEEDS_ATTENTION`; the
  system does not promise automatic database rollback.
- Future automatic binary replacement, multi-node orchestration, Kubernetes execution, and controlled database restore
  require a separate topic and design authority.

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full `Work Contract` in the tracking file, not here.

## Archive Evidence

- Final implementation head: `01abcd32 feat(update): enforce release rollout authority`.
- The prerequisite-to-rollout commit chain begins at `47358a21` and contains 17 scoped commits.
- The active recovery index no longer routes to this topic. Historical evidence remains in this directory.

## Validation Targets

```bash
cd server && go run ./cmd/graft validate backend
cd web && bun run check
scripts/smoke_compose_runner.sh
git diff --check
```

## Loop Entry

- Preferred entry: `ai-plan/public/platform-self-update/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
