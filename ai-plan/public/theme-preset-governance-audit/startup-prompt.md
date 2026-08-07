Continue the Theme Preset Governance Audit after rerunning the root startup preflight.

Round context:

- governance source: root `AGENTS.md`
- task class: `web`
- recovery source: `parent topic`
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
  - `$graft-platform-architecture-review`
  - `$graft-consistency-review`
  - `$graft-delete-review`

Topic objective:

- Produce the persisted Chinese audit proving whether every `THEME_PRESET_DEFINITIONS` visual difference can be recreated through writable Personalization Workbench state.

Work contract summary:

- Long-running audit topic with one bounded multi-agent evidence wave; no implementation, roadmap, or ADR is authorized.

Locked decisions:

1. Presets must materialize editable configuration and cannot remain runtime visual authority.
2. Recommendations must prefer editable configuration, semantic tokens, shared Material tokens, and derived rules.

Current batch plan:

1. Collect per-preset mapping, token reachability, Workbench control, and hidden-authority evidence.
2. Persist the audit matrix and governance recommendations; retain the topic for a separately authorized remediation decision.

Validation expectations:

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```
