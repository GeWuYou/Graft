Continue the `task-submission-lifecycle` topic after completing root `AGENTS.md` startup preflight.

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/task-submission-lifecycle/README.md`
- local execution truth: `server/AGENTS.md`; `web/AGENTS.md`; `ai-plan/AGENTS.md`
- design authority: `ai-plan/design/domains/task/任务提交生命周期设计.md`; `ai-plan/design/decisions/ADR-022-task-submission-materialization.md`
- AI skills: `$graft-multi-agent-loop`, `$graft-multi-agent-batch`, `$graft-table-design`, `$graft-sql-migration`, `$graft-validation-runner`

Topic objective:

- Replace task-level Reservation activation with an independent Submission lifecycle and atomically materialize ready Tasks for local prerequisites.

Current batch plan:

1. Establish Task ready and Submission stable contracts.
2. Add submission persistence and migrate Build to the generic materializer.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- Update tracking and trace files in the same change.
- State the completed batch, authority owner, and validation evidence.
