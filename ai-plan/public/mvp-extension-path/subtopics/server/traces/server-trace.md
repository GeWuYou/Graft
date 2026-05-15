# MVP Extension Path Server Trace

## 2026-05-12 backend baseline

- Added the first-pass `server` runtime shell with explicit plugin ordering, registries, lightweight DI, and the
  sample `user` plugin.
- Switched backend configuration to env-first loading with PostgreSQL and Redis as required infrastructure.
- Updated backend dependency hygiene without changing plugin-boundary rules.

## 2026-05-12 to 2026-05-13 backend contract hardening

- Moved backend data access from GORM assumptions to Ent plus Atlas versioned migrations and explicit CLI ownership.
- Narrowed plugin-facing data access to a repository/store factory boundary.
- Added Cobra entrypoints for `graft migrate up`, `graft serve`, and later `graft dev`.
- Hardened migration-directory resolution, shutdown ordering, singleton construction, and HTTP server lifecycle
  sequencing.

## 2026-05-13 backend governance and extension hooks

- Completed the hand-written backend comment-governance sweep across the relevant runtime, registry, repository, and
  sample plugin packages.
- Added shared logger and i18n services to backend core and reserved the localized error-response contract.
- Added focused backend PR-review follow-up fixes, including migration-command fallback behavior and the first `en-US`
  error catalog slice.

## 2026-05-14 subtopic extraction

- Extracted backend recovery state from the overloaded parent `mvp-extension-path` topic into this dedicated `server`
  subtopic.
- Left cross-boundary direction, shared risks, and shared validation summaries in the parent topic.

## 2026-05-14 auth and RBAC persistence baseline

- Extended the `users` schema with password-hash fields while keeping the existing `store.User` DTO boundary stable.
- Added Ent schema plus Atlas migration assets for `refresh_sessions`, `roles`, `permissions`, `user_roles`, and
  `role_permissions`.
- Expanded the store factory with dedicated `Auth` and `RBAC` repository entrypoints so future plugins can consume
  stable DTOs without touching Ent client internals.
- Validated the repository-boundary fallout with `go test ./internal/app ./plugins/user ./internal/store ./internal/store/entstore`
  and kept `go build ./cmd/graft` green.

## 2026-05-14 auth utility baseline

- Added auth configuration defaults and validation for token TTLs, signing secret/key, and refresh-cookie settings.
- Reserved the stable auth DTO and service interfaces in `pluginapi` for future request-auth and RBAC wiring.
- Added the first `server/plugins/user` auth utility layer for bcrypt password hashing and HS256 access-token
  issue/parse helpers, while intentionally keeping login, refresh, and request middleware out of scope for this slice.
- Validated the utility layer with `go test ./plugins/user ./internal/config ./internal/pluginapi ./internal/store ./internal/store/entstore ./internal/app`
  and kept `go build ./cmd/graft` green.

## 2026-05-15 PR #7 review follow-up

- Removed hard-coded auth signing defaults so `server` now requires explicit JWT signing material from environment or dotenv inputs.
- Hardened the Ent-backed store boundary with `ErrInvalidID`, a nil-client fail-fast guard, and repository-level mapping that keeps invalid identifiers distinct from true not-found cases.
- Tightened schema safety by marking `users.password_hash` as sensitive and enforcing namespaced permission codes at the Ent schema layer.
- Normalized the active `web` tracking snapshot to repository-portable Bun commands so `ai-plan/**` no longer records machine-specific absolute paths.
- Aligned the `migrate` CLI fallback regression test with the new explicit auth-signing requirement, so `go test ./...` no longer fails on missing JWT signing configuration before reaching the context-fallback assertion.

## 2026-05-15 request auth context slice

- Replaced the `server/internal/httpx` request-header authorization placeholder with bearer-token parsing plus a stable request auth context built on `pluginapi.RequestAuthContext`.
- Extended `server/internal/pluginapi` with stable auth error semantics and context helpers so core middleware, auth parsing, and authorization decisions can exchange one explicit request-auth view without framework-global coupling.
- Registered the minimal `pluginapi.AuthService` in `server/plugins/user`, reusing the existing HS256 access-token helper and stable user repository boundary to resolve the current request user.
- Added the first real `server/plugins/rbac` plugin and exposed `pluginapi.Authorizer` on top of the RBAC repository boundary, then wired `graft serve` to boot both `user` and `rbac`.
- Updated direct `httpx` / `user` / `rbac` tests to lock down bearer-token auth, permission denial, and request-context propagation behavior, then kept focused backend validation and `go build ./cmd/graft` green.

## 2026-05-15 minimal login slice

- Added the minimal `/auth/login` route inside `server/plugins/user`, keeping the business logic inside the plugin and reusing `store.Auth()`, `store.Users()`, the bcrypt helper, and the existing HS256 access-token helper.
- Kept the HTTP failure contract on the existing localized `message_key + message + locale` structure, and introduced one stable `auth.invalid_credentials` message key for login failures.
- Returned the minimal login payload as `access_token + expires_at + current user summary`, without adding refresh-token rotation, cookie handling, or session persistence to this slice.
- Added direct `server/plugins/user` route tests for successful login, invalid credentials, and missing input, plus the matching `server/internal/i18n` catalog assertion, then revalidated with `go test ./plugins/user ./internal/i18n` and `go build ./cmd/graft`.

## 2026-05-15 refresh session slice

