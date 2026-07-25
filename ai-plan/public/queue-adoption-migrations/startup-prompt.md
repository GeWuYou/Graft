Continue the `queue-adoption-migrations` topic in `topic-completion-loop` mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/queue-adoption-migrations/README.md`
- local execution truth: `server/AGENTS.md`, `web/AGENTS.md`, and `ai-plan/AGENTS.md`
- design authority: existing Task Runtime and queue-adoption design materials
- AI skills: `$graft-multi-agent-loop`, `$graft-multi-agent-task`, `$graft-validation-runner`

Topic objective:

- Migrate selected high-value request-blocking operations to the existing Task Runtime without adding protocol compatibility layers.

Work contract summary:

- `refactor / long-running / topic=true / design=false / roadmap=false / adr=false / graft-multi-agent-loop`

Locked decisions:

1. Repair canonical Task Runtime, module, and OpenAPI authorities together.
2. Stop and split before the local change set reaches 80 files.

Current batch plan:

1. Validate Docker image pull Task adoption and record the batch outcome.
2. Select the next bounded Container operation from the pending backlog.

Loop instructions:

- Advance one bounded batch.
- Update the topic tracking and trace files in the same change.
- Do not add a dual NDJSON/Task endpoint or consumer fallback.
- Evaluate `$graft-commit` only after validation and confirmable ownership.
