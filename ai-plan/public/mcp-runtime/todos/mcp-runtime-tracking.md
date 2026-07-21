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
- Batch 3 compiles only opted-in low-risk GET operations from the embedded canonical OpenAPI bundle. It rejects invalid
  metadata, unsafe read declarations, unbound resource URI placeholders, and normalized name collisions. The resulting
  tool list is deterministic and startup-immutable; no manual business registry, resource, prompt, or action projection
  was introduced. The generic dispatcher re-enters the Gin REST route with the existing caller context so REST handler,
  authorization, and audit behavior remain authoritative.
- Next batch: `mcp-compatibility-resources-actions`.

## Task Checklist

- [x] `mcp-governance-and-contract`
- [x] `mcp-auth-streamable-foundation`
- [x] `mcp-openapi-compiler-read-tools`
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
    "mcp-auth-streamable-foundation",
    "mcp-openapi-compiler-read-tools"
  ],
  "pending_batches": [
    "mcp-compatibility-resources-actions",
    "mcp-hardening-stdio"
  ],
  "current_batch": null,
  "next_batch": "mcp-compatibility-resources-actions",
  "closeout_status": "committed"
}
```

## Latest Validation

- `cd server && go test ./internal/mcp ./internal/httpx ./internal/app`
- `cd server && go run ./cmd/graft validate backend`
- `cd web && bun run check`
- `just openapi-check`
