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
- Representative resource and action metadata exists, but no MCP runtime, compiler, or transport implementation has
  been started.
- Next batch: `mcp-auth-streamable-foundation`.

## Task Checklist

- [x] `mcp-governance-and-contract`
- [ ] `mcp-auth-streamable-foundation`
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
    "mcp-governance-and-contract"
  ],
  "pending_batches": [
    "mcp-auth-streamable-foundation",
    "mcp-openapi-compiler-read-tools",
    "mcp-compatibility-resources-actions",
    "mcp-hardening-stdio"
  ],
  "current_batch": null,
  "next_batch": "mcp-auth-streamable-foundation",
  "closeout_status": "committed"
}
```
