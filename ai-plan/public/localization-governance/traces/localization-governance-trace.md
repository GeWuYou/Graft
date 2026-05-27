# Localization Governance Trace

## 2026-05-27 active topic initialized

- Created a new active public recovery topic `localization-governance`.
- Renamed the implementation workspace from the archived-topic-derived identity to:
  - worktree `/home/gewuyou/project/go/Graft-wt/feat/wt-localization-governance`
  - branch `feat/wt-localization-governance`
- Updated `ai-plan/public/README.md` so the recovery index no longer points to `None` for this active task.
- Recorded the frozen localization governance compatibility rules before any business-code implementation:
  - locale bundles must be future-compatible with plugin-provided sources
  - all locale keys require owner namespaces
  - menu, permission display, and error semantics use stable keys
  - frontend consumes `key + fallback`
  - backend registry remains registration/validation/fallback only
  - OpenAPI stays key-semantic only and does not become a multilingual copy store

## 2026-05-27 key-first baseline audit closed

- Audited the bounded localization-governance slice across `web`, `server`, and `openapi` without reopening the archived OpenAPI topic.
- Confirmed current frontend error rendering paths already consume `messageKey + message fallback` through shared helpers or equivalent local handling.
- Confirmed bootstrap menu -> route -> breadcrumb/tab title flow already prefers `title_key` and only falls back to backend `title` when locale catalogs do not define the key.
- Confirmed backend i18n remains namespace-scoped, duplicate-protected, and freeze-aware; no late-registration or missing-owner gap was found in the scanned plugin registrations.
- Confirmed current permission `display` / `description` payloads are still fallback-only and should evolve additively with a future `display_key` contract if needed.
- Confirmed OpenAPI schema text remains limited to key-field, fallback, and locale semantics instead of expanding into a multilingual copy system.
- Promoted the topic to `closeout-ready`; remaining notes are additive follow-ups rather than blocking baseline gaps.
