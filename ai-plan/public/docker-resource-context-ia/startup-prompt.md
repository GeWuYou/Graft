Continue the Docker Resource Context IA topic through cross-boundary validation, browser evidence, and closeout.

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/docker-resource-context-ia/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `web/AGENTS.md`
- design authority:
  - `ai-plan/design/domains/container/容器管理设计.md`
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- AI skills:
  - `$graft-multi-agent-batch`
  - `$graft-validation-runner`

Topic objective:

- Make Network and Volume context-first Docker resources using canonical OpenAPI contracts and the shared reading model `Overview -> Context -> Relations -> Configuration -> Metadata -> Danger Zone`.

Work contract summary:

- Long-running cross-boundary refactor; Context and relationship trust are server-owned, while web presents the canonical contract.

Locked decisions:

1. Context is business information and may not be reconstructed from metadata in the web layer.
2. Metadata is diagnostic-only, default-collapsed, and may not drive navigation or business judgment.

Completed batches:

- Context contract, relationship-trust projection, and the Docker resource design guideline.
- Network and Volume contract, server projection, and Web IA implementation.

Current batch:

- Run cross-boundary validation, collect browser evidence, and close out the topic.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the cross-boundary validation and browser evidence status.
- State validation run and any deliberate gaps.
- Update topic tracking and trace before handing off.
