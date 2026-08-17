Continue the Dashboard Workbench Redesign topic after rerunning the root startup preflight.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/dashboard-workbench-redesign/README.md`
- local execution truth:
  - `web/AGENTS.md`
  - `server/AGENTS.md`
  - `ai-plan/AGENTS.md`
- design authority:
  - `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`
  - `ai-plan/public/dashboard-workbench-redesign/roadmap/dashboard-workbench-redesign-roadmap.md`
- AI skills:
  - `$graft-web-vibe-coding`
  - `$graft-openapi-contract-review`
  - `$graft-cross-boundary-review`
  - `$graft-localization-governance`
  - `$graft-validation-runner`
  - `$graft-task-closeout`

Topic objective:

- Build a stable operational workbench whose information priority and geometry do not grow linearly with module count.

Work contract summary:

- Feature, long-running, with design/topic/roadmap required and no ADR; implementation is dispatched through the specialized Web workflow.

Locked decisions:

1. Keep contribution facts server-owned and presentation status dashboard-owned.
2. Keep the preview development-only until the migrated formal homepage receives human acceptance after the corrective repair, then delete it.
3. Do not equate unknown evidence or a source loader failure with a confirmed business error.
4. Keep `title_key` authoritative and fallback `title` optional for alert items.
5. Keep PostgreSQL and Redis homepage health facts owned only by CapabilityCoordinator; Monitor owns active anomalies.

Implementation guardrails:

- Repair the highest available authority first.
- Repair OpenAPI source before generated projections; do not hand-edit generated artifacts.
- Keep the Dashboard Registry shape and capability/monitor public APIs unchanged.
- Keep visible copy in the dashboard locale catalogs.
- Use TDesign Vue Next and repository tokens only.

Current batch plan:

1. The alert optionality and platform-health authority repairs are complete and fully validated.
2. Manually refresh the migrated production `/` in the already-open authorized in-app browser, inspect the key-item and health rows, and record annotations or acceptance.
3. After formal-homepage acceptance, delete the development preview in one bounded slice; plan typed attention evolution separately.

Validation expectations:

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
just openapi-check
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

Required closeout:

- Update topic tracking and trace in the same change.
- Record TDesign MCP, browser, validation, shared-asset, and comment-governance evidence.
