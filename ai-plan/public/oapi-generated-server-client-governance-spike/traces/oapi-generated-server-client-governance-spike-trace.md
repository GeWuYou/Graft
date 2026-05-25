# OAPI Generated Server/Client Governance Spike Trace

## 2026-05-25 implementation spike

- Renamed the worktree topic branch from `feat/wt-oapi-codegen-types-only-spike` to `feat/oapi-generated-server-client-governance-spike`.
- Added a monitor-only generated OpenAPI server contract package under `server/internal/contract/openapi/monitor/**`.
- Kept the backend router ownership explicit:
  - the monitor plugin still registers `GET /api/monitor/server-status` itself
  - the generated layer only owns parameter binding and compile-time handler interface conformance
- Rejected `strict-server` for this pilot implementation because it would force the response envelope away from `httpx`.
- Updated the monitor frontend module API so the server-status call is now operation-bound to generated OpenAPI typings while still running through `request.ts`.
- Kept page/module boundaries unchanged:
  - pages still call module API helpers
  - no page directly consumes generated client/runtime code

## Validation Notes

- The generated Go server binding emits the expected OpenAPI `3.1.x` warning from `oapi-codegen`.
- That warning does not block the pilot, but it remains a real governance risk for future broader rollout.

## 2026-05-25 Phase 4 governance review

- Completed the Phase 4 docs-only governance review for commit `eda1849`.
- Classified the spike verdict as `partial success`, not `success` and not `failed`.
- Confirmed the backend generated server adapter stayed narrow:
  - it constrains handler shape and generated parameter/header/query semantics
  - it does not take over plugin route registration, Gin middleware, `httpx` envelope ownership, or localized error handling
- Confirmed the frontend adapter delivered the clearer governance win:
  - monitor module API now binds to generated operation types
  - module response types now alias generated schemas
  - `request.ts` remains the only transport/runtime truth
  - pages continue to consume only module API and module-owned types
- Confirmed the minimal governance patches stayed proportionate:
  - generated-file lint exclusions are scoped to the monitor generated file
  - backend validation cache namespacing is limited to temp-cache isolation by module-root hash
- Recorded the main remaining gap:
  - there is still no explicit backend generated artifact freshness gate equivalent to frontend `bun run openapi:types:check`
- Settled the recommendation order:
  - first add a generated freshness/check gate
  - do not expand the pilot to another interface before that gate exists
  - do not promote generated runtime server/client ownership from this topic
- If expansion is revisited after freshness gating lands, the next low-risk candidate should be `GET /api/permissions`, not an auth/session route and not a write-heavy interface.

## Next-Session Startup Prompt

```text
使用 $graft-multi-agent-loop。

governance source: root AGENTS.md
task class: docs/automation
recovery source:
  - current repository state
  - ai-plan/public/oapi-generated-server-client-governance-spike/README.md
  - ai-plan/public/oapi-generated-server-client-governance-spike/traces/oapi-generated-server-client-governance-spike-trace.md
  - ai-plan/public/README.md
branch / worktree:
  - feat/oapi-generated-server-client-governance-spike
owned scope:
  - ai-plan/public/oapi-generated-server-client-governance-spike/**
  - ai-plan/public/README.md if topic status changes
  - docs/traces/todos only
forbidden scope:
  - 不修改 server 业务实现
  - 不修改 web 业务实现
  - 不修改 OpenAPI spec 语义
  - 不扩大 generated runtime 覆盖面
objective:
  - 为 monitor generated server/client pilot 设计并收口最小 generated freshness/check gate
  - 明确 backend generated Go artifact 与 frontend generated TS schema 的 blocking/non-blocking gate 位置
  - 判断该 gate 应该落在 docs/automation 还是单独的新治理 topic
validation:
  - git diff --check
  - git status --short
```


## 2026-05-25 Phase 5 freshness gate

