# MCP Runtime 实施计划

## 目标

在不复制业务 API 的前提下，为已有 OpenAPI operation 建立产品 MCP runtime。OpenAPI 是 capability contract；MCP
adapter 复用 canonical JSON、错误、permission 与 audit 语义。开发者本机 MCP 工具不属于本计划的 runtime。

## 已锁定前提

- `x-graft-mcp` 是一个 extensible OpenAPI object，anchor 为 `#/x-graft-mcp-schema`。
- tool ID 仅由 `operationId` 正规化为 snake_case；不从 tag、path 或 summary 推导。
- `risk` 至少包含 `level`；`confirmation` 至少包含 `required`、`strategy`、`ttl`，适用 action 使用 two-phase
  confirmation。
- resource URI 模板固定使用 `graft://docker/containers/{id}`、`graft://applications/{id}`、
  `graft://runtime-targets/{id}` 约定。
- transport 属于 runtime，不能进入 tools 或 `x-graft-mcp` metadata。

## Batch Plan

### 1. `mcp-governance-and-contract` (complete)

- Persist Work Intake, active topic, ADR-005, this roadmap, and the developer-tool/runtime governance distinction.
- Add the reusable OpenAPI anchor and representative metadata examples only; do not alter business operation behavior
  or generated artifacts.

Acceptance: canonical design and OpenAPI metadata explain the compiler input without claiming runtime tools exist.

### 2. `mcp-auth-streamable-foundation`

- Add server-owned caller/authentication context, permission bridge, bounded confirmation token lifecycle, and the
  streamable runtime foundation.
- Preserve current REST middleware and audit ownership; do not expose tools yet.

Acceptance: a product MCP request resolves the same actor and authorization context as REST, and confirmation tokens
are server-validated, expiring, and single-purpose.

### 3. `mcp-openapi-compiler-read-tools`

- Implement the compiler from opted-in OpenAPI operations to read-only tool descriptors.
- Validate extension shape, URI templates and explicit placeholder bindings, required `operationId`, normalization
  collisions, and absence of transport selection in metadata.

Acceptance: read tools are generated from canonical OpenAPI only; there is no tag/path fallback or manual registry.

### 4. `mcp-compatibility-resources-actions`

- Add resource projection and action exposure for the approved container, application, and runtime-target families.
- Enforce risk metadata and two-phase confirmation before applicable action execution.

Acceptance: resource URIs and action descriptors resolve through existing business contracts without new business API
semantics.

### 2.5. REST/MCP Compatibility Matrix (mandatory gate)

Run this gate after the compiler is available and before broad actions or a production transport are considered:

| Case | REST baseline | MCP projection | Required assertion |
| --- | --- | --- | --- |
| Read tool/resource success | existing operation response | tool/resource result | canonical JSON payload is equivalent after transport framing is removed |
| Validation or not-found failure | existing status and error envelope | MCP error result | canonical error code, message key, and detail semantics match |
| Permission denial | existing authorization path | same caller and target | both transports deny the same actor/resource combination without information leak |
| Read audit | existing audit policy | MCP adapter invocation | audit action, actor, resource, result, request correlation, and outcome remain equivalent |
| Confirmed action | existing action path | two-phase confirmed action | canonical JSON, permission checks, side effect, and success audit remain equivalent |
| Missing, expired, or reused confirmation | REST-side guarded baseline | MCP action request | action is rejected with the canonical error semantics and rejected audit evidence |

The test harness must compare canonical JSON, error, permission, and audit semantics explicitly. Transport-specific
envelope fields may differ only when documented by the adapter; their presence must not change the canonical payload.

### 5. `mcp-hardening-stdio` (complete)

- Choose and harden the first runtime transport after the prior batches pass, beginning with stdio only if the runtime
  decision confirms it.
- Add lifecycle, shutdown, limits, observability, abuse controls, and transport-specific conformance coverage.

Acceptance: transport hardening cannot bypass the compiler, confirmation, permission, audit, or compatibility gate.

Delivered: Streamable HTTP and stdio use the same OpenAPI-compiled server and Gin dispatcher. Runtime-owned deployment
limits bound request size, request lifetime, sessions, and concurrency; lifecycle closure stops admission and server
shutdown cancels residual connections. Adapter metrics and structured correlation-safe logs contain no tool arguments,
tokens, or response bodies.

## Non-goals

- No second business API, hand-written tool manifest, tag/path-derived tool identity, or client-owned confirmation.
- No product dependency on developer-local Codex MCP configuration, AI skills, or AI tool availability.
- No generated OpenAPI artifact update during Batch 1 unless a later implementation batch changes a generated contract
surface.
