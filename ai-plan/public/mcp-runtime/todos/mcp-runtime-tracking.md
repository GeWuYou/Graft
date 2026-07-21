# MCP Runtime Tracking

## Topic

MCP Runtime

## Scope

Build the product MCP runtime from canonical OpenAPI operations while retaining REST business semantics, server-owned
authorization, confirmation, error, and audit behavior.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/decisions/ADR-005-mcp-runtime-contract-and-transport-boundary.md`
- `openapi/openapi.yaml`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: OpenAPI and its path fragments are the sole MCP capability contract; MCP is a product runtime transport adapter, not a second business API or developer-tool policy.
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-loop
bootstrap:
  targets:
    - ai-plan/public/mcp-runtime
    - ai-plan/design/decisions/ADR-005-mcp-runtime-contract-and-transport-boundary.md
    - ai-plan/roadmap/MCP运行时实施计划.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Work Intake created the active topic and Batch 1 established the authority-level OpenAPI metadata contract.
- Batch 2 added auth-owned expiring personal API Tokens, a Streamable HTTP foundation, a shared caller context,
  RBAC-first scope narrowing, and server-validated single-use confirmation primitives. The foundation exposes no
  business tools, resources, or prompts.
- Next batch: `mcp-openapi-compiler-read-tools`.

## Task Checklist

- [x] `mcp-governance-and-contract`
- [x] `mcp-auth-streamable-foundation`
- [ ] `mcp-openapi-compiler-read-tools`
- [ ] `mcp-compatibility-resources-actions`
- [ ] `mcp-hardening-stdio`

## Acceptance Conditions

- OpenAPI remains the sole MCP capability contract and no manual tool registry is introduced.
- Tool identity, resource URI, confirmation, transport, and compatibility decisions remain aligned with ADR-005.
- Phase 2.5 proves REST/MCP canonical JSON, error, permission, and audit equivalence before broad action exposure.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "mcp-governance-and-contract",
    "mcp-auth-streamable-foundation"
  ],
  "pending_batches": [
    "mcp-openapi-compiler-read-tools",
    "mcp-compatibility-resources-actions",
    "mcp-hardening-stdio"
  ],
  "current_batch": null,
  "next_batch": "mcp-openapi-compiler-read-tools",
  "closeout_status": "committed"
}
```

## Latest Validation

- `cd server && go run ./cmd/graft validate backend`
- `cd web && bun run check`
- `just openapi-check`
- `python3 scripts/validate_sql_migrations.py`
- focused in-memory Streamable HTTP tests, including explicit session close
