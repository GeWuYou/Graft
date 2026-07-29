# Compose Update Policy Reset Startup Prompt

Continue the same `topic-completion-loop` after rerunning root startup preflight.

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/compose-update-policy-reset/README.md`
- design authority: `ai-plan/design/release/platform-self-update.md`, `ai-plan/design/decisions/ADR-007-compose-update-policy-reset.md`, and `ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md`
- required skills: `$graft-multi-agent-loop`, `$graft-validation-runner`

Topic objective:

- finish the Beta self-update reliability repair without retaining the old repository-plus-digest deployment contract.

Locked decisions:

1. Deployment `.env` owns explicit image references and `GRAFT_UPDATE_POLICY`.
2. Only `stable`, `beta`, `fixed`, and `manual` are supported; no `nightly` path exists.
3. The runner must verify pulled image digests against a verified release manifest before mutation.

Current batch plan:

1. Complete server/OpenAPI/runner policy and receipt/App Log semantics.
2. Complete Web policy selection and truthful stage-progress rendering.
3. Run cross-boundary validation and archive-readiness review.

Loop instructions:

- Advance one bounded batch per round and update this topic's tracking and trace materials in the same change.
- Do not add aliases, dual reads, or compatibility DTOs for removed image configuration keys.
- Use `Next-session startup prompt:` only in a terminal closeout.
