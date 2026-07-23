Continue the `platform-self-update` topic with `$graft-multi-agent-loop` in `topic-completion-loop` mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/platform-self-update/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/AGENTS.md`
- design authority:
  - `ai-plan/design/release/upgrade-policy.md`
  - `ai-plan/design/release/migration-policy.md`
  - `compose.yml`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-task`

Topic objective:

- Deliver an auditable, manually confirmed Graft self-update lifecycle without weakening immutable-image and Compose deployment boundaries.

Work contract summary:

- Long-running cross-boundary feature with repository-wide release design, a roadmap, a topic, and a runner trust-boundary ADR.

Locked decisions:

1. Compose update execution uses a short-lived runner only after the read-only update path exists.
2. Binary installation is first-class for discovery and manual guidance, but not automatic replacement in the MVP.

Current batch plan:

1. `read-only-update-discovery`
2. `update-center-ui`
3. `backup-capability`
4. `compose-execution-and-recovery`
5. `archive-readiness`

Loop instructions:

- Default `loop_mode=topic-completion-loop`.
- Advance exactly one bounded batch this round.
- Update the topic tracking and trace files in the same change.
- Run the smallest required validation before closeout.
- Evaluate `$graft-commit` only after validation and only for confirmable owned scope.
