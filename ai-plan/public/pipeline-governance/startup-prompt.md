Continue the `pipeline-governance` topic after completing root `AGENTS.md` startup preflight.

- governance source: root `AGENTS.md`
- task class: `docs/automation`
- recovery source: parent topic `migration-governance` for migration bootstrap semantics
- recovery entry: `ai-plan/public/pipeline-governance/README.md`
- local execution truth: `ai-plan/AGENTS.md`
- design authority: root `AGENTS.md`; `.github/workflows/pull-request-validation.yml`; `.github/workflows/migration-check.yml`
- AI skills: `$graft-multi-agent-batch`, `$graft-validation-runner`

Topic objective: complete remaining pipeline-governance batches without changing `Build Artifacts`, `Web Check`, or remote GitHub settings.

Work contract summary: `refactor`, `long-running`, active topic plus topic-local roadmap, batch-oriented execution.

Locked decisions:

1. Do not weaken blocking quality gates.
2. Require real GitHub Actions evidence before declaring the workflow-observation batch complete.

Current batch plan:

1. Run local validation for the first reliability batch.
2. After a maintainer pushes the branch, inspect the resulting pull-request workflow and repository settings checklist.

Validation expectations:

```bash
git diff --check
```

Required closeout: update tracking and trace files in the same change, report the changed authority, and use a next-session startup prompt only for a terminal handoff.
