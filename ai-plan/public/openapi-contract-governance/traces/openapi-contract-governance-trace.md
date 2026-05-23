# OpenAPI Contract Governance Trace

## 2026-05-23

- Created dedicated worktree `feat/wt-openapi-contract-governance`.
- Established public recovery topic `openapi-contract-governance`.
- Captured Phase 1 planning scope for OpenAPI First governance.
- Kept implementation untouched.
- Completed Phase 1.5 server boundary review for OpenAPI follow-up planning.
- Completed Phase 1.6 same-package lightweight file reorganization for `server/plugins/user` and `server/plugins/rbac`.
- Kept `package user` / `package rbac` unchanged and did not introduce subpackages, Go generated models, or OpenAPI files.
- Recorded Phase 1.6 as preparation for a future package-boundary refactor, not as the final server directory architecture.
- Completed Phase 2A minimal OpenAPI First baseline.
- Added root `openapi/` spec, path fragments, reusable schemas, security docs, and common error responses.
- Preserved the actual `/healthz` plain JSON contract instead of forcing it into the success envelope.
- Wired OpenAPI validation into `graft validate openapi` and `graft validate backend --stage openapi`, and inserted it at
  the front of the full backend validation chain.
- Kept `server/plugins/user` and `server/plugins/rbac` business logic unchanged.
- Kept `web/src/utils/request.ts` untouched and did not start generated TypeScript runtime consumption.
- Audited the existing Phase 2A partial diff and confirmed the owned scope stayed within `openapi/**`, backend OpenAPI
  validation wiring, and topic recovery docs.
- Repaired the root spec after `kin-openapi` rejected `info.summary` for the current validation path.
- Validated the repaired slice with `go run ./cmd/graft validate openapi`, `go run ./cmd/graft validate backend --stage openapi`,
  focused `go test ./internal/cli`, and `go build ./cmd/graft`.
- Completed Phase 2B minimal web TypeScript generation wiring with `openapi-typescript`.
- Added tracked generated output at `web/src/contracts/openapi/generated/schema.ts`.
- Added `web` scripts for generation and freshness checking without changing `request.ts` or consuming generated types in module APIs.
- Confirmed the generated file must be formatted after generation to satisfy the existing frontend Prettier gate.
- Validated Phase 2B with `bun run openapi:types`, `bun run openapi:types:check`, `bun run check`, and `go run ./cmd/graft validate backend --stage openapi`.
- Completed Phase 2C minimal generated TypeScript consumption pilot for `GET /api/permissions`.
- Added a thin `rbac` module type alias layer that re-exports `PermissionListItem` and `PermissionListResponse` from `web/src/contracts/openapi/generated/schema.ts`.
- Switched `web/src/modules/rbac/api/rbac.ts` and the two `rbac` permission-consuming pages to the module-local generated permission DTO aliases.
- Kept `web/src/utils/request.ts`, request payload typings, `openapi/**`, and `web/src/contracts/openapi/generated/schema.ts` unchanged.
- Closed the remaining Phase 2C validation blocker by regenerating `web/src/contracts/openapi/generated/schema.ts` from the current root spec so `openapi:types:check` and the full `bun run check` path agree on generated type freshness.
- Expanded Phase 2C from the initial permission-list pilot to the full currently-covered frontend response DTO surface:
  `auth/login`, `auth/refresh`, `auth/bootstrap`, `users`, `roles`, and `permissions`.
- Replaced the handwritten auth/bootstrap response DTO definitions in `web/src/api/model/authModel.ts` with local aliases over generated OpenAPI schema components.
- Replaced the handwritten user-list response DTO definitions in `web/src/modules/user/types/user.ts`, while keeping `email` and `last_login_at` as page-local compatibility fields only.
- Replaced the handwritten role-list response DTO definitions in `web/src/modules/rbac/contract/role.ts`, while keeping `remark` fallback compatibility page-local instead of weakening the generated alias.
- Fixed `web/package.json` `openapi:types:check` so the temporary generated file is formatted under repo-local `.tmp/` with the project Prettier config, avoiding false drift from `.prettierignore` on `/tmp/tmp*`.
- Started Phase 2E request payload generated type consumption with a single RBAC pilot for `POST /api/roles/{id}/permissions/assign`.
- Added a reusable OpenAPI request schema for `permission_ids` replace semantics and wired it into the new RBAC write-path fragment.
- Regenerated `web/src/contracts/openapi/generated/schema.ts` so the generated output now includes `ReplaceRolePermissionsRequest` and the `postRolePermissionAssign` requestBody contract.
- Switched `web/src/modules/rbac/types/rbac.ts` to a thin generated alias for `ReplaceRolePermissionsPayload`.
- Kept the role-permission drawer's local selection state and added a narrow mapper so generated payload types do not become page form state.
- Validated the Phase 2E pilot with `bun run openapi:types`, `go run ./cmd/graft validate backend --stage openapi`, `bun run openapi:types:check`, and `bun run check`.
- Added a second narrow request-payload sample for `POST /api/users`, covering the create-user request body and success response in the root OpenAPI spec while keeping the existing runtime envelope semantics unchanged.
- Switched `web/src/modules/user/types/user.ts` create payload typing to the generated `CreateUserRequest` alias instead of the handwritten local interface.
- Tightened the `server/plugins/user` create-user failure path with focused tests for `AUTH_PASSWORD_POLICY_VIOLATION` and structured logs that preserve `operation`, `route`, `message_key`, `response_code`, and raw error details without leaking the submitted password.
- Updated the `web` user-create drawer so API failures surface the backend-provided message instead of collapsing to the generic "create failed" copy.
- Added inline password-policy guidance plus weak/medium/strong strength feedback to the create-user form, aligned to the current server rule: minimum 12 characters, letters plus digits, and no reuse of `graft-admin`.
- Validated the create-user sample with `bun run openapi:types`, `bun run openapi:types:check`, `bun run test:run src/modules/user/pages/index.test.ts`, `go test ./plugins/user/...`, and `go run ./cmd/graft validate backend --stage openapi`.
