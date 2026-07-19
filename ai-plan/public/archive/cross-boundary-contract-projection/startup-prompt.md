Continue work inside the same `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/cross-boundary-contract-projection/README.md`
- local execution truth: `server/AGENTS.md`, `web/AGENTS.md`, and `ai-plan/AGENTS.md` when their paths are touched
- design authority:
  - `ai-plan/design/governance/platform/跨边界契约投影设计.md`
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-validation-runner`

Topic objective:

- Implement authority-aware projection from existing OpenAPI and Go server contracts to generated web artifacts without creating a second contract authority.

Work contract summary:

- long-running refactor; topic, design and roadmap are required; the execution engine is `$graft-multi-agent-loop`.

Locked decisions:

1. OpenAPI owns wire contract; Go server contracts own non-HTTP values; web consumes generated artifacts only.
2. Export metadata references existing Go constants and `visibility=web` controls bundle exposure.
3. Error/message values remain open strings with fallback; server runtime remains authority for menu, permission and capability availability.

Implementation guardrails:

- Repair the highest available authority first.
- Keep generator output derived and deterministic; do not introduce protobuf, a shared runtime package, or a new hand-written IDL.
- Do not add future-batch artifacts early.

Current batch plan:

1. `generator-foundation`
2. `pilot-migration`
3. `ci-integration`
4. `broader-migration-and-final-archive-readiness`

Validation expectations:

```bash
git diff --check
```

Add the smallest server/web/OpenAPI validation required by the files changed in the current batch.
