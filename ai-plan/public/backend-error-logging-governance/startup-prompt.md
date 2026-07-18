Continue backend error logging governance inside the same `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `subtopic`
- recovery entry: `ai-plan/public/backend-error-logging-governance/README.md`
- local execution truth: `server/AGENTS.md` and `ai-plan/AGENTS.md`
- design authority: `ai-plan/design/domains/audit/日志治理开发规范.md`
- AI skills: `$graft-multi-agent-loop`, `$graft-multi-agent-task`, `$graft-validation-runner`

Topic objective:

- Make backend failures operationally diagnosable by preserving causes, recording each system error once with context, keeping Access Log factual, and recovering panics with a correlated stack.

Work contract summary:

- Long-running refactor; design and topic required; roadmap and ADR not required; execute by looped multi-agent tasks and archive after acceptance.

Locked decisions:

1. `AppLogger` plus `ReportError` is the ordinary application/system-error reporting path.
2. `httpx` maps typed errors and only logs an unreported internal fallback.
3. Access Log is always `INFO`; panic recovery is the sole full-stack path.

Implementation guardrails:

- Repair the highest available authority first.
- Preserve the existing HTTP error envelope and key-first i18n behavior.
- Keep causes out of public response data and do not add compatibility DTOs.
- Update tracking and trace in every batch.

Current batch plan:

1. Read the tracking file to select the current batch.
2. Advance exactly one bounded batch and commit it after validation.

Loop instructions:

- Default `loop_mode=topic-completion-loop`.
- Update batch state after each accepted closeout.
- Use `Next-session startup prompt:` only for terminal states.

Validation expectations:

```bash
cd server && go run ./cmd/graft validate backend
git diff --check
```
