# MVP Extension Path Server Tracking

## Subtopic

- Parent Topic: `mvp-extension-path`
- Subtopic: `server`
- Scope: `server/core`, registries, plugin lifecycle, Ent/Atlas, CLI, backend auth/RBAC path, and backend-focused
  governance follow-up

## Goal

- Keep backend recovery material separate from frontend iteration while preserving the parent `mvp-extension-path`
  topic as the default MVP entrypoint.

## Current Recovery Point

- `server` has a minimal runtime shell with explicit plugin registration, lifecycle ordering, registries, and a sample
  `user` plugin.
- The backend runtime now uses env-first configuration with PostgreSQL and Redis as required core infrastructure.
- Repository truth for backend data access is stable on Ent plus Atlas versioned migrations executed through explicit
  CLI flow.
- `plugin.Context` and cross-plugin contracts now reserve a repository/store factory boundary instead of exposing a
  concrete ORM handle.
- `graft migrate up`, `graft serve`, and `graft dev` are the supported backend entrypoints.
- Backend permission protection now uses bearer access-token parsing plus a stable request auth context wired through
  `pluginapi.AuthService` and `pluginapi.Authorizer`, with the minimal auth implementation in `user` and the minimal
  authorization implementation in `rbac`.
- The backend runtime now owns first-class logger and i18n services, and localized HTTP errors use the stable
  `message_key + message + locale` contract.
- The backend side of the comment-governance sweep is complete across the hand-written core/runtime/plugin packages.
- `server/internal/config` now carries the minimal auth configuration skeleton for token TTLs and refresh-cookie
  settings.
- `server/internal/pluginapi` now reserves the stable auth DTO and interface skeletons needed for future plugin
  wiring.
- `server/internal/ent/schema` and `server/internal/store` now reserve the MVP auth/RBAC persistence baseline,
  including password-hash fields, refresh sessions, roles, permissions, and stable repository/store DTO boundaries.
- The current auth/session and RBAC migration baseline has now been live-validated against a disposable PostgreSQL
  target through `graft migrate up`, Atlas status checks, and a minimal `graft serve` healthz probe.
- `server/internal/cli` now also exposes `graft validate smoke` as the repository-local minimal backend validation
  entrypoint for already-prepared disposable PostgreSQL/Redis targets, explicitly composing `migrate up` plus a
  one-shot runtime health probe without adding Docker provisioning magic to core or CLI startup.
- `server/plugins/user` now contains the first auth utility layer for bcrypt password hashing and HS256 access-token
  issue/parse helpers, and also exposes the minimal `pluginapi.AuthService` needed to parse bearer access tokens and
  resolve the current user from stable request claims.
- `server/plugins/user` now also exposes the minimal `/auth/login` route, reusing `store.Auth()` + `store.Users()`,
  the bcrypt helper, and the access-token helper to return localized invalid-credentials errors plus the current user
  summary without leaking Ent details into pluginapi or core.
- `server/plugins/user` now also closes the minimal refresh-session loop by persisting refresh sessions on login,
  writing the refresh cookie, and rotating the session through `POST /api/auth/refresh` while keeping helper and
  cookie semantics inside the plugin boundary.
- `server/plugins/user` now also exposes the minimal `POST /api/auth/logout` path that reads the current refresh
  cookie, revokes only that refresh session, and clears the cookie while keeping the localized error contract and
  session/cookie logic inside the plugin boundary.
- `server/plugins/user` now also exposes the minimal `POST /api/auth/sessions/revoke-all` self-service path that
  reuses the existing request-auth context to revoke the current user's full refresh-session set and clear the
  current refresh cookie without widening core auth semantics.
- `server/plugins/user` now also exposes the minimal admin-driven `POST /api/users/:id/sessions/revoke-all` path,
  protected by the plugin-local `user.session.revoke` permission, so administrators can revoke a specified user's full
  refresh-session set without extending schema or moving session governance into core.
