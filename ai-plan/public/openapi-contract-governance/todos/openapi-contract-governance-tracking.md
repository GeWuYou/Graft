# OpenAPI Contract Governance Tracking

## Current State

- Phase 2A minimal spec and validation wiring landed in owned scope.
- Phase 2B minimal TypeScript generation wiring now lands generated OpenAPI types in `web/src/contracts/openapi/generated/schema.ts`.
- Root `openapi/` spec plus fragments now exist for the first covered endpoints.
- Backend validation now has explicit OpenAPI validation wiring.
- No OpenAPI-driven business behavior changes were introduced.
- The audited Phase 2A spec now passes the actual `kin-openapi` validation path used by `graft validate openapi`.

## Active Goals

- Establish OpenAPI First as the long-term contract truth.
- Keep `web/src/utils/request.ts` as the transport/runtime truth for token refresh, locale, and trace propagation.
- Use generated TypeScript types without creating a second source of truth.
- Keep `oapi-codegen` out of the server interface for the initial phases.

## Phase Ownership

- Phase 1: contract layout, schema design, CI shape, and frontend type-generation plan.
- Phase 1.5: server boundary review for Ent / DTO / mapper / OpenAPI generated-code placement decisions.
- Phase 1.6: same-package lightweight reorganization of `server/plugins/user` and `server/plugins/rbac` route-layer files to reduce later OpenAPI DTO integration risk.
- Phase 2: spec fragments and validation wiring.
- Phase 3: generated TS consumption and optional lightweight client evaluation.
- Phase 4: delayed Go generation evaluation.

## Phase 2A Notes

- Covered endpoints are limited to `/healthz`, `/api/auth/login`, `/api/auth/refresh`,
  `/api/auth/logout`, `/api/auth/bootstrap`, `/api/users`, `/api/roles`, `/api/permissions`.
- `/healthz` remains a plain JSON response and is intentionally not modeled as the standard envelope.
- `httpx.WriteSuccess` and `httpx.WriteLocalizedError` semantics remain the canonical runtime truth.
- OpenAPI validation is wired through `graft validate openapi` and `graft validate backend --stage openapi`,
  and runs first in the full backend validation chain.
- This phase still does not generate Go code and does not switch web runtime calls to generated clients.
- During audit/repair, the root spec dropped `info.summary` because the current `kin-openapi` validator rejects that
  field in this repository's OpenAPI validation path.
- Audit validation evidence:
  - `cd server && go run ./cmd/graft validate openapi`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd server && go test ./internal/cli -run 'TestRunValidateBackend(OpenAPIStage|FullStage|LintStage)|TestResolveBackendModuleRootFrom(ServerDir|RepoRoot)'`
  - `cd server && go build ./cmd/graft`

## Phase 2B Notes

- `web/package.json` now owns `openapi:types` and `openapi:types:check` as the minimal TypeScript generation entrypoints.
- `openapi:types` generates `web/src/contracts/openapi/generated/schema.ts` from `openapi/openapi.yaml`, then formats the tracked output with Prettier.
- `openapi:types:check` generates to a temporary `.ts` file, formats that temporary output, and compares it with the tracked generated file to detect drift without polluting the worktree on failure.
- `request.ts` remains unchanged, and `modules/user/api` plus `modules/rbac/api` still use handwritten `request.get/post<T>` typing.
- This phase does not consume generated types in business API modules and does not generate Go code.
- Phase 2B validation evidence:
  - `cd web && bun run openapi:types`
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run check`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`

## Phase 2C Notes

- Phase 2C now covers the full currently-modeled frontend response DTO consumption surface for:
  - `POST /api/auth/login`
  - `POST /api/auth/refresh`
  - `GET /api/auth/bootstrap`
  - `GET /api/users`
  - `GET /api/roles`
  - `GET /api/permissions`
- `auth`, `user`, and `rbac` each own thin local generated type alias layers so business API files, stores, and pages do not import `generated/schema.ts` directly.
- The rollout consumes `components['schemas']` aliases for `LoginResponse`, `BootstrapResponse`, `UserListItem`, `UserListResponse`, `RoleListItem`, `RoleListResponse`, `PermissionListItem`, and `PermissionListResponse`.
- `request.ts` remains the transport/runtime truth; this phase does not change envelope unwrap, auth refresh, locale propagation, or trace behavior.
- Request payload types for role create, role update, and permission assignment remain handwritten in this phase.
- User-list page compatibility for legacy read-only fields such as `email` and `last_login_at` stays page-local instead of weakening the generated list item alias.
- Role-list page compatibility for legacy `remark` fallback stays page-local instead of weakening the generated role DTO alias.
- During Phase 2C closeout, the tracked generated file `web/src/contracts/openapi/generated/schema.ts` was refreshed to match the current root spec, and `openapi:types:check` was fixed to format a repo-local temporary file instead of a `.prettierignore`-matched `/tmp` path.
- Next recommended follow-up is extending the same approach from response DTO consumption into request payload governance only after the current OpenAPI coverage expands beyond the present read-focused endpoints.

## Phase 2E Notes

- Phase 2E starts request payload generated type consumption with one narrow pilot only: `POST /api/roles/{id}/permissions/assign`.
- The root spec now models `ReplaceRolePermissionsRequest` under `components.schemas` and reuses it from the new RBAC write-path `requestBody`.
- `web/src/modules/rbac/types/rbac.ts` now owns the thin generated alias for `ReplaceRolePermissionsPayload`, keeping `generated/schema.ts` imports out of page and API callsites.
- The role-permission drawer keeps its local selection state and uses a narrow mapper to normalize that state into the generated API payload, rather than reusing the generated payload as page state.
- `request.ts` remains unchanged, and this phase still does not introduce `openapi-fetch`, SDK generation, or Go generated code.
- Phase 2E validation evidence:
  - `cd web && bun run openapi:types`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run check`

