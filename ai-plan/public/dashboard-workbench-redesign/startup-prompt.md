Continue the Dashboard Workbench Redesign topic after rerunning the root startup preflight.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/dashboard-workbench-redesign/README.md`
- local execution truth:
  - `web/AGENTS.md`
  - `ai-plan/AGENTS.md`
- design authority:
  - `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`
  - `ai-plan/public/dashboard-workbench-redesign/roadmap/dashboard-workbench-redesign-roadmap.md`
- AI skills:
  - `$graft-web-vibe-coding`
  - `$graft-validation-runner`
  - `$graft-task-closeout`

Topic objective:

- Build a stable operational workbench whose information priority and geometry do not grow linearly with module count.

Work contract summary:

- Feature, long-running, with design/topic/roadmap required and no ADR; implementation is dispatched through the specialized Web workflow.

Locked decisions:

1. Keep contribution facts server-owned and presentation status dashboard-owned.
2. Keep the preview development-only until the migrated formal homepage receives human acceptance, then delete it.
3. Do not equate unknown evidence or a source loader failure with a confirmed business error.

Implementation guardrails:

- Repair the highest available authority first.
- Keep OpenAPI and Dashboard Registry shapes unchanged during the production Web adoption batch.
- Keep visible copy in the dashboard locale catalogs.
- Use TDesign Vue Next and repository tokens only.

Current batch plan:

1. Verify the migrated production `/` against the accepted preview.
2. After formal-homepage acceptance, delete the development preview in one bounded slice.
3. Plan typed attention contract evolution separately.

Validation expectations:

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
cd web && bun run check
```

Required closeout:

- Update topic tracking and trace in the same change.
- Record TDesign MCP, browser, validation, shared-asset, and comment-governance evidence.
