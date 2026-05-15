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

## 2026-05-15 targeted-session revoke slice

- Added current-user `POST /api/auth/sessions/:sessionID/revoke` plus admin `POST /api/users/:id/sessions/:sessionID/revoke` inside `server/plugins/user`, keeping targeted session revoke in the same plugin boundary instead of widening core or `pluginapi`.
- Narrowed the extra repository expansion to one `RevokeRefreshSessionByUserID` operation so the first targeted-revoke slice constrains writes by explicit `userID + sessionID` matching and only revokes still-active sessions.
- Reused the existing `user.session.revoke` permission for admin-targeted revoke, kept the self-service path on the existing authenticated request context, and introduced the stable localized `auth.session_not_found` contract for missing or already inactive sessions.
- Added direct plugin tests for self/admin targeted revoke, current-session cookie clearing, not-found behavior, and untouched-session protection, plus `entstore` invalid-ID coverage and the matching i18n catalog assertion before rerunning focused backend validation.

## 2026-05-15 session-list limit slice

- Added an explicit plugin-local `limit` query constraint to current-user `GET /api/auth/sessions` and admin `GET /api/users/:id/sessions`, keeping the first bounded-list behavior inside `server/plugins/user` instead of widening the repository boundary into pagination semantics.
- Limited the new behavior to one narrow query parameter with a fixed upper bound so the session-governance path can cap response size while still reusing the existing newest-first active-session repository ordering.
- Reused the current localized `common.invalid_argument` contract for invalid `limit` inputs and kept the repository contract unchanged because the slice only trims already-ordered session summaries after repository resolution.
- Added direct plugin tests for current-user/admin limit application plus invalid-limit rejection before rerunning focused backend validation.

## 2026-05-15 disposable PostgreSQL + Atlas live validation

- Reused the current auth/session and RBAC migration assets against a disposable local PostgreSQL container by building the current `graft` CLI, then running `graft migrate up` with explicit database, Redis, and auth-signing environment inputs.
- Confirmed the live migration path was idempotent by rerunning `graft migrate up`, which returned `No migration files to execute` after the first successful apply.
- Verified Atlas state with `atlas migrate status`, which reported current version `202605140001`, `Executed Files: 2`, and `Pending Files: 0`.
- Queried the disposable PostgreSQL target to confirm the six expected auth/session/RBAC tables, the `users.password_hash` and `users.password_changed_at` columns, and the foreign-key constraints on `refresh_sessions`, `user_roles`, and `role_permissions`.
- Revalidated the affected backend surface with focused `go test ./internal/cli ./internal/app ./internal/store ./internal/store/entstore ./plugins/user ./plugins/rbac`.
- Added one minimal runtime smoke check by starting `graft serve` against the disposable PostgreSQL target plus a disposable Redis target and receiving `200 OK` from `/healthz`.

## 2026-05-15 CLI smoke validation entrypoint

- Added `graft validate smoke` under `server/internal/cli` as the repository-local minimal backend validation entrypoint.
- Kept the orchestration explicit by reusing the existing `migrate up` and `serve` command boundaries instead of adding Docker provisioning or startup-time migration magic.
- Defined the smoke success condition as one successful `/healthz` probe followed by an intentional runtime shutdown, so the command verifies both minimal startup and graceful stop semantics.
- Added focused `server/internal/cli` tests for migrate-before-serve ordering, migration short-circuit behavior, serve-failure propagation, health-check failure propagation, wildcard listen-address probe normalization, and root-command registration.
- Revalidated the slice with `cd server && go test ./internal/cli` and `cd server && go build ./cmd/graft`.

## 2026-05-15 current-user revoke-others slice

- Added `POST /api/auth/sessions/revoke-others` inside `server/plugins/user` so the current user can keep the
  access-token-bound session while clearing the same user's other active refresh sessions.
- Kept the implementation inside the existing plugin boundary by composing the current active-session list with the
  existing targeted revoke capability, avoiding any new `core`, `pluginapi`, store, or schema contract.
- Preserved the existing request-auth and localized-error flow by reusing the current authenticated-actor guard and
  returning `204 No Content` without clearing the current refresh cookie.
- Added focused plugin-route tests for successful revoke-others behavior and missing-actor rejection, then revalidated
  with `cd server && go test ./plugins/user` and `cd server && go build ./cmd/graft`.

## 2026-05-15 PR #8 review follow-up

- Expanded `server/.gitignore` build-output rules so the backend workspace no longer relies on the single-path `graft`
  ignore entry.
- Fixed `server/internal/cli/validate_test.go` to record smoke-validation steps under a mutex, removing the data-race
  window flagged by review on concurrent `append`.
- Hardened `server/internal/store/entstore/auth_repository.go` so refresh-session rotation only succeeds when the old
  session is still active at the conditional update point, and so successful commits no longer fall through the
  rollback defer path.
- Added direct regression coverage for reused refresh cookies in `server/plugins/user/plugin_test.go`, propagated RBAC
  repository failures in `server/plugins/rbac/plugin_test.go`, and supplemented doc comments around the auth service
  implementation to help clear the docstring coverage gate.
