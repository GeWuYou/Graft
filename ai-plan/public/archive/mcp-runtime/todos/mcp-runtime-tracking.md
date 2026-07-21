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
  metadata, unsafe read declarations, unbound resource URI placeholders, and normalized name collisions.
- Batch 4 projects approved canonical resource templates for container, application, and runtime-target detail reads,
  and projects the opted-in high-risk container restart operation from its normalized `operationId`. The adapter issues
  a caller/action/request-bound two-phase token before dispatching the action through the existing Gin route. The
  executable Phase 2.5 matrix compares REST/MCP canonical JSON, error payloads, permission-denial behavior, and audit
  counts; resource errors retain the canonical REST payload in MCP error data.
- All implementation batches are committed and outer archive-readiness acceptance has passed.

## Task Checklist

- [x] `mcp-governance-and-contract`
- [x] `mcp-auth-streamable-foundation`
- [x] `mcp-openapi-compiler-read-tools`
- [x] `mcp-compatibility-resources-actions`
- [x] `mcp-hardening-stdio`

## Acceptance Conditions

- OpenAPI remains the sole MCP capability contract and no manual tool registry is introduced.
- Tool identity, resource URI, confirmation, transport, and compatibility decisions remain aligned with ADR-005.
- Phase 2.5 proves REST/MCP canonical JSON, error, permission, and audit equivalence before broad action exposure.
- HTTP and stdio share one compiled capability registry, one Gin dispatcher, bounded lifecycle, limits, error boundary, and correlation-safe observability.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "mcp-governance-and-contract",
    "mcp-auth-streamable-foundation",
    "mcp-openapi-compiler-read-tools",
    "mcp-compatibility-resources-actions",
    "mcp-hardening-stdio"
  ],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```

## Latest Validation

- `cd server && go test ./internal/mcp ./internal/httpx ./internal/app`
- `cd server && go test ./internal/config`
- `cd server && go run ./cmd/graft validate backend`
- `cd web && bun run check`
- `just openapi-check`
