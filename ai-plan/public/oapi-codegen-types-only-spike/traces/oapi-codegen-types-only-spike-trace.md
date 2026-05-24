# OAPI Codegen Types-Only Spike Trace

## 2026-05-24 topic bootstrap

- Replaced the old `feat/wt-openapi-contract-governance` dedicated pair with a new `oapi-codegen-types-only-spike` topic and pair so the active worktree/topic mapping stays aligned.
- Archived the completed `openapi-contract-governance` and `write-interface-error-contract-standardization` topics under `ai-plan/public/archive/`.
- Kept the accepted governance baseline unchanged:
  - no generated server interfaces
  - no runtime handler wiring changes
  - no `request.ts` replacement
  - no reopening of the broader write-route rollout
- Narrowed the new implementation goal to one isolated backend types-only spike under `server/internal/contract/openapi/**`.
- Added the initial isolated spike scaffold under `server/internal/contract/openapi/**`:
  - pinned `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`
  - explicit `go generate` entrypoint
  - checked-in generated types under `generated/**`
  - focused compile/test-only comparison coverage
- Validated the first scaffold with:
  - `cd server && go generate ./internal/contract/openapi`
  - `cd server && go test ./internal/contract/openapi/...`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
- Recorded the first material caveat from the tool itself:
  - generation succeeds, but `oapi-codegen` warns that OpenAPI `3.1.x` is not fully supported
  - generated request types are route-shaped, for example `PostUsersJSONRequestBody`, rather than runtime DTO replacements

## 2026-05-24 generated request body pure-write classification closeout

- Confirmed the current primary sample chain remains narrow and unchanged:
  - `generated request body -> handler thin binding/thin validation -> mapper -> command/service`
- Retained the isolated comparison-only sample set at:
  - `POST /api/users`
  - `POST /api/users/{id}/update`
  - `POST /api/users/{id}/status`
- Verified that the auth write routes do not directly join the current generated request body primary sample chain:
  - `POST /auth/login`
    - generated types exist in the isolated output, but runtime still binds handwritten `loginRequest`
    - the handler also performs `TrimSpace(username)` and explicit empty-field validation
  - `POST /auth/change-password`
    - runtime still binds handwritten `changePasswordRequest`
    - route and service semantics remain heavier than the thin-route sample definition
  - `POST /auth/complete-required-password-change`
    - runtime still binds handwritten `completeRequiredPasswordChangeRequest`
    - the current generated output does not expose a matching generated request body
    - recorded only as a future candidate for a separate handwritten-DTO pure-write subgroup
- Verified that `POST /api/users/{id}/reset-password` also stays outside the current primary sample chain because runtime still binds handwritten `resetUserPasswordRequest`.
- Removed the temporary `reset-password` alias/test additions from `server/internal/contract/openapi/types.go` and `server/internal/contract/openapi/spike_test.go` so the spike output matches the narrowed classification boundary.

## 2026-05-24 scope expansion to non-auth OpenAPI-covered interfaces

- Updated the topic tracking boundary to reflect the newly approved execution scope instead of the original isolated `server/internal/contract/openapi/**`-only spike.
- Kept the accepted baseline unchanged:
  - `server/internal/httpx` remains the only backend envelope owner
  - `web/src/utils/request.ts` remains the only frontend transport/runtime owner
  - this slice still does not introduce generated client runtime, server stubs, or strict-server output
- Recorded the new scope decision:
  - exclude all `auth` interfaces from the current slice
  - include the remaining non-`auth` interfaces already covered by `openapi/openapi.yaml`
  - allow this topic to extend into selected `server/plugins/{user,rbac}` and `web/src/modules/{user,rbac}` consumers for generated type adoption
- Recorded the current target interface set for the expanded slice:
  - `GET/POST /api/users`
  - `POST /api/users/{id}/update`
  - `POST /api/users/{id}/status`
  - `POST /api/users/{id}/reset-password`
  - `GET/POST /api/roles`
  - `POST /api/roles/{id}/update`
  - `POST /api/roles/{id}/permissions/assign`
  - `GET /api/permissions`
  - `GET /healthz`
- Noted that `reset-password` moved from the earlier excluded classification into the active non-`auth` migration scope.

