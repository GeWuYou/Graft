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
- Loop acceptance recorded the batch commit `ddbe3d1f` and advanced the current state to the next batch.

## Next Step

- Start `mcp-auth-streamable-foundation` with fresh startup preflight, then design the server-owned caller,
  authorization, confirmation-token, and streamable foundation without exposing tools prematurely.

## 2026-07-21 `mcp-auth-streamable-foundation`

- Added auth-owned expiring personal API Tokens with one-time secret delivery, SHA-256 persistence, owner-only
  management routes, revocation, last-used tracking, and live-row store semantics.
- Added the opt-in `/mcp` Streamable HTTP adapter. It consumes the verified auth caller, binds sessions to the token
  ID, runs RBAC before Token scope narrowing, and starts with no business tools, resources, or prompts.
- Added short-lived, caller/action/request-bound confirmation tokens that expire and become invalid after the first
  consume attempt; the next batch may consume these primitives but must not add a manual tool registry.
- Regenerated Ent, migration embedding, and OpenAPI projections. Validation passed: `go run ./cmd/graft validate backend`,
  `bun run check`, `just openapi-check`, and `python3 scripts/validate_sql_migrations.py`.
- Added the existing `.tmp/**` disposable test-output path to ESLint ignores after the web validation fixture was
  otherwise linted as application source; this keeps the required frontend entrypoint reproducible without changing
  product behavior.
- The OpenAPI generator emitted its known OpenAPI 3.1 support warning; freshness and contract checks still passed.

## Next Step

- Start `mcp-openapi-compiler-read-tools` with fresh startup preflight and compile opted-in, read-only tool
  descriptors from canonical OpenAPI metadata only.