- Added `scripts/openapi_generated_freshness_check.py` as the repository-owned backend generated freshness gate.
- Kept the gate in `check` mode by default:
  - regenerate monitor-only generated Go output to a temp file
  - diff against `server/internal/contract/openapi/monitor/zz_generated.types.go`
  - fail if the tracked generated artifact is stale or manually edited
- Added explicit `--mode fix` support, but did not mix regeneration into normal validation behavior.
- Wired backend freshness into `cd server && go run ./cmd/graft validate backend` through the existing `openapi` stage.
- Confirmed frontend freshness remains owned by `cd web && bun run openapi:types:check`; this slice does not replace it.
- Kept the scope monitor-only and did not broaden generated runtime coverage or endpoint migration.

## 2026-05-25 Phase 6 guarded progressive migration batch 1

- Added `server/internal/contract/openapi/rbac/**` as a narrow generated contract package for `getPermissions` only.
- Kept the RBAC plugin as the runtime owner of:
  - explicit `/api/permissions` route registration
  - permission middleware wiring
  - `httpx` success/error envelope behavior
  - read-service invocation
- Added a compile-time generated handler-shape assertion for `GET /api/permissions` without switching to generated
  router/runtime ownership.
- Updated `web/src/modules/rbac/api/rbac.ts` so `getPermissions()` now binds to the generated OpenAPI operation type
  while still calling `request.ts`.
- Extended `scripts/openapi_generated_freshness_check.py` with `backend-rbac-permissions` so the new generated backend
  artifact can be checked without weakening the existing monitor freshness gate.

## 2026-05-25 Phase 6 guarded progressive migration batch 2

- Expanded `server/internal/contract/openapi/rbac/**` from a permissions-only artifact to a guarded RBAC read batch:
  - `getPermissions`
  - `getRoles`
  - `getRolePermissions`
- Renamed the generated artifact to `server/internal/contract/openapi/rbac/zz_generated.management.go`.
- Kept the RBAC plugin as the runtime owner of:
  - explicit route registration
  - permission middleware wiring
  - `httpx` success/error envelope behavior
  - read-service invocation
- Added compile-time generated handler-shape assertions for the RBAC read batch without switching to generated
  router/runtime ownership.
- Updated `web/src/modules/rbac/api/rbac.ts` so the RBAC read helpers bind to their generated OpenAPI operation types
  while still calling `request.ts`.
- Generalized the backend freshness target naming to `backend-rbac-management` while preserving the existing monitor
  freshness gate.

## 2026-05-25 Phase 6 guarded progressive migration batch 3

- Expanded the unified RBAC management generated artifact at
  `server/internal/contract/openapi/rbac/zz_generated.management.go` to cover:
  - `getUserRoles`
  - `postUserRolesAssign`
- Kept the RBAC plugin as the runtime owner of:
  - explicit `/api/users/:id/roles` and `/api/users/:id/roles/assign` route registration
  - permission middleware wiring
  - `httpx` success/error envelope behavior
  - user-role read/write service invocation
- Bound the backend generated layer only to:
  - path/header parameter semantics for `GET /api/users/{id}/roles`
  - path/header/request-body shape for `POST /api/users/{id}/roles/assign`
  - compile-time handler interface conformance via `rbacopenapi.UserRoleServerInterface`
- Updated `web/src/modules/user/api/user-roles.ts` so user-role helpers now bind to generated OpenAPI operation types
  while still calling `request.ts`.
- Extended backend freshness validation through the unified `backend-rbac-management` target and kept the existing
  monitor target intact.

## 2026-05-25 Phase 6 guarded progressive migration batch 4

- Expanded the unified RBAC management generated artifact at
  `server/internal/contract/openapi/rbac/zz_generated.management.go` to cover:
  - `postRolePermissionAssign`
- Kept the RBAC plugin as the runtime owner of:
  - explicit `/api/roles/:id/permissions/assign` route registration
  - permission middleware wiring
  - `httpx` success/error envelope behavior
  - role-permission write-service invocation