## `POST /api/users` Sample Notes

- A narrow cross-boundary sample now extends the root OpenAPI coverage for `POST /api/users` without broadening this topic into full write-path governance.
- `web/src/modules/user/types/user.ts` now consumes the generated `CreateUserRequest` request payload type, keeping the generated type import behind the module-local alias boundary.
- The current create-user sample explicitly keeps `request.ts` as the transport truth and only fixes module-level error consumption plus password-policy UX in the `user` page.
- The create-user page now shows the real server password policy up front: at least 12 characters, letters plus digits, and no reuse of the default admin password.
- This sample does not attempt to normalize all `server/plugins/user` or `server/plugins/rbac` write routes; future rollout should audit each route before copying the pattern.
- Sample validation evidence:
  - `cd web && bun run openapi:types`
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run test:run src/modules/user/pages/index.test.ts`
  - `cd server && go test ./plugins/user/...`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`

## `POST /api/auth/login` Sample Notes

- The auth/user slice now also consumes the generated `LoginRequest` payload type via the existing shell-local alias file `web/src/api/model/authModel.ts`.
- `web/src/app/auth/components/Login.vue` keeps its current local `account/password` form state; the generated payload type is used only at the API boundary.
- `web/src/store/modules/user.ts` keeps the explicit `account -> username` mapper before calling `loginApi`, preserving current backend behavior and avoiding generated payload types as page form state.
- Validation evidence:
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd web && bun run openapi:types`
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run check`

## `POST /api/roles` and `POST /api/roles/{id}/update` Sample Notes

- The RBAC slice now extends the root OpenAPI coverage to the existing role create and role update write paths without changing backend runtime semantics.
- `openapi/paths/roles.list.yaml` now models `POST /api/roles`, and `openapi/paths/roles.update.yaml` now models `POST /api/roles/{id}/update`.
- The root spec now owns reusable `CreateRoleRequest`, `UpdateRoleRequest`, and `EnvelopedRoleItemResponse` schemas so the role write endpoints do not duplicate request or envelope structure inline.
- `web/src/modules/rbac/types/rbac.ts` now exposes thin generated aliases for `CreateRolePayload` and `UpdateRolePayload`, keeping direct imports from `web/src/contracts/openapi/generated/schema.ts` out of page and API callsites.
- `web/src/modules/rbac/pages/index.vue` keeps its current local drawer form state and uses narrow submit-time mappers so generated payload types remain API-boundary contracts instead of page state.
- Validation evidence:
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd web && bun run check`

## Phase 1.6 Notes

- Phase 1.6 intentionally keeps `package user` and `package rbac` unchanged.
- Phase 1.6 is not the final directory architecture; it is a preparatory same-package cleanup before any future package-boundary refactor.
- Phase 1.6 does not introduce OpenAPI files, Go generated models, `oapi-codegen`, or plugin/public API changes.