- `server/plugins/user` now also hardens the protected bearer request path by requiring the access-token-linked
  session to still exist, remain unrevoked, and stay unexpired before `pluginapi.AuthService` accepts the token.
- `server/plugins/user` now also exposes the minimal active-session visibility path through current-user
  `GET /api/auth/sessions` and admin `GET /api/users/:id/sessions`, keeping session reading inside the plugin on top of
  a stable active-session repository boundary plus explicit `user.session.read` permission semantics.
- `server/plugins/user` now also exposes single-session targeted revoke through current-user
  `POST /api/auth/sessions/:sessionID/revoke` and admin `POST /api/users/:id/sessions/:sessionID/revoke`, keeping
  targeted session governance on top of a stable `userID + sessionID` repository boundary plus the existing explicit
  `user.session.revoke` permission semantics.
- `server/plugins/user` now also supports an explicit `limit` query on current-user `GET /api/auth/sessions` and
  admin `GET /api/users/:id/sessions`, applying a plugin-local bounded slice over the existing active-session
  repository result so the session-governance path gains a narrower list workflow without widening `core`,
  `pluginapi`, or store contracts.
- `server/plugins/user` now also exposes current-user `POST /api/auth/sessions/revoke-others`, preserving the current
  access-token-bound session while revoking the same user's other active refresh sessions through plugin-local
  orchestration on top of the existing list + targeted-revoke boundaries, without widening `core`, `pluginapi`, or
  store contracts.
- `server/plugins/rbac` now exists as the minimal authorization plugin that exposes `pluginapi.Authorizer` on top of
  the stable RBAC repository boundary.

## Active Risks

- The disposable PostgreSQL + Atlas live validation path now has a repository-local one-command validation entrypoint,
  but disposable PostgreSQL/Redis provisioning itself is still manual by design instead of hidden behind CLI or core
  automation.
- The current request-auth chain now covers the current user's full refresh-session revoke loop, but it still lacks
  richer session/audit governance beyond the minimal login/refresh/logout/self-list/self-revoke/self-revoke-others/
  admin-list/admin-revoke, targeted session revoke, explicit list limit, plus protected-request hardening loop.
- The temporary placement of minimal `AuthService` inside `server/plugins/user` keeps the critical path moving, but
  future work should reevaluate whether a dedicated auth plugin boundary is needed once login and refresh APIs land.
- Future backend work must avoid leaking Ent-specific details through `plugin.Context` or cross-plugin public APIs.

## Latest Validation

- Historical backend validation commands before the subtopic split are preserved in the parent-topic archive.
- The latest focused backend validation before this split included:
  - `cd server && go test ./internal/cli ./internal/httpx ./internal/i18n ./internal/plugin ./plugins/user`
  - `cd server && go build ./cmd/graft`
- The latest auth/RBAC persistence baseline validation included:
  - `cd server && go test ./internal/app ./plugins/user ./internal/store ./internal/store/entstore`
  - `cd server && go build ./cmd/graft`
  - `cd server && atlas migrate hash --dir file://internal/ent/migrate/migrations`
- The latest auth utility validation included:
  - `cd server && go test ./plugins/user ./internal/config ./internal/pluginapi ./internal/store ./internal/store/entstore ./internal/app`
  - `cd server && go build ./cmd/graft`
- The latest PR `#7` review-follow-up validation included:
  - `cd server && go generate ./internal/ent`
  - `cd server && go test ./internal/config ./internal/store ./internal/store/entstore ./plugins/user ./internal/app`
  - `cd server && go build ./cmd/graft`
- The latest migration CLI regression follow-up validation included:
  - `cd server && env GOCACHE=/tmp/graft-go-cache go test ./...`
- The latest request-auth-context follow-up validation included:
  - `cd server && go test ./internal/httpx ./plugins/user ./plugins/rbac`
  - `cd server && go test ./internal/cli ./internal/app ./internal/pluginapi ./internal/store ./internal/store/entstore`
  - `cd server && go build ./cmd/graft`
