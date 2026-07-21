# MCP Runtime Trace

## 2026-07-21 intake and `mcp-governance-and-contract`

- Completed the inherited cross-boundary startup receipt and confirmed no active topic owned the product MCP runtime.
- Work Intake classified the work as a long-running feature requiring an active topic, design authority, roadmap,
  ADR, and `graft-multi-agent-loop` execution.
- ADR-005 established OpenAPI operation metadata as the sole MCP capability contract. Product MCP is a runtime
  transport adapter and developer-local MCP governance remains separate.
- Added the extensible `x-graft-mcp` validation anchor and examples for container, application, and runtime-target
  resources. The examples use existing operations only and do not introduce business semantics or a manual tool list.
- Locked normalized `operationId` tool IDs, runtime-owned transport selection, canonical resource URI templates, and
  server-validated expiring two-phase confirmation for applicable actions.
- Added the mandatory Phase 2.5 compatibility matrix for canonical JSON, errors, permissions, and audit semantics.
- Validation passed: `git diff --check`, `python3 scripts/validate_ai_plan_structure.py`,
  `python3 scripts/validate_ai_governance.py`, and `just openapi-check`. The last check confirms the OpenAPI source,
  runtime-path projection, web generated schema freshness, and contract projection without modifying generated files.

## Next Step

- Start `mcp-auth-streamable-foundation` with fresh startup preflight, then design the server-owned caller,
  authorization, confirmation-token, and streamable foundation without exposing tools prematurely.
