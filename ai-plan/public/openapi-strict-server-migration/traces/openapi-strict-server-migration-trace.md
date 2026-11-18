# OpenAPI Strict Server Migration Trace

## 2026-11-19 Batch 0: bootstrap and inventory

- Bootstrapped topic per ADR-003 contract-driven minimal bootstrap; Work Contract v1
  recorded in tracking.
- Inventoried 297 hand-registered route registrations across `server/modules`,
  `server/internal/httpx`, `server/internal/dashboard`, `server/internal/capability`,
  `server/internal/realtime`, `server/internal/mcp`, `server/internal/moduleruntime`,
  `server/internal/app` and 321 OpenAPI bundle operations.
- Classified strict candidates, non-strict endpoints (WebSocket shell, SSE/WS realtime,
  MCP streamable, healthz/docs) and core-owned route registrations outside module scope.
- Design decisions recorded in tracking batch 0: hybrid strict adapter writing envelopes
  through `httpx` for byte compatibility; generated `ServerInterfaceWrapper` mounted per
  route with existing group and fragment constants; per-route `RequirePermission`
  preserved; configurable binding error handlers for envelope compatibility.
- Next: batch 1 generation infrastructure.
