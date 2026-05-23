# Write-Interface Error Contract Standardization Trace

## 2026-05-23 `POST /api/users` sample closure in current worktree

- Bound this topic to the existing `feat/wt-openapi-contract-governance` worktree instead of creating a new worktree.
- Kept the accepted baseline unchanged:
  - `spec-first + TS-first + explicit server DTOs`
  - `web/src/utils/request.ts` remains the only frontend runtime transport truth
  - `server/internal/httpx` remains the backend envelope and localized-error owner
  - no `oapi-codegen`, `openapi-fetch`, TS runtime SDK, or generated Go server runtime wiring
- Standardized the `POST /api/users` sample decision that `data.field` uses the current request-contract field name.
- Narrowed the create-user password-policy error field from `new_password` to `password` at the user-plugin create route boundary only.
- Kept the mapping out of `httpx` and out of `request.ts`.
- Added the matching OpenAPI `400` examples for invalid argument and password-policy violation.
- Kept the follow-up conclusion unchanged: `oapi-codegen` is still deferred, and a future Go types-only spike is still blocked on broader write-interface hardening beyond this one sample.
