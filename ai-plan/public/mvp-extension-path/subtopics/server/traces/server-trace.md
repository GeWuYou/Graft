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

## Next Step

- Execute live Atlas validation against a disposable PostgreSQL target, then continue into the real auth + RBAC
  backend path.
