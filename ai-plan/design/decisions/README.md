# Decisions

This directory stores ADRs that lock design or governance decisions before wider retrofits.

## Use This Directory For

- one explicit decision per ADR
- decisions that later batches or validators must converge on

Current architecture decisions also include `ADR-004-task-runtime-state-machine.md`, which fixes the platform Task Runtime boundary before consumer implementation.
`ADR-005-mcp-runtime-contract-and-transport-boundary.md` fixes the product MCP runtime boundary before its adapter
and compiler batches begin.
`ADR-020-configuration-governance-schema.md` fixes the versioned deployment-configuration Schema authority before
runtime, Compose, template, and CI consumers converge.
`ADR-025-provider-oriented-project-layout.md` fixes the Provider-oriented project layout and legacy-frozen migration
boundary before new external capability implementations are added.

## Rules

- Keep ADRs as bounded decision records, not general design-doc catch-alls.
- Use the repository ADR naming pattern `ADR-XXX-short-title.md`.
- Put repository-wide baseline prose in sibling `design/` directories rather than duplicating it inside ADRs.