- Bound the backend generated layer only to:
  - path/header/request-body semantics for `POST /api/roles/{id}/permissions/assign`
  - compile-time handler interface conformance via `rbacopenapi.WriteServerInterface`
- Updated `web/src/modules/rbac/api/rbac.ts` so `assignRolePermissions()` now binds to the generated OpenAPI request
  body type while still calling `request.ts`.
- Extended backend freshness validation through the same unified `backend-rbac-management` target without introducing a
  second RBAC generated artifact.

## 2026-05-25 Phase 6 guarded progressive migration batch 5

- Expanded the unified RBAC management generated artifact at
  `server/internal/contract/openapi/rbac/zz_generated.management.go` to cover:
  - `postRoles`
  - `postRoleUpdate`
- Kept the RBAC plugin as the runtime owner of:
  - explicit `/api/roles` and `/api/roles/:id/update` route registration
  - permission middleware wiring
  - `httpx` success/error envelope behavior
  - role write-service invocation and normalization behavior
- Bound the backend generated layer only to:
  - header/request-body semantics for `POST /api/roles`
  - path/header/request-body semantics for `POST /api/roles/{id}/update`
  - compile-time handler interface conformance via `rbacopenapi.WriteServerInterface`
- Updated `web/src/modules/rbac/api/rbac.ts` so `createRole()` and `updateRole()` now bind to generated OpenAPI
  operation request-body types while still calling `request.ts`.
- Kept backend freshness validation under the same unified `backend-rbac-management` target and did not introduce a
  second RBAC generated artifact.

## 2026-05-25 Phase 6 guarded progressive migration batch 6

- Added `server/internal/contract/openapi/user/**` as a narrow generated contract package for:
  - `postUsers`
  - `postUserUpdate`
- Kept the user plugin as the runtime owner of:
  - explicit `/api/users` and `/api/users/:id/update` route registration
  - permission middleware wiring
  - `httpx` success/error envelope behavior
  - user write-service invocation and validation behavior
- Bound the backend generated layer only to:
  - header/request-body semantics for `POST /api/users`
  - path/header/request-body semantics for `POST /api/users/{id}/update`
  - compile-time handler interface conformance via `useropenapi.WriteServerInterface`
- Updated `web/src/modules/user/api/users.ts` so `createUser()` and `updateUser()` now bind to generated OpenAPI
  request-body types while still calling `request.ts`.
- Added a dedicated backend freshness target `backend-user-write` so the new generated user artifact is checked without
  changing monitor or RBAC ownership boundaries.

## 2026-05-25 Phase 6 guarded progressive migration batch 7

- Expanded the existing `server/internal/contract/openapi/user/**` write artifact to cover:
  - `postUserDelete`
- Kept the user plugin as the runtime owner of:
  - explicit `/api/users/:id/delete` route registration
  - permission middleware wiring
  - `httpx` success/error envelope behavior
  - user delete service invocation and session-revoke side effects
- Bound the backend generated layer only to:
  - path/header semantics for `POST /api/users/{id}/delete`
  - compile-time handler interface conformance via `useropenapi.WriteServerInterface`
- Updated `web/src/modules/user/api/users.ts` so `deleteUser()` now binds to the generated `postUserDelete` operation
  response typing while still calling `request.ts`.
- Kept backend freshness validation under the existing `backend-user-write` target without introducing a second user
  generated artifact.

## 2026-05-25 Phase 6 guarded progressive migration batch 8

- Kept the existing auth generated artifact at `server/internal/contract/openapi/auth/zz_generated.auth.go` scoped to:
  - `postAuthLogin`
  - `getAuthBootstrap`
- Kept the auth plugin as the runtime owner of:
  - explicit `/api/auth/login` and `/api/auth/bootstrap` route registration
  - guard and middleware wiring
  - `httpx` success/error envelope behavior
  - login/bootstrap service invocation and validation behavior
