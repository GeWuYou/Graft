# OAPI Codegen Types-Only Spike Plan

## Objective

Evaluate whether `oapi-codegen` can generate useful Go contract types from the existing root OpenAPI spec without changing runtime ownership, plugin boundaries, or the current handwritten DTO baseline.

## Fixed Decisions

- Use the existing root spec at `openapi/openapi.yaml`.
- Generate Go types only.
- Do not generate server interfaces, strict-server stubs, or runtime clients.
- Keep all generated output under `server/internal/contract/openapi/generated/**`.
- Keep any handwritten glue or comparison helpers under `server/internal/contract/openapi/**` only.
- Do not replace or rewire `server/plugins/user/dto_http_request.go`, `server/plugins/rbac/dto_http_request.go`, or `server/internal/httpx/**`.

## Deliverables

- Pinned `oapi-codegen` tool dependency in `server/go.mod`.
- Explicit `go generate` entrypoint plus checked-in config under `server/internal/contract/openapi/**`.
- Checked-in generated Go types for the shared envelope and the `POST /api/users` sample coverage from the root spec.
- Focused compile/test-only comparison proving the generated package can be consumed without entering runtime code.
- Final go/no-go recommendation recorded in this topic's tracking/trace.
- Early spike findings should be recorded explicitly:
  - `oapi-codegen` currently emits a warning for the repository's OpenAPI 3.1 root spec even when model generation succeeds.
  - request payload types may be emitted under route-shaped names such as `PostUsersJSONRequestBody`, not under the handwritten DTO names used by runtime code.

## Validation

- `cd server && go generate ./internal/contract/openapi`
- `cd server && go test ./internal/contract/openapi/...`
- `cd server && go run ./cmd/graft validate backend --stage openapi`
- `cd server && go run ./cmd/graft validate backend`

## Non-Goals

- No `openapi/**` edits in this topic unless a validation blocker proves the current root spec is not consumable by `oapi-codegen`.
- No plugin runtime rewiring.
- No `web` runtime changes.
- No widening from the minimal isolated spike into a multi-route migration plan.
