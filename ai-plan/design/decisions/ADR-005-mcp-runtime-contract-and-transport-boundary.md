# ADR-005: MCP Runtime Contract And Transport Boundary

- Status: accepted
- Date: 2026-07-21
- Scope: product MCP runtime, `openapi/**`, runtime adapter batches

## Context

Graft needs a product-facing MCP runtime for existing administration capabilities. Existing developer-local MCP
servers remain useful for AI exploration, but they cannot own product behavior, user authorization, or the public
business contract. A separate hand-written tool surface would duplicate REST semantics and eventually drift from
OpenAPI, permissions, errors, and audit logging.

## Decision

1. Product MCP is a `server` runtime transport adapter, not an AI development tool and not a second business API.
   It invokes the existing operation semantics and shares their permission, error, and audit behavior.
2. `openapi/openapi.yaml` and referenced path fragments are the sole MCP capability contract. `x-graft-mcp` is one
   operation-level extensible object whose validation anchor is `#/x-graft-mcp-schema`.
   A later compiler reads this metadata; no manually maintained tool registry is allowed.
3. A tool identifier is the normalized snake_case form of `operationId` only. The compiler must require an
   `operationId`, use no tag/path/summary fallback, and reject normalized-name collisions. For example,
   `getContainer` becomes `get_container` and `postContainerRestart` becomes `post_container_restart`.
4. `x-graft-mcp.risk` requires `level` and reserves `reason`, `reversible`, and `impact`. `confirmation` requires
   `required`, `strategy`, and `ttl`; applicable actions use `strategy: two_phase` with an expiring confirmation
   token. A client-side confirmation prompt alone is not authorization evidence.
5. Resource URI templates are canonical metadata: `graft://docker/containers/{id}`,
   `graft://applications/{id}`, and `graft://runtime-targets/{id}`. Runtime transport selection is owned by the
   runtime configuration and adapter, never by a tool or OpenAPI metadata field.
6. Phase 2.5 is mandatory before broad resource/action exposure. It compares REST and MCP behavior for canonical
   JSON, errors, permission denials, and audit events. The same underlying business contract must be observable on
   both transports.

## Validation Rules

The later OpenAPI compiler must validate every opted-in operation against the anchor and these semantic rules:

- `x-graft-mcp` is an object; unknown fields are allowed for forward extension but must be documented before use.
- each resource URI template starts with `graft://`, uses the canonical family, and every URI placeholder has an
  explicit `resource_uri_parameter_bindings` entry whose target is a path parameter declared by the operation;
- `confirmation.required=false` requires `strategy=none` and `ttl=PT0S`;
- `confirmation.required=true` requires `strategy=two_phase` and a positive ISO 8601 duration;
- high- and critical-risk actions must require confirmation; and
- a metadata example may not introduce a request, response, permission, error, or audit semantic that OpenAPI does
  not already own.

## Consequences

- Product runtime implementation may add a bounded server-side MCP adapter in later batches. The developer-local MCP
  dependency prohibition does not apply to that product adapter.
- REST remains the canonical external contract; MCP is a transport projection whose compatibility must be tested, not
  assumed.
- This ADR does not select stdio, HTTP, or any other transport. The runtime chooses transport after the shared
  compiler, authorization, and parity foundation exists.
