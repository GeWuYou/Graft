Continue the Runtime Targets Infrastructure IA topic in `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/runtime-targets-infrastructure-ia/README.md`
- local execution truth:
  - `ai-plan/AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
- design authority:
  - `ai-plan/design/architecture/导航与资源路由信息架构规范.md`
  - `ai-plan/design/architecture/运行目标与基础设施信息架构设计.md`
  - `ai-plan/design/domains/container/容器管理设计.md`
  - `ai-plan/design/domains/compose/Compose项目管理设计.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-task`

Topic objective:

- Add Runtime Targets and Provider-scoped Infrastructure resources while preserving the existing sidebar navigation mode and Application/Compose lifecycle authority.

Work contract summary:

- long-running feature; design, roadmap, and active topic required; no ADR; execution through `$graft-multi-agent-loop` and `$graft-multi-agent-task`.

Locked decisions:

1. Section Labels are not menu nodes and do not create routes, permissions, breadcrumbs, Tabs, or quick actions.
2. Runtime Targets own connections; Provider pages consume targets; Docker plaintext TCP is rejected.
3. Compose remains Application/Project lifecycle authority; Container filtering uses `deployment_type` and `runtime_target`, not “source”.

Implementation guardrails:

- Repair the highest available authority first.
- Preserve the normal sidebar; do not introduce Workspace Mode.
- Do not render Kubernetes or Podman until a matching Target and actual capability exist.
- Consume the existing Work Contract; do not recreate topic, design, roadmap, or ADR artifacts inside ordinary execution rounds.

Current batch plan:

1. `menu-section-contract-and-sidebar-rendering`
2. `runtime-target-foundation`

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the current batch, authority owner, validation, and updated loop batch state.
- Use `Next-session startup prompt:` only for terminal states.