- Bound the backend login route directly to the generated OpenAPI request-body type and removed the stale handwritten
  login request DTO so the generated request shape remains the only route-entry contract for this slice.
- Kept frontend login/bootstrap typing at the module API boundary:
  - `web/src/modules/auth/api/auth.ts` now accepts the module-owned generated `LoginPayload` alias for login
  - `getBootstrap()` continues to unwrap the generated operation response through `request.ts` into the module-owned
    `BootstrapResponse` alias
- Confirmed backend freshness coverage already exists under `backend-auth-session`; this slice did not require spec or
  generated artifact changes.

## 2026-05-25 Phase 6 guarded progressive migration batch 9

- Expanded the existing auth generated artifact at `server/internal/contract/openapi/auth/zz_generated.auth.go` to cover:
  - `postAuthRefresh`
  - `postAuthLogout`
- Kept the auth plugin as the runtime owner of:
  - explicit `/api/auth/refresh` and `/api/auth/logout` route registration
  - refresh-cookie read/write/clear behavior
  - `httpx` success/error envelope behavior
  - refresh/logout service invocation and validation behavior
- Bound the backend generated layer only to:
  - header/security semantics for `POST /api/auth/refresh`
  - header/security semantics for `POST /api/auth/logout`
  - compile-time handler interface conformance via `authopenapi.ServerInterface`
- Updated `web/src/modules/auth/api/auth.ts` so `refresh()` and `logout()` now bind to generated OpenAPI operation
  response types while still calling `request.ts`.
- Kept the module API boundary explicit:
  - `refresh()` continues to expose login-response data semantics
  - `logout()` absorbs the generated empty envelope and still resolves as `Promise<void>`
- Extended backend freshness validation under the existing `backend-auth-session` target without introducing a second
  auth generated artifact.

## 2026-05-25 Phase 6 guarded progressive migration batch 10

- Resolved the blocked Batch 2 auth sessions migration without broadening scope.
- This round only touched these four current-user session interfaces:
  - `GET /api/auth/sessions`
  - `POST /api/auth/sessions/revoke-all`
  - `POST /api/auth/sessions/revoke-others`
  - `POST /api/auth/sessions/{sessionID}/revoke`
- Kept Batch 1 commit `713a676` intact:
  - `POST /api/auth/refresh`
  - `POST /api/auth/logout`
- Did not start Batch 3 password flows:
  - `POST /api/auth/change-password`
  - `POST /api/auth/complete-required-password-change`
- Expanded the existing auth generated artifact at `server/internal/contract/openapi/auth/zz_generated.auth.go` to cover:
  - `getAuthSessions`
  - `postAuthSessionsRevokeAll`
  - `postAuthSessionsRevokeOthers`
  - `postAuthSessionRevoke`
- Updated backend freshness coverage under the existing `backend-auth-session` target so the checked-in auth generated
  file must match the current generator output for both Batch 1 and Batch 2 operations.
- Kept the auth plugin as the runtime owner of:
  - explicit session route registration
  - route-local validation and mapper boundaries
  - service-command invocation
  - `httpx` success/error envelope behavior
- Bound the backend generated layer to:
  - `GET /api/auth/sessions` header/query semantics
  - `POST /api/auth/sessions/revoke-all` header semantics
  - `POST /api/auth/sessions/revoke-others` header semantics
  - `POST /api/auth/sessions/{sessionID}/revoke` header semantics plus compile-time operation coverage
- Recorded the current `oapi-codegen` constraint for this slice:
  - with the repository's current `--generate types` flow, the generated Go params type for
    `postAuthSessionRevoke` does not expose `sessionID`
  - the plugin therefore keeps explicit `ginCtx.Param(\"sessionID\")` ownership and validation, while frontend typing
    still consumes the generated OpenAPI path-param contract
