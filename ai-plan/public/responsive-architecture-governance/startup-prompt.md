Continue the `responsive-architecture-governance` topic in `topic-completion-loop` mode unless the caller explicitly changes it.

Round context:

- governance source: root `AGENTS.md`
- task class: `web + docs/automation`
- recovery source: `none`
- recovery entry: `ai-plan/public/responsive-architecture-governance/README.md`
- local execution truth:
  - `web/AGENTS.md`
  - `ai-plan/AGENTS.md` when touching `ai-plan/**`
- design authority:
  - `ai-plan/design/governance/frontend/Graft响应式架构治理规范.md`
  - `ai-plan/design/governance/frontend/前端视觉设计规范.md`
  - `ai-plan/roadmap/Graft响应式架构迁移路线.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-task`
  - `$graft-validation-runner`

Topic objective:

- Establish the shared, container-first responsive architecture so new Graft pages are Desktop-first and Mobile Friendly without device-specific business branches.

Work contract summary:

- `refactor`, `long-running`; requires design/topic/roadmap, no ADR; engine and dispatch skill are `graft-multi-agent-loop`; closeout requires lessons review.

Locked decisions:

1. `web/src/shared/**` and `web/src/style/**` own runtime responsive capability; business pages only consume semantic variants.
2. Governance manifests and debt/exception records do not enter the production runtime bundle.

Implementation guardrails:

- Repair the highest available authority first.
- Keep each batch bounded and update tracking/trace in the same change.
- Do not introduce Mobile pages, device booleans, page-local breakpoints, or a second UI framework.
- Consume the existing Work Contract; do not recreate intake artifacts during ordinary rounds.

Current batch plan:

1. `B2-foundation-runtime`: implement shared responsive tokens, container/variant infrastructure, and shell authority repairs.
2. `B3-responsive-primitives`: converge shared Responsive components and governance records.

Validation expectations:

```bash
bun run check
git diff --check
```

Required closeout:

- State current batch, authority owner, validation and scoped commit evidence.
- Update loop batch state.
- Use `Next-session startup prompt:` only for terminal states.
