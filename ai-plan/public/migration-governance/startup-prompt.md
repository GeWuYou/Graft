Continue the Migration Governance topic.

- governance source: root `AGENTS.md`
- task class: `docs/automation`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/migration-governance/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `ai-plan/AGENTS.md`
- design authority:
  - `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`
- AI skills:
  - `$graft-multi-agent-batch`
  - `$graft-validation-runner`

Topic objective:

- Complete Phase 1 SQL Schema Governance and Migration Contract Gate without rewriting historical migrations.

Locked decisions:

1. Use PostgreSQL catalog state for schema semantics.
2. Keep destructive changes review-only until a PostgreSQL-aware automated analyzer is available.

Current batch plan:

1. Implement catalog check and bootstrap integration.
2. Wire the validated gate into Justfile and PR CI.

Required closeout:

- Update the topic tracking and trace files.
- State validation evidence and any remaining Phase 1 blocker.
- Use `Next-session startup prompt:` only for terminal handoff.