- Extended `server/plugins/user` so successful `/auth/login` now creates a refresh session, signs a refresh token, and writes the configured refresh cookie while keeping token/cookie helpers inside the plugin.
- Added `POST /api/auth/refresh` inside the same plugin boundary, reusing the localized error contract and rotating refresh sessions before returning a new access token plus replacement refresh cookie.
- Narrowed the extra store expansion to one transactional `RotateRefreshSession` method because the old `create/get/revoke` trio left a double-consume race between refresh validation and revocation.
- Added direct `server/plugins/user` tests for login cookie write, refresh success, and missing-cookie failure, and extended `server/internal/i18n` with the stable `auth.invalid_refresh_session` catalog key.

## 2026-05-15 logout current-session slice

- Added `POST /api/auth/logout` inside `server/plugins/user`, keeping logout, refresh-cookie parsing, current-session revoke, and cookie clearing inside the plugin boundary instead of pushing them into core or middleware helpers.
- Reused the existing refresh-token parser plus `GetRefreshSessionByTokenID` and `RevokeRefreshSession` store methods for the minimal current-session revoke loop, so this slice did not widen the store boundary.
- Added direct `server/plugins/user` tests for successful current-session logout and missing-cookie failure, while keeping logout failures on the existing localized `auth.invalid_refresh_session` contract.

## 2026-05-15 request-auth session hardening slice

- Tightened `server/plugins/user` so `pluginapi.AuthService.ParseAccessToken` now validates the access-token-linked session state in addition to JWT syntax and signature.
- Kept the hardening logic inside the `user` plugin by reusing the existing `AuthRepository.GetRefreshSessionByTokenID` boundary, so no new `pluginapi` or `httpx` contract expansion was required for this slice.
- Reused the existing unauthenticated response path by mapping missing, revoked, expired, or mismatched sessions to the current access-token failure semantics instead of adding new HTTP error keys.
- Updated direct `server/plugins/user` tests to seed valid sessions for protected requests and added inactive-session coverage for missing, revoked, and expired session states before rerunning focused backend validation.

## 2026-05-15 current-user all-sessions revoke slice

- Added `POST /api/auth/sessions/revoke-all` inside `server/plugins/user` as the minimal self-service revoke entrypoint, guarded by the existing bearer request-auth path instead of a new core auth concept.
- Narrowed the extra store expansion to one idempotent `AuthRepository.RevokeRefreshSessionsByUserID` operation because the existing single-session revoke methods did not cover current-user bulk revoke without leaking session iteration into the plugin.
- Reused the current localized error contract and refresh-cookie helper so successful revoke-all clears the current cookie, while later protected requests and refresh attempts fail through the already-established unauthenticated and invalid-refresh paths.
- Added direct `server/plugins/user` route coverage for successful revoke-all and missing-actor rejection, plus a focused `entstore` invalid-ID test for the new bulk-revoke repository boundary before rerunning focused backend validation and `go build ./cmd/graft`.

## 2026-05-15 admin user-session revoke slice

- Added the minimal admin-driven `POST /api/users/:id/sessions/revoke-all` route inside `server/plugins/user`, keeping the business logic on top of the existing auth/session repository boundary instead of widening core or schema responsibilities.
- Registered the dedicated plugin-local permission code `user.session.revoke` so the revoke-by-user-ID entrypoint stays explicit and does not silently piggyback on `user.read`.
- Reused `AuthRepository.RevokeRefreshSessionsByUserID` for the admin path, and only cleared the current refresh cookie when the authenticated operator revoked their own sessions, preserving the existing cookie/error contract for all other cases.
- Added direct plugin-route tests for successful target-user revoke, self-revoke cookie clearing, dedicated-permission enforcement, and invalid-ID rejection, then revalidated with `go test ./plugins/user` and `go build ./cmd/graft`.

## 2026-05-15 active-session visibility slice

- Added current-user `GET /api/auth/sessions` plus admin `GET /api/users/:id/sessions` inside `server/plugins/user`, keeping session visibility in the existing plugin boundary instead of widening `pluginapi` or core auth contracts.
- Narrowed the extra repository expansion to one active-only `ListActiveRefreshSessionsByUserID` operation so the first visibility slice returns only non-revoked, non-expired refresh sessions in a stable order without exposing rotation history.
- Registered the dedicated permission code `user.session.read` for admin session visibility and kept the current-user path on the existing authenticated request context without adding a second auth model.
- Added direct plugin tests for current-user and admin session listing, current-session marking, dedicated-permission enforcement, and user-not-found behavior, plus an `entstore` invalid-ID guard for the new repository boundary before rerunning focused backend validation.

## 2026-05-15 disposable PostgreSQL + Atlas live validation

- Reused the current auth/session and RBAC migration assets against a disposable local PostgreSQL container by building the current `graft` CLI, then running `graft migrate up` with explicit database, Redis, and auth-signing environment inputs.
- Confirmed the live migration path was idempotent by rerunning `graft migrate up`, which returned `No migration files to execute` after the first successful apply.
- Verified Atlas state with `atlas migrate status`, which reported current version `202605140001`, `Executed Files: 2`, and `Pending Files: 0`.
- Queried the disposable PostgreSQL target to confirm the six expected auth/session/RBAC tables, the `users.password_hash` and `users.password_changed_at` columns, and the foreign-key constraints on `refresh_sessions`, `user_roles`, and `role_permissions`.
- Revalidated the affected backend surface with focused `go test ./internal/cli ./internal/app ./internal/store ./internal/store/entstore ./plugins/user ./plugins/rbac`.
- Added one minimal runtime smoke check by starting `graft serve` against the disposable PostgreSQL target plus a disposable Redis target and receiving `200 OK` from `/healthz`.

## Next Step

- Add admin-driven session controls or richer audit-linked session governance on top of the new login, refresh,
  logout, current-user revoke-all, and request-auth hardening chain.
