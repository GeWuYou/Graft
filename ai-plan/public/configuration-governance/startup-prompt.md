Continue the configuration-governance topic after completing root `AGENTS.md` startup preflight.

Round context:

- governance source: root `AGENTS.md`
- task class: `docs/automation with server impact`
- recovery source: `none`
- recovery entry: `ai-plan/public/configuration-governance/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `ai-plan/AGENTS.md`
- design authority:
  - `ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md`
  - `ai-plan/design/governance/platform/配置治理与迁移规范.md`
- AI skills:
  - `$graft-multi-agent-batch`
  - `$graft-validation-runner`

Topic objective:

- Establish versioned, source-aware environment and Compose configuration governance before migration or runtime initialization.

Work contract summary:

- Long-running platform feature; requires a repository design, ADR, roadmap, active topic, bounded batch execution, and eventual archive review.

Locked decisions:

1. Schema authority is embedded versioned YAML, not duplicated Go defaults or Compose templates.
2. First release blocks legacy/missing schema versions but only produces migration patches.

Implementation guardrails:

- Repair the schema authority before adding lower-layer compatibility.
- Keep production Compose validation daemon-free and read-only.
- Do not modify user configuration files or introduce runtime aliases.

Current batch plan:

1. Land design authority and Schema data model.
2. Implement resolver and CLI validation path.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the completed batch, authority owner, and validation evidence.
- Update the topic tracking and trace files with the next recovery point.
