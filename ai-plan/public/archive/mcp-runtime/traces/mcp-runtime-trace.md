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

## 2026-07-22 `mcp-openapi-compiler-read-tools`

- Added a startup-immutable compiler for the embedded canonical OpenAPI bundle. It accepts only explicit
  `x-graft-mcp` GET operations and derives each snake_case tool ID from `operationId` without tag, path, summary, or
  hand-written registry fallback.
- Added validation for the root metadata anchor, risk and confirmation declarations, positive ISO-8601 confirmation
  TTLs, canonical resource URI placeholder bindings, normalized tool-name collisions, and low-risk reversible read
  eligibility. Input schemas derive path, query, and JSON body inputs from the same operation.
- Added a generic in-process dispatcher that invokes the registered Gin REST route. The verified personal-token and
  request-auth contexts propagate into the route, preserving existing permission middleware and route-owned audit
  behavior; focused Streamable HTTP tests cover the full tool-call path.
- The embedded bundle currently projects `get_application`, `get_container`, and `get_runtime_target`. Tool lists are
  sorted deterministically and cannot change during the process lifetime; no Redis or new runtime cache is needed.
- Shared-asset review reused the existing OpenAPI contract bundle and HTTP runtime boundaries; no curated asset registry
  change is warranted. Comment-governance review accepted the added Chinese Go comments as responsibility or constraint
  documentation and found no stale or mechanical comments.
- Validation passed: `go test ./internal/mcp ./internal/httpx ./internal/app`, `go run ./cmd/graft validate backend`,
  `bun run check`, and `just openapi-check`. The OpenAPI generator retains its known OpenAPI 3.1 support warning;
  generated artifacts and projections are fresh.

## 2026-07-22 `mcp-compatibility-resources-actions`

- Extended the OpenAPI compiler into one capability projection: approved low-risk GET operations produce both Tools
  and canonical Resource Templates, while the opted-in high-risk `postContainerRestart` operation produces its
  normalized operationId Tool. No path/tag-derived identity or manual business registry was added.
- Registered `graft://docker/containers/{id}`, `graft://applications/{id}`, and
  `graft://runtime-targets/{id}` resource templates. Reads and confirmed actions re-enter the existing Gin routes,
  preserving existing Handler, Service, authorization, error, and audit ownership.
- Added server-enforced two-phase action confirmation. The first request issues a caller/action/request-fingerprint
  bound token; only a second request with that token reaches the REST route. Missing, expired, reused, or mismatched
  tokens remain rejected before any business side effect.
- Added an executable Phase 2.5 compatibility matrix that compares canonical REST/MCP success JSON, 403/404 errors,
  resource error data, permission-denial behavior, and audit evidence. Focused MCP, HTTP, and app tests passed.

## 2026-07-22 `mcp-hardening-stdio`

- Added bounded MCP runtime limits for request bytes, request duration, idle sessions, concurrent work, and total HTTP
  sessions. The values are deployment configuration, never OpenAPI capability metadata.
- Added adapter lifecycle admission shutdown, non-sensitive cumulative metrics, and structured invocation logs carrying
  request/trace correlation when an HTTP request provides it. Adapter errors remain at the transport boundary and do
  not alter canonical REST errors.
- Added `RunStdio`, which accepts a caller already authenticated by the existing personal API Token service and runs
  the same compiled `mcp.Server`, confirmation store, and in-process Gin route dispatcher as Streamable HTTP. It does
  not parse credentials or create a second authorization path.
- Added focused lifecycle and stdio protocol coverage. The stdio test proves `tools/list` and `tools/call` project the
  OpenAPI-derived capability and return the same REST JSON while retaining caller context. No browser, Compose, or
  temporary database was needed or started.

## 2026-07-22 Archive Readiness

- Accepted all five committed implementation batches after the required cross-boundary validation evidence was recorded.
- Confirmed the topic acceptance conditions: OpenAPI remains the sole capability authority, normalized `operationId`
  tool identity and canonical resource URIs are preserved, two-phase confirmation remains server-owned, Phase 2.5
  verifies REST/MCP JSON, error, permission, and audit parity, and HTTP/stdio share one compiled registry and Gin
  dispatcher.
- Moved the topic from the active recovery index to `ai-plan/public/archive/mcp-runtime/`. This archive is historical
  evidence; it is not a future startup or implementation path.
