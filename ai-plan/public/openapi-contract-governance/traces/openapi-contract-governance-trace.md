# OpenAPI Contract Governance Trace

## 2026-05-23 topic closeout

- Marked `openapi-contract-governance` as completed, archived in place, and inactive without moving the topic directory.
- Removed the topic from `ai-plan/public/README.md` active recovery entries and kept only a brief historical pointer.
- Preserved the accepted final decisions in place:
  - Phase 3 stays on generated schema types + module-local alias layers + existing `request.ts`.
  - Phase 4 stays deferred/no-go for `oapi-codegen`; keep `spec-first + TS-first + explicit server DTOs`.
- Recorded the final validation evidence for the completed topic:
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run check`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd server && go run ./cmd/graft validate backend`
- Recorded the final closeout commits:
  - `3d7a16a docs(openapi-contract-governance): close phase 3 evaluation`
  - `3765d6a docs(openapi-contract-governance): close phase 4 evaluation`
- Declared that there is no next-session startup prompt for continuing this topic.
- Added the deferred follow-up evaluation and next-topic recommendation at
  `ai-plan/public/openapi-contract-governance/traces/oapi-codegen-followup-evaluation.md`.

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
- Extended the same bounded request-payload rollout to `POST /api/auth/login`, which was already covered in the OpenAPI spec as `LoginRequest`.
- Replaced the handwritten `LoginPayload` in `web/src/api/model/authModel.ts` with a thin alias over generated `components['schemas']['LoginRequest']`.
- Kept the login page form state local in `web/src/app/auth/components/Login.vue` and preserved the explicit `account -> username` submit-time mapping inside `web/src/store/modules/user.ts`.
- Validated the auth/user payload alias slice with `go run ./cmd/graft validate backend --stage openapi`, `bun run openapi:types`, `bun run openapi:types:check`, and `bun run check`.
- Extended the bounded request-payload rollout to the existing RBAC role write paths `POST /api/roles` and `POST /api/roles/{id}/update`.
- Added reusable root OpenAPI schemas for `CreateRoleRequest`, `UpdateRoleRequest`, and `EnvelopedRoleItemResponse`, and wired the new role update path fragment into `openapi/openapi.yaml`.
- Replaced the handwritten `CreateRolePayload` and `UpdateRolePayload` in `web/src/modules/rbac/types/rbac.ts` with thin aliases over generated schema components.
- Kept the role drawer's local form state in `web/src/modules/rbac/pages/index.vue` and moved the generated payload shaping to narrow submit-time mappers.
- Validated the RBAC role write-path slice with `go run ./cmd/graft validate backend --stage openapi` and `bun run check`.
- Started the next user write-path payload slice for `POST /api/users/{id}/update`, `POST /api/users/{id}/status`, and `POST /api/users/{id}/reset-password`.
- Added root OpenAPI request schemas and path fragments for the user update, status update, and password reset write paths, keeping the existing server DTO shapes unchanged.
- Switched `web/src/modules/user/types/user.ts` to thin generated aliases for `UpdateUserPayload`, `UpdateUserStatusPayload`, and `ResetUserPasswordPayload`.
- Ran the Phase 2E user write payload rollout validation closeout for the `web` side.
- Rechecked `web/package.json` `openapi:types:check` against the current worktree and confirmed the repo-local `.tmp/` plus project-Prettier alignment fix is already present; no additional `web/package.json` or `.prettierignore` change was required in this slice.
- Validated the closeout state with `cd web && bun run openapi:types:check` and the full frontend entrypoint `cd web && bun run check`.
- Audited the current codebase against the original Phase 3 goal and confirmed the generated TypeScript consumption target is already satisfied by the existing `auth`, `user`, and `rbac` alias layers over `web/src/contracts/openapi/generated/schema.ts`.
- Confirmed `web/package.json` has no `openapi-fetch` dependency and no separate generated runtime client exists in the current web tree.
- Closed the optional lightweight client evaluation with a "do not introduce `openapi-fetch` now" result because `web/src/utils/request.ts` still owns the repository's token refresh, locale header propagation, envelope unwrap, auth-failure redirect, and API error normalization semantics.
- Recorded Phase 3 as complete without widening runtime scope, keeping generated schema aliases as the preferred API-boundary consumption pattern and leaving Phase 4 Go-generation evaluation unchanged.
- Revalidated the accepted Phase 3 closeout state with `cd web && bun run openapi:types:check`, `cd web && bun run check`, and `cd server && go run ./cmd/graft validate backend --stage openapi`.
- Audited the current codebase for Phase 4 and confirmed there is still no `oapi-codegen` dependency, no generated Go contract directory, and no generated server-interface wiring in the active backend validation or plugin lifecycle path.
- Used the current backend DTO layout in `server/plugins/user/dto_http_request.go` and `server/plugins/rbac/dto_http_request.go` as concrete evidence that introducing generated Go models now would create a second server-side DTO truth without a matching ownership or validation benefit.
- Closed Phase 4 as an evaluated deferred/no-go result for the current rollout scope: keep `spec-first + TS-first + explicit server DTOs`, do not let `oapi-codegen` take over server interfaces, and do not invent new generated-Go runtime wiring in this topic.
- Revalidated the accepted Phase 4 closeout state with `cd web && bun run check` and `cd server && go run ./cmd/graft validate backend`.
