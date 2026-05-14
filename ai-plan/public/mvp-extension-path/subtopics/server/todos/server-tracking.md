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
- Backend permission protection currently exists as an MVP placeholder based on request headers and still needs the
  real auth + RBAC plugin chain.
- The backend runtime now owns first-class logger and i18n services, and localized HTTP errors use the stable
  `message_key + message + locale` contract.
- The backend side of the comment-governance sweep is complete across the hand-written core/runtime/plugin packages.

## Active Risks

- Atlas CLI execution still lacks live validation against a disposable PostgreSQL target in this environment.
- The request-header authorization placeholder must be replaced without leaking auth logic into core or breaking plugin
  boundaries.
- Future backend work must avoid leaking Ent-specific details through `plugin.Context` or cross-plugin public APIs.

## Latest Validation

- Historical backend validation commands before the subtopic split are preserved in the parent-topic archive.
- The latest focused backend validation before this split included:
  - `cd server && go test ./internal/cli ./internal/httpx ./internal/i18n ./internal/plugin ./plugins/user`
  - `cd server && go build ./cmd/graft`
- This subtopic introduction is documentation-only and should be validated through consistency checks with the parent
  topic and repository `ai-plan` rules.

## Immediate Next Step

- Run the first disposable PostgreSQL + Atlas end-to-end backend validation path, then replace the request-header
  authorization placeholder with the real auth + RBAC plugin chain.