- Updated `web/src/modules/auth/api/auth.ts` so the auth module API now binds these four session endpoints to generated
  OpenAPI operation types while still using `request.ts` as the only transport/runtime truth.
- Focused validation results:
  - passed: `cd server && go test ./internal/contract/openapi/auth ./plugins/auth`
  - passed: `python3 scripts/openapi_generated_freshness_check.py --target backend-auth-session --mode check`
  - passed: `cd web && bun run test:run -- auth`
  - passed: `cd web && bun run typecheck`
  - environment note: `cd web && bun test -- auth` is not the repository's Vitest entrypoint and runs Bun's native
    test runner instead; it fails on existing repo-level test infrastructure assumptions such as `import.meta.glob`
    and `vi.hoisted`
- Completion validation results:
  - passed: `cd server && go run ./cmd/graft validate backend`
  - passed: `cd web && bun run check`
  - passed: `git diff --check`
  - passed: `git status --short` after commit returned clean
- Commit status:
  - committed: `a28ea34`
  - title: `feat(auth): migrate session endpoints to generated contracts`

## 2026-05-25 boundary adapter closure follow-up

- Reframed the topic goal explicitly as:
  - `OpenAPI-generated boundary adapter closure with explicit runtime bridge`
  - not `generated runtime full takeover`
- Confirmed the accepted runtime split stays unchanged:
  - frontend `request.ts` remains the only HTTP runtime truth
  - backend `Gin` / `httpx` / plugin runtime remain the only backend runtime truth
  - OpenAPI-generated types only constrain cross-boundary adapter shapes
- Tightened frontend adapter closure without expanding endpoint scope:
  - removed remaining direct OpenAPI path literal type bridges in:
    - `web/src/modules/user/api/users.ts`
    - `web/src/modules/user/api/user-roles.ts`
    - `web/src/modules/rbac/api/rbac.ts`
  - promoted template-path truth into module contract constants in:
    - `web/src/modules/user/contract/paths.ts`
    - `web/src/modules/rbac/contract/paths.ts`
  - kept all module adapters `request.ts`-compatible and did not introduce a second transport
- Tightened backend response-boundary closure only where the generated shape fit the current explicit runtime bridge:
  - `server/plugins/auth/dto_http.go` now reuses generated-derived bootstrap menu and locale leaf types
  - kept `login/bootstrap/session` outer response wrappers explicit so existing plugin/runtime tests and bridge semantics remain stable
  - intentionally did not force generated anonymous outer structs into `user` / `rbac` response wrappers because that would expand scope into broad mapper/test rewrites rather than boundary closure
- Validation for this follow-up slice:
  - passed: `git diff --check`
  - passed: `cd web && bun run typecheck`
  - passed: `cd web && bun run test:run -- src/modules/auth/api/auth.test.ts src/modules/user/api/users.test.ts src/modules/user/api/user-roles.test.ts src/modules/rbac/api/rbac.test.ts src/modules/user/api/user-sessions.test.ts src/modules/monitor/api/server-status.test.ts`
  - passed: `cd server && go test ./plugins/auth ./plugins/user ./plugins/rbac`
  - passed: `python3 scripts/openapi_generated_freshness_check.py --target backend-auth-session --mode check`
  - passed: `python3 scripts/openapi_generated_freshness_check.py --target backend-user-write --mode check`
  - passed: `python3 scripts/openapi_generated_freshness_check.py --target backend-rbac-management --mode check`
  - passed: `cd web && bun run openapi:types:check`
- Remaining non-blocking gap after this slice:
  - backend response wrappers are now only partially generated-derived because some generated outer response structs use anonymous nested members and generated pointer semantics that would otherwise force broad, non-boundary-scoped bridge/test rewrites

## 2026-05-25 backend dto file cleanup batch

- Ran this cleanup after commit `777227b` as a bounded `graft-multi-agent-batch` wave.
- Main-agent orchestration kept ownership of:
  - integration review
  - final validation
  - closeout notes
