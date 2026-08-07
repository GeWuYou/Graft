Continue the Theme Preset Governance Audit after rerunning the root startup preflight.

Round context:

- governance source: root `AGENTS.md`
- task class: `web`
- recovery source: `parent topic`
- authority summary: writable theme authority state, editable token definitions, and shared derived CSS rules own rendered visuals; preset definitions may only materialize initial values into those owners.
- recovery entry: `ai-plan/public/theme-preset-governance-audit/README.md`
- local execution truth:
  - `web/AGENTS.md`
  - `ai-plan/AGENTS.md` when updating this topic
- design authority:
  - `DESIGN.md`
  - `ai-plan/design/governance/frontend/前端视觉设计规范.md`
  - `web/src/config/theme-workbench.ts`
- AI skills:
  - `$graft-multi-agent-batch`
  - `$graft-ai-plan-governance` when updating this topic's recovery material
  - `$graft-platform-architecture-review`
  - `$graft-consistency-review`
  - `$graft-delete-review`

Topic objective:

- Complete validation of the authority-first remediation that makes every `THEME_PRESET_DEFINITIONS` visual difference reproducible through writable Personalization Workbench state.

Work contract summary:

- Long-running audit topic with an authorized, completed remediation wave; current scope is validation and archive-readiness only.

Locked decisions:

1. Presets must materialize editable configuration and cannot remain runtime visual authority.
2. Runtime token resolution must consume editable state only; chart and Material tokens use shared editable definitions.

Current batch plan:

1. Run the project Vitest entrypoint for focused Vue SFC tests when additional coverage is needed; do not use Bun's native `bun test` runner for those files.
2. Rerun `bun run check`, then perform archive-readiness evaluation.

Validation expectations:

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
cd web && bun run check
```
