Copy to `ai-plan/public/<topic>/startup-prompt.md` and replace every placeholder before use. Keep the prompt aligned
with root `AGENTS.md`; do not invent a second startup receipt.

Continue work inside the same `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `<server | web | cross-boundary | docs/automation>`
- recovery source: `<none | parent topic | subtopic>`
- recovery entry: `ai-plan/public/<topic>/README.md`
- local execution truth:
  - `<local AGENTS or README path when applicable>`
- design authority:
  - `<authority file or document 1>`
  - `<authority file or document 2>`
- AI skills:
  - `$graft-multi-agent-loop`
  - `<other required skill or remove this line>`

Topic objective:

- <describe the topic objective>

Work contract summary:

- <kind / scope / required artifacts / execution engine>

Locked decisions:

1. <decision>
2. <decision>

Implementation guardrails:

- Repair the highest available authority first.
- Keep work inside the owned scope and record any required escalation.
- Do not add future-batch artifacts early.
- Consume the existing `Work Contract`; do not re-decide whether this topic needs `design`, `roadmap`, `ADR`, or
  `topic` creation inside ordinary execution rounds.

Current batch plan:

1. `<current batch>`
2. `<next batch>`

Loop instructions:

- Default `loop_mode=topic-completion-loop`.
- Advance exactly one bounded batch this round.
- Update the topic tracking and trace files in the same change.
- Run the smallest required validation before closeout.
- Evaluate `$graft-commit` only after validation and only for confirmable owned scope.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the current batch.
- State the authority owner changed this round.
- State which validation ran and why stronger checks were skipped.
- Update loop batch state.
- Use `Next-session startup prompt:` only for terminal states.
