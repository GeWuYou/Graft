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
- `server/plugins/user` now contains the first auth utility layer for bcrypt password hashing and HS256 access-token
  issue/parse helpers, and also exposes the minimal `pluginapi.AuthService` needed to parse bearer access tokens and
  resolve the current user from stable request claims.
- `server/plugins/rbac` now exists as the minimal authorization plugin that exposes `pluginapi.Authorizer` on top of
  the stable RBAC repository boundary.

## Active Risks

- Atlas CLI execution still lacks live validation against a disposable PostgreSQL target in this environment.
- The current request-auth chain still lacks login, refresh-token rotation, session revocation, and cookie handling.
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

## Immediate Next Step

- Wire login, refresh-session rotation, and request-auth session hardening onto the new bearer-token and request-auth
  context path without leaking Ent details into `pluginapi` or core middleware.
