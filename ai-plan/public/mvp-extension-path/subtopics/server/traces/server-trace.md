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

## Next Step

- Execute live Atlas validation against a disposable PostgreSQL target, then replace the request-header authorization
  placeholder with the real request-auth, login, refresh, and RBAC plugin chain on top of the new persistence
  baseline.