- Worker slice `auth + user`:
  - removed `server/plugins/auth/dto_http.go`
  - removed `server/plugins/user/dto_http_response.go`
  - moved still-required explicit user boundary structs closer to their real owners:
    - bootstrap response structs into `server/plugins/user/bootstrap.go`
    - login/session helper structs into `server/plugins/user/session.go`
    - user list response structs into `server/plugins/user/mapper_http.go`
  - upgraded auth mapper output to generated OpenAPI response models in `server/plugins/auth/mapper_http.go`
- Worker slice `rbac`:
  - removed `server/plugins/rbac/dto_http_request.go`
  - removed `server/plugins/rbac/dto_http_response.go`
  - moved still-required RBAC response payload structs into `server/plugins/rbac/mapper_http.go`
- Resulting rule after this cleanup:
  - plugin-local `dto_http*.go` files are no longer needed for the current OpenAPI-generated boundary adapter shape
  - explicit boundary structs may still exist, but only inline beside the mapper/bootstrap/session code that truly owns them
  - generated outer response models are used where they fit the explicit runtime bridge cleanly
  - where generated outer response models would force anonymous-struct churn or pointer-semantics drift, the explicit wrapper stays local instead of living in a separate DTO file
- Validation for the full cleanup batch:
  - passed: `git diff --check`
  - passed: `cd server && go test ./plugins/auth ./plugins/user ./plugins/rbac`
  - pending main-agent final pass:
    - `cd server && go run ./cmd/graft validate backend`
    - `cd web && bun run openapi:types:check`
    - `cd web && bun run check`

## 2026-05-25 Phase 6 guarded progressive migration batch 11

- Completed the final auth generated-contract migration batch without broadening scope.
- This round only touched these two password-flow interfaces:
  - `POST /api/auth/change-password`
  - `POST /api/auth/complete-required-password-change`
- Kept prior auth batches intact:
  - Batch 1 commit `713a676` remains the completed refresh/logout slice
  - Batch 2 commit `a28ea34` remains the completed current-user sessions slice
- Expanded the existing auth generated artifact inputs for `server/internal/contract/openapi/auth/zz_generated.auth.go` to cover:
  - `postAuthChangePassword`
  - `postAuthCompleteRequiredPasswordChange`
- Updated backend freshness coverage under the existing `backend-auth-session` target so Batch 3 generated auth types
  are checked through the same repository-owned auth target rather than a second auth artifact.
- Kept the auth plugin as the runtime owner of:
  - explicit password route registration
  - route-local JSON binding and field validation
  - password-change service invocation
  - `httpx` success/error envelope behavior
- Bound the backend generated layer only to:
  - header/request-body semantics for `POST /api/auth/change-password`
  - header/request-body semantics for `POST /api/auth/complete-required-password-change`
  - compile-time handler interface conformance via `authopenapi.ServerInterface`
- Updated `web/src/modules/auth/api/auth.ts` so the auth module API now binds both password-flow endpoints to generated
  OpenAPI operation types while still using `request.ts` as the only transport/runtime truth.
- Kept the frontend page/form boundary explicit:
  - the module API accepts generated contract payload aliases
  - form-local state still stays outside generated transport/runtime code
- Focused validation results:
  - passed: `cd server && go test ./internal/contract/openapi/auth ./plugins/auth`
  - passed: `python3 scripts/openapi_generated_freshness_check.py --target backend-auth-session --mode check`
  - passed: `cd web && bun run test:run -- auth`
  - passed: `cd web && bun run typecheck`
- Completion validation results:
  - passed: `cd server && go run ./cmd/graft validate backend`
  - passed: `cd web && bun run check`
  - passed: `git diff --check`
  - passed: `git status --short` before commit showed only Batch 3 owned scope
- Commit status:
  - committed: `38a287f`
  - title: `feat(auth): migrate password flows to generated contracts`

## 2026-05-25 auth boundary completeness audit