- Revalidated the review follow-up with `cd server && go test ./internal/cli ./internal/store/entstore ./plugins/user ./plugins/rbac`
  and `cd server && go build ./cmd/graft`.
- Followed up on the previously overlooked `greptile-apps[bot]` comments by removing the implicit RBAC dependency from
  authentication-only `httpx.RequirePermission(..., "")` routes and by narrowing `authService.Login` to pure
  authentication so it no longer issues an access token that cannot pass later session validation.

## 2026-05-15 PR #8 revoke-others idempotency follow-up

- Verified the latest PR review against local HEAD and confirmed the `Login()` orphan-session comment was already stale,
  while the `revoke-others` concurrent-expiry/revocation comment still applied to the current implementation.
- Hardened `server/plugins/user` so `RevokeOtherCurrentUserSessions` now treats an already-missing target session as an
  idempotent success inside the per-session loop, preventing one raced revoke from aborting the remaining cleanup.
- Added direct `server/plugins/user` route coverage that simulates a listed session being concurrently revoked just
  before the first targeted revoke, and locked the behavior to `204 No Content` plus continued cleanup of remaining
  sessions.
- Revalidated the focused follow-up with `cd server && go test ./plugins/user` and `cd server && go build ./cmd/graft`.

## 2026-05-15 PR #8 AI review hardening follow-up

- Re-checked the latest CodeRabbit open threads against local HEAD and kept only the still-applicable behavior,
  privacy, and test-stability findings in scope.
- Tightened `server/internal/httpx` fail-closed coverage so the missing-auth-dependency test now proves the protected
  handler is not reached after middleware failure.
- Reduced login enumeration and log-retention risk in `server/plugins/user` by adding a placeholder bcrypt compare for
  missing credentials and by removing username fields from login-failure error logs.
- Stabilized `server/plugins/user` tests by switching mutex-bearing repository helpers to pointer receivers, replacing
  timestamp-based seeded session IDs with UUIDs, and documenting the permission-registry ordering contract relied on by
  the registration test.
- Verified one reported Gin route-conflict comment against `go test ./plugins/user -run TestRegisterPublishesContracts`
  and confirmed it does not reproduce on the current route set, so no route-shape change was applied in this slice.
- Revalidated the accepted follow-up with `cd server && go test ./internal/httpx ./plugins/user`, `cd server && go vet ./plugins/user`,
  and `cd server && go build ./cmd/graft`.

## 2026-05-15 event bus slice

- Added `server/internal/eventbus` as the MVP-stage in-process event bus boundary, keeping the public surface limited
  to `Subscribe / Publish`, ordered synchronous dispatch, panic recover, and error logging.
- Wired the bus as a core runtime resource in `server/internal/app/runtime.go`, so the same `eventbus.Bus` instance is
  held by `Runtime`, registered into the singleton container, and exposed on `plugin.Context`.
- Added direct `server/internal/eventbus` tests for invalid subscription input, ordered delivery, error aggregation,
  and panic recovery, plus `server/internal/app/runtime_test.go` coverage that locks the singleton registration and
  plugin-context injection path.

## 2026-05-15 local auth config onboarding fix

- Kept `server/internal/config.Config.Validate()` strict on explicit auth signing material and did not add any dev/local fallback secret, random generation, or token-semantics change.
- Updated `server/.env.example` with a local-only JWT signing example and a production-replacement warning so local startup guidance now matches the current config contract.
- Tightened the root README startup section to document `working directory=server`, repository-root fallback loading of `server/.env`, the minimal `server/.env.example -> server/.env -> go run ./cmd/graft dev` flow, and the exact missing-secret error string.
- Expanded `server/internal/config/config_test.go` with isolated auth-signing-material cases that prove missing-both fails and single-source `JWTSecret` / `SigningKey` inputs still pass without depending on a real developer environment file.

## 2026-05-15 local auth key generator tools

- Added `server/internal/keygen` as a tiny development-only helper package that uses `crypto/rand` plus URL-safe base64 output to generate `.env`-ready auth key lines.
- Added two standalone helper programs, `server/cmd/graft-jwt-secret` and `server/cmd/graft-signing-key`, so local developers can generate either required auth environment line without touching runtime auth behavior.
- Kept the change outside the main `graft` Cobra tree to avoid coupling config-prep helpers to runtime command semantics.
- Added direct `server/internal/keygen` tests and updated the root README to document the new helper commands in the local server startup flow.

## 2026-05-15 audit slice

- Added `server/internal/audit` plus the `store.AuditRepository` boundary so request-level and active audit paths can
  converge on one minimal write-only persistence contract.
- Extended Ent with the `audit_logs` schema and migration assets, then kept the plugin-facing repository surface
  limited to stable DTO-based writes instead of exposing query DSL or ORM internals.
- Added `server/plugins/audit` to mount request audit middleware on the shared router and subscribe to
  `pluginapi.AuditRecordEventName` through the shared `eventbus.Bus`.
