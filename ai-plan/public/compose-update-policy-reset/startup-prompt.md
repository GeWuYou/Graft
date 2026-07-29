# Compose Image Tag Strategy Reset Startup Prompt

Continue the same `topic-completion-loop` after rerunning root startup preflight.

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/compose-update-policy-reset/README.md`
- design authority: `ai-plan/design/release/platform-self-update.md`, `ai-plan/design/decisions/ADR-007-compose-update-policy-reset.md`, and `ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md`
- required skills: `$graft-multi-agent-loop`, `$graft-validation-runner`

Topic objective:

- finish the Beta self-update reliability repair without retaining the old repository-plus-digest deployment contract.

Locked decisions:

1. Deployment `.env` owns `GRAFT_IMAGE_TAG` as the shared official image tag and sole update strategy: `latest` tracks stable, `beta` tracks Beta, and SemVer tags are fixed releases.
2. A fixed upgrade can select only a strictly newer verified release in the same channel. Tracking/fixed or channel switching is a separate operation and is out of scope; no `nightly` path exists.
3. The runner must resolve a verified manifest release and digest at runtime, verify pulled image digests before mutation, and preserve `latest` or `beta` in `.env` after a tracking update.

Current batch plan:

1. Complete cross-boundary validation and verified PR-review remediation.
2. Run archive-readiness review.

Loop instructions:

- Advance one bounded batch per round and update this topic's tracking and trace materials in the same change.
- Do not add `GRAFT_UPDATE_POLICY`, aliases, dual reads, or compatibility DTOs for removed image configuration keys.
- Use `Next-session startup prompt:` only in a terminal closeout.