- Audited the completed auth generated-contract surface without broadening scope:
  - `POST /api/auth/login`
  - `GET /api/auth/bootstrap`
  - `POST /api/auth/refresh`
  - `POST /api/auth/logout`
  - `GET /api/auth/sessions`
  - `POST /api/auth/sessions/revoke-all`
  - `POST /api/auth/sessions/revoke-others`
  - `POST /api/auth/sessions/{sessionID}/revoke`
  - `POST /api/auth/change-password`
  - `POST /api/auth/complete-required-password-change`
- Confirmed the backend boundary remains complete:
  - OpenAPI path coverage exists for all audited auth endpoints
  - `server/internal/contract/openapi/auth/**` still exposes generated request/response/operation types for the full
    audited auth slice
  - `server/plugins/auth/route_handlers.go` still keeps explicit JSON binding, header/query binding, mapper
    boundaries, service-command invocation, and `httpx` envelope ownership
  - generated DTO types still stop at the route-entry boundary and do not leak into the service layer
- Confirmed the frontend boundary remains complete:
  - `web/src/modules/auth/api/auth.ts` still binds audited operations to generated OpenAPI typing while routing every
    call through `request.ts`
  - the auth module store/tests continue to consume only module API/helpers instead of generated runtime code
- Fixed the one confirmed hardcoding drift risk:
  - `web/src/modules/auth/contract/paths.ts` now owns the canonical
    `/api/auth/sessions/{sessionID}/revoke` template
  - `web/src/modules/auth/api/auth.ts` now builds the runtime path from that module contract constant instead of a
    second handwritten full path literal
- Updated the topic README so recovery state no longer claims the topic is still monitor-only after the completed auth
  rollout slices.

## 2026-05-25 final remaining interface migration

- Completed the last repository OpenAPI interfaces that were still outside the guarded generated boundary:
  - `GET /healthz`
  - `GET /api/users/{id}/sessions`
  - `POST /api/users/{id}/sessions/revoke-all`
  - `POST /api/users/{id}/sessions/{sessionID}/revoke`
- Added `server/internal/contract/openapi/health/**` as a narrow generated contract package for `getHealthz`.
- Kept runtime core as the owner of:
  - explicit `/healthz` route registration
  - plain JSON response shape without `httpx` envelope takeover
  - runtime registry count assembly
- Expanded the existing `server/internal/contract/openapi/user/**` management artifact to cover:
  - `getUserSessions`
  - `postUserSessionsRevokeAll`
  - `postUserSessionRevoke`
- Kept the user plugin as the runtime owner of:
  - explicit admin user-session route registration
  - `id` / `sessionID` / `limit` parsing and validation
  - target-user lookup and auth-service invocation
  - `httpx` success/error envelope behavior
- Updated the frontend user module boundary with module-owned canonical templates and generated operation typing in:
  - `web/src/modules/user/contract/paths.ts`
  - `web/src/modules/user/api/user-sessions.ts`
- Added backend freshness coverage for `healthz` through:
  - `python3 scripts/openapi_generated_freshness_check.py --target backend-health --mode check`
- Expanded backend freshness coverage for the user management artifact so the same `backend-user-write` target now also
  checks the three admin user-session operations.
- Repository OpenAPI migration status after this slice:
  - `30 / 30` operations migrated
  - `remaining_interfaces = []`
- Validation results:
  - passed: `cd server && go test ./internal/contract/openapi/health ./internal/contract/openapi/user ./plugins/user`
  - passed: `cd web && bun run test:run -- user`
  - passed: `cd web && bun run typecheck`
  - passed: `python3 scripts/openapi_generated_freshness_check.py --target backend-health --mode check`
  - passed: `python3 scripts/openapi_generated_freshness_check.py --target backend-user-write --mode check`
  - passed: `cd server && go run ./cmd/graft validate backend`
  - passed: `cd web && bun run check`
  - passed: `git diff --check`
