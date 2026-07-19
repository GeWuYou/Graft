# Cross-Boundary Contract Projection Convergence

Continue work in the same `topic-completion-loop`.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: parent topic
- recovery entry: `ai-plan/public/cross-boundary-contract-projection-convergence/README.md`
- local execution truth: `server/AGENTS.md`, `web/AGENTS.md`, and `ai-plan/AGENTS.md` when topic materials change
- design authority: `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md` and `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- AI skills: `$graft-multi-agent-loop`, `$graft-task-closeout`, `$graft-commit`

Topic objective:

- Converge all remaining server-owned cross-boundary contracts and OpenAPI runtime API paths onto generated web artifacts.

Locked decisions:

1. OpenAPI owns HTTP paths, operations, wire schemas, and public wire enums.
2. Go server contracts own non-HTTP cross-boundary values; web has no hand-written authority mirror.

Current batch plan:

1. `inventory-and-openapi-runtime-path-projection`
2. `notification-project-runtime-target-task-migration`

Loop instructions:

- Default `loop_mode=topic-completion-loop`.
- Advance exactly one bounded batch this round.
- Update the tracking and trace documents in the same change.
- Run the smallest required validation before closeout.
- Evaluate `$graft-commit` only after validation and only for confirmable owned scope.
