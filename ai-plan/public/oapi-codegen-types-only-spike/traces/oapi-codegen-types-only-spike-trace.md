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
