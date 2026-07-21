Continue work inside the same `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: parent topic `mcp-runtime`
- recovery entry: `ai-plan/public/mcp-runtime/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/AGENTS.md`
- design authority:
  - `ai-plan/design/decisions/ADR-005-mcp-runtime-contract-and-transport-boundary.md`
  - `ai-plan/roadmap/MCP运行时实施计划.md`
  - `openapi/openapi.yaml`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-validation-runner`

Topic objective:

- Build the product MCP runtime as an OpenAPI-derived transport adapter with REST-equivalent semantics.

Work contract summary:

- feature / long-running / topic + design + roadmap + ADR / `graft-multi-agent-loop`.

Locked decisions:

1. Generate tool IDs from normalized `operationId` only and keep transport runtime-owned.
2. Use server-validated two-phase confirmation for applicable actions and test REST/MCP semantic compatibility.

Implementation guardrails:

- Rerun startup preflight before reading recovery state.
- Repair the highest available authority first and do not create hand-maintained tools or a second business API.
- Keep developer-local MCP policy separate from product runtime implementation.
- Consume the existing Work Contract; do not recreate intake artifacts during ordinary rounds.

Current batch plan:

1. `mcp-auth-streamable-foundation`
2. `mcp-openapi-compiler-read-tools`
3. `mcp-compatibility-resources-actions`
4. `mcp-hardening-stdio`

Loop instructions:

- Default `loop_mode=topic-completion-loop`.
- Advance exactly one bounded batch this round.
- Update the topic tracking and trace files in the same change.
- Run the smallest required validation before closeout.
- Evaluate `$graft-commit` only after validation and only for confirmable owned scope.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the current batch and its authority owner.
- State the REST/MCP compatibility impact or why it is not yet applicable.
- Return the loop JSON closeout with the next pending batch.
