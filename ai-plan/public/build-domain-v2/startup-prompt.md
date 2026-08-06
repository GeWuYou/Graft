# Build Domain v2 Loop Startup Prompt

Continue the `build-domain-v2` topic using `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/build-domain-v2/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/AGENTS.md` when changing topic/design/roadmap materials
- design authority:
  - `ai-plan/design/architecture/build-domain-v2.md`
  - `ai-plan/roadmap/build-domain-v2.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-batch`
  - `$graft-validation-runner`

Topic objective:

- Deliver the approved four-phase Build Domain v2 without reintroducing Docker-first input, duplicate Runtime Target
  connections, or a second execution runtime.

Work contract summary:

- Long-running cross-boundary feature; repository-level design, roadmap and recoverable topic are required; execution
  uses `$graft-multi-agent-loop` and may use bounded `$graft-multi-agent-batch` slices.

Locked decisions:

1. Submission freezes Workspace Snapshot and Execution Plan before Task Runtime submission.
2. Runtime Target owns physical build capability, Task Runtime owns execution, and Artifact is an immutable first-class
   resource independent of Build Job.

Implementation guardrails:

- Repair the highest available authority first and promote the task to cross-boundary when shared contracts change.
- Do not introduce compatibility fields for Docker-first writes, arbitrary host paths, endpoint details or credentials.
- Treat Builder Pool/distributed fan-out as later work requiring truthful Runtime capability and explicit Task Runtime
  support.
- Consume the Work Contract already persisted in tracking; do not repeat Work Intake during normal batches.

Current batch plan:

1. `authority-bootstrap` establishes topic authority and recovery material.
2. The controller selects the next bounded Phase 1 slice after settling this batch.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the batch and authority owner changed this round.
- Update tracking and trace in the same change.
- Run the smallest required validation and scoped commit evaluation.
- Do not emit a next-session startup prompt except for a controller-selected terminal state.
