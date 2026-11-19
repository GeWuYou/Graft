# OpenAPI Strict Server Migration

## Goal

Migrate Graft's backend from generated types plus hand-wired gin routes to oapi-codegen
strict server interfaces, so route-to-contract mismatch becomes a compile error for all
business modules under `server/modules/`. HTTP wire semantics, response envelopes,
permission behavior and error mapping stay unchanged.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: cross-boundary
- recovery source: this topic
- authority summary: OpenAPI is the shared wire authority; `server/internal/contract/openapi`
  is the derived consumption surface; `internal/httpx` owns the response envelope and
  authorization middleware; `server/modules/<name>/contract` owns route and permission
  constants.

## Owned Scope

- `server/internal/contract/openapi/**`
- `server/modules/*/route_registration.go` and adjacent route/handler files per module
- `server/modules/*/contract/**`
- `scripts/openapi_generated_freshness_check.py`
- `ai-plan/public/openapi-strict-server-migration/**`
- generated strict-server outputs and snapshots owned by the above

Out of scope for this topic: `openapi/paths/**` and `openapi/components/**` semantic
edits, `web/**`, `Justfile`, `.github/workflows/**`, core route owners under
`server/internal/**` (access log explorer, agent transport, dashboard, module runtime,
capability, realtime, mcp, healthz).

## Design Summary

- Per-domain oapi-codegen configs generate `models + gin-server + strict-server` into the
  existing per-domain packages under `server/internal/contract/openapi/<domain>/`.
- Modules implement the generated `StrictServerInterface` through a thin adapter. Handler
  bodies keep using `httpx.WriteSuccess` / `httpx.AbortLocalizedError` / `AbortAppError`
  so envelopes stay byte-compatible; strict response objects are not serialized because
  the generated envelope field order and trailing newline would change bytes.
- Route registration keeps module group and fragment constants and mounts the generated
  `ServerInterfaceWrapper` methods per route. `RequirePermission` stays a per-route
  middleware with the module contract permission constants, preserving middleware order.
- Public, optional-auth and WebSocket endpoints keep explicit handling.
- Freshness discovers all per-domain outputs from their Go generation directives.
- Route coverage compares each business operation and HTTP method with its unique generated mount in module source. Existing module tests exercise Gin registration and behavior.

## Pending Directions

- The 19 route-owning modules mount 295 generated strict operations. The other three business modules expose services only.
- Core HTTP routes and the container Shell WebSocket retain their own registration.
- The audit deletion URL mismatch remains a separately owned contract decision. The existing live fragment now mounts the generated operation without changing its URL or permission.