- Reused `server/internal/httpx` to retain the stable `message_key` in Gin context, so failed requests can record the
  same localized error contract inside audit logs without inventing a second error channel.
- Revalidated the slice with
  `cd server && go test ./internal/app ./internal/audit ./plugins/audit ./internal/store/entstore ./internal/httpx ./plugins/user`
  and `cd server && go build ./cmd/graft`.

## 2026-05-15 scheduler slice

- Added `server/internal/scheduler` as the repository-local runtime wrapper around `robfig/cron/v3`, keeping the
  public surface limited to explicit `RegisterJob / RemoveJob / Start / Stop` semantics.
- Extended `server/internal/cronx` so plugin-registered jobs now carry an explicit `Run` entrypoint and declaration
  validation, preserving the rule that `Register` only declares jobs while the runtime wrapper owns scheduling.
- Added `server/plugins/scheduler` and wired `graft serve` to boot it alongside the current core plugins, so runtime
  startup now consumes the `cron registry` snapshot and shuts the scheduler down through plugin lifecycle order.
- Revalidated the slice with `cd server && go test ./internal/scheduler ./plugins/scheduler ./internal/cli` and
  `cd server && go build ./cmd/graft`.

## 2026-05-15 bootstrap contract slice

- Added protected `GET /api/auth/bootstrap` inside `server/plugins/user`, keeping the first real
  `auth + current user + permission + menu + locale` bootstrap payload inside the existing plugin boundary instead of
  expanding `core` or adding a new shared abstraction.
- Reused the existing request-auth middleware context for current-user identity, the RBAC repository for current
  permission resolution, the menu registry for registration-order-stable menu filtering, and the i18n/config snapshot
  for locale bootstrap data.
- Added focused `server/plugins/user` route coverage for unauthenticated rejection plus the successful contract path
  that locks permission-code dedup/sort, permission-filtered menus, and locale snapshot fields.
- Revalidated the slice with `cd server && go test ./plugins/user` and `cd server && go build ./cmd/graft`.

## 2026-05-15 PR #9 review follow-up

- Re-checked the latest PR #9 open threads against local HEAD and kept only the still-applicable `server` findings in
  scope instead of mechanically applying every AI comment.
- Hardened `server/internal/audit/service.go` so normalized `Action` values now use the same trim result for validation
  and persistence, preventing avoidable whitespace drift across audit records.
- Extended `server/plugins/audit` to accept both `pluginapi.AuditEvent` values and pointers on the event bus path, and
  clarified `pluginapi.AuditEvent` field semantics so cross-plugin publishers know which fields are required, optional,
  or defaulted by the consumer.
- Deduplicated bootstrap locale fallback output when `defaultLocale` and `fallbackLocale` collapse to the same value,
  and supplemented scheduler lifecycle comments plus `Stop(nil)` wait-behavior coverage to match the repository
  documentation standard.

## 2026-05-15 PR #9 greptile server follow-up

- Verified the remaining greptile `server` findings against local HEAD instead of assuming the review threads were
  stale after the earlier PR #9 follow-up.
- Removed the unused `logJobFailure` helper from `server/plugins/scheduler` because runtime job-failure logging is
  already centralized in `server/internal/scheduler/runtime.go`.
- Narrowed `server/plugins/audit` request-level audit capture so `ResourceType` now records the first stable resource
  segment derived from the route template, while `RequestPath` continues to preserve the full route pattern for request
  tracing.
- Added focused `server/plugins/audit` regression coverage that locks the new `ResourceType` extraction contract for an
  authenticated `/api/users/:id` request.

## 2026-05-15 PR #9 scheduler shutdown context follow-up

- Re-ran the repository PR-review workflow against local HEAD and confirmed the only remaining applicable AI finding was
  the `scheduler` plugin shutdown path bypassing host lifecycle context.
- Extended `plugin.Context` with explicit `LifecycleContext` semantics, keeping Register/Boot on the runtime `runCtx`
  while switching Shutdown to a fresh bounded cleanup context so plugins do not inherit an already-canceled parent.
- Updated `server/plugins/scheduler` to forward `LifecycleContext` into the scheduler runtime stop path and added direct
  runtime/plugin tests that lock the new shutdown-context propagation contract.

## 2026-05-15 minimal users list contract

- Extended the stable `store.UserRepository` boundary with one minimal `List()` read path, explicitly limited to the
  current MVP need of replacing the fake `/users` frontend page with a real backend-driven list.
- Added `GET /api/users` inside `server/plugins/user`, reusing the existing `user.read` permission boundary and
  returning a narrow DTO of `id / username / display / created_at / updated_at` rather than widening into CRUD.
- Added focused `server/plugins/user` route coverage for successful list reads and localized internal-error behavior,
  then revalidated with `cd server && go test ./plugins/user ./internal/store/entstore` and `cd server && go build ./cmd/graft`.

## Next Step

- Keep the new bootstrap contract stable enough for `web` starter-shell hookup, then move the next batch to
  synchronized `server` + `web` work instead of widening backend-only session-governance behavior.