- The latest minimal-login slice validation included:
  - `cd server && go test ./plugins/user ./internal/i18n`
  - `cd server && go build ./cmd/graft`
- The latest refresh-session slice validation included:
  - `cd server && go test ./plugins/user ./internal/i18n ./internal/store ./internal/store/entstore`
  - `cd server && go build ./cmd/graft`
- The latest logout current-session slice validation included:
  - `cd server && go test ./plugins/user ./internal/i18n`
  - `cd server && go build ./cmd/graft`
- The latest request-auth session-hardening slice validation included:
  - `cd server && go test ./plugins/user ./internal/httpx ./internal/i18n`
  - `cd server && go build ./cmd/graft`
- The latest current-user all-sessions revoke slice validation included:
  - `cd server && go test ./plugins/user ./internal/store/entstore ./internal/i18n`
  - `cd server && go build ./cmd/graft`
- The latest admin user-session revoke slice validation included:
  - `cd server && go test ./plugins/user`
  - `cd server && go build ./cmd/graft`
- The latest active-session visibility slice validation included:
  - `cd server && go test ./plugins/user ./internal/store/entstore`
  - `cd server && go build ./cmd/graft`
- The latest targeted-session revoke slice validation included:
  - `cd server && go test ./plugins/user ./internal/store/entstore ./internal/i18n`
  - `cd server && go build ./cmd/graft`
- The latest session-list limit slice validation included:
  - `cd server && go test ./plugins/user`
  - `cd server && go build ./cmd/graft`
- The latest current-user revoke-others slice validation included:
  - `cd server && go test ./plugins/user`
  - `cd server && go build ./cmd/graft`
- The latest disposable PostgreSQL + Atlas live validation included:
  - `cd server && go build -o <temp-binary> ./cmd/graft`
  - `cd server && GRAFT_DATABASE_URL=postgres://graft:graft@127.0.0.1:<pg-port>/graft?sslmode=disable GRAFT_REDIS_ADDR=127.0.0.1:6379 GRAFT_AUTH_JWT_SECRET=<secret> <temp-binary> migrate up`
  - `cd server && GRAFT_DATABASE_URL=postgres://graft:graft@127.0.0.1:<pg-port>/graft?sslmode=disable GRAFT_REDIS_ADDR=127.0.0.1:6379 GRAFT_AUTH_JWT_SECRET=<secret> <temp-binary> migrate up` again, which returned `No migration files to execute`
  - `cd server && atlas migrate status --dir file://$(pwd)/internal/ent/migrate/migrations --url postgres://graft:graft@127.0.0.1:<pg-port>/graft?sslmode=disable`
  - Queried the disposable PostgreSQL target to confirm the six auth/session/RBAC tables, the `users.password_hash` and `users.password_changed_at` columns, and the expected foreign-key constraints on `refresh_sessions`, `user_roles`, and `role_permissions`
  - `cd server && go test ./internal/cli ./internal/app ./internal/store ./internal/store/entstore ./plugins/user ./plugins/rbac`
  - `cd server && GRAFT_DATABASE_URL=postgres://graft:graft@127.0.0.1:<pg-port>/graft?sslmode=disable GRAFT_REDIS_ADDR=127.0.0.1:<redis-port> GRAFT_AUTH_JWT_SECRET=<secret> GRAFT_HTTP_ADDR=127.0.0.1:<http-port> <temp-binary> serve`, followed by `curl http://127.0.0.1:<http-port>/healthz`
- The latest CLI smoke-entrypoint validation included:
  - `cd server && go test ./internal/cli`
  - `cd server && go build ./cmd/graft`

## Immediate Next Step

- Run `graft validate smoke` against the next disposable PostgreSQL + Redis target to replace the remaining manual
  migrate-plus-healthz invocation steps, then extend the session-management path with audit-linked governance,
  admin-preserve-current revoke flows, or richer filters while keeping the new active-session visibility, explicit
  list limit, targeted revoke, revoke-others, and bulk revoke semantics inside the existing plugin boundary.
