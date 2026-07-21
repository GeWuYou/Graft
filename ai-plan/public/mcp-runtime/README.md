# MCP Runtime

## Current Status Summary

- Topic objective: 在 OpenAPI authority 上建立产品 MCP runtime，并保持 REST 与 MCP 业务语义一致。
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: 这是需要 design、roadmap、ADR、稳定恢复与多 batch 执行的长期 feature。
- Canonical authority:
  - `openapi/openapi.yaml` and referenced path fragments
  - `ai-plan/design/decisions/ADR-005-mcp-runtime-contract-and-transport-boundary.md`
  - `ai-plan/roadmap/MCP运行时实施计划.md`
- Completed so far: `mcp-governance-and-contract`, `mcp-auth-streamable-foundation`, `mcp-openapi-compiler-read-tools`
- Not started yet: resource/action projection and transport hardening.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: parent topic `mcp-runtime`
- authority summary: OpenAPI operation metadata is the sole MCP capability contract; product MCP is a runtime adapter,
  not a developer-tool MCP policy or a second business API.

## Owned Scope

- `ai-plan/public/mcp-runtime/**`
- MCP ADR, roadmap, AI/MCP governance, and contract-governance rules
- `openapi/**` MCP metadata anchors and examples
- Batch 2 auth-owned personal API Token schema, migration, lifecycle routes, and stable capability
- Batch 2 MCP Streamable HTTP adapter, caller context, RBAC-plus-scope bridge, and confirmation-token foundation
- Batch 3 OpenAPI read-tool compiler, Gin in-process dispatcher, deterministic bundled-spec integration, and focused compatibility evidence
- Derived server and web OpenAPI artifacts for the Batch 2 wire contract

Out of scope:

- Hand-written tool definitions, tag/path-derived tool identity, and transport selection in OpenAPI metadata
- Resource/action projection, REST/MCP action compatibility execution, stdio, and transport hardening
- developer-local MCP client configuration, AI skills, and unrelated active topics

## Locked Decisions

1. Tool IDs are normalized from `operationId` only; runtime owns transport selection.
2. High-risk applicable actions use server-validated, expiring two-phase confirmation.
3. REST/MCP compatibility testing must compare canonical JSON, errors, permissions, and audit semantics.

## Phase Plan

- `mcp-auth-streamable-foundation`
- `mcp-openapi-compiler-read-tools`
- `mcp-compatibility-resources-actions`
- `mcp-hardening-stdio`

## Current Recovery Point

- Batch 1 recorded the canonical metadata, URI conventions, compatibility gate, and runtime/developer-tool boundary.
- Batch 2 added auth-owned personal API Tokens, an opt-in Streamable HTTP adapter with no business capabilities,
  server-validated confirmation primitives, and OpenAPI-derived request/response artifacts.
- Batch 3 compiles only `x-graft-mcp` opted-in low-risk GET operations from the embedded canonical OpenAPI bundle.
  Tool IDs derive solely from normalized `operationId`; schemas, resource bindings, risk, and confirmation metadata are
  validated before registration. The dispatcher invokes the same Gin REST route with the verified caller context, so
  existing permission checks and audit behavior remain the execution authority.
- Next step: `mcp-compatibility-resources-actions`.

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full `Work Contract` in the tracking file, not here.
- Use this README for navigation, summary, and recovery entry only.

## Pending Batch Direction

- `mcp-compatibility-resources-actions`
- `mcp-hardening-stdio`

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
python3 scripts/validate_ai_governance.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/mcp-runtime/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
