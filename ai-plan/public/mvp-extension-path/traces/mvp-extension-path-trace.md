# MVP Extension Path Trace

## 2026-05-12 topic bootstrap

- Established `mvp-extension-path` as the first long-lived active topic for Graft and bound it to branch
  `feat/mvp-extension-path`.
- Migrated repository-wide design and roadmap truth from `plan/` into `ai-plan/`.
- Added `ai-plan/design/AI任务追踪与恢复设计.md` and aligned `AGENTS.md`, `README.md`, and `graft-boot` with the new
  recovery model.

## 2026-05-12 to 2026-05-14 shared MVP milestones

- Landed the first executable MVP shell across `server` runtime scaffolding and the `web` admin shell.
- Fixed repository-wide truth on Ent, Atlas versioned migrations, explicit migration CLI flow, plugin-facing store
  boundaries, and cross-boundary i18n extension hooks.
- Added repository-level PR review support, comment governance truth, and frontend governance truth.
- Preserved detailed pre-subtopic history in
  `ai-plan/public/mvp-extension-path/archive/traces/mvp-extension-path-trace-pre-subtopics-2026-05-14.md`.

## 2026-05-14 parent/subtopic split

- Refactored `mvp-extension-path` from one overloaded active topic into one parent topic plus bounded `server` and
  `web` subtopics.
- Kept the parent topic as the default `boot` recovery entrypoint so startup remains stable.
- Moved pure backend execution history and recovery state into the `server` subtopic.
- Moved pure frontend execution history and recovery state into the `web` subtopic.
- Limited the parent topic to cross-boundary direction, shared risks, shared validation summaries, and subtopic entry
  guidance.

## 2026-05-15 first real bootstrap contract hookup

- Landed the first shared `auth + current user + permission + menu + locale` bootstrap contract as protected
  `GET /api/auth/bootstrap` inside the existing `server/plugins/user` boundary.
- Replaced the `web` starter shell's mock login/bootstrap path with real `login / refresh / bootstrap` calls and
  switched the first dynamic menu slice to consume backend bootstrap menus instead of static demo menus.
- Kept the initial real dynamic route scope intentionally narrow by only enabling backend-returned menus that already
  have page implementations in `web`, with `/users` as the first hooked route.
- Revalidated the cross-boundary slice with focused backend validation and one full host Windows Bun `bun run check`
  pass on `web`.

## 2026-05-15 auth response convergence

- Completed the first cross-boundary convergence pass for auth / RBAC response semantics, freezing the current
  `AUTH_*` code mapping, the shared success/error envelope, and the request-id/trace-id propagation path in `server`.
- Completed the paired frontend convergence pass so `web` request recovery now refreshes only on
  `AUTH_TOKEN_EXPIRED`, exits through one cleanup path on `AUTH_TOKEN_INVALID / AUTH_TOKEN_MISSING`, and no longer
  relies on localized message text for auth control flow.
- Replaced the earlier request-layer dynamic store import with an explicit auth session bridge registered by the
  `user` store, removing the warning that previously prevented `bun run check` from satisfying the repository's
  zero-warning completion rule.
- Revalidated the full cross-boundary slice with focused backend validation, focused frontend Vitest + typecheck, and
  one full host Windows Bun `bun run check` pass.

## 2026-05-15 default-admin and forced-password-change docs sync

- Recorded the next bounded cross-boundary auth-governance slice in repository truth before code implementation.
- Fixed the MVP boundary so `graft-admin` is documented as an initialization-only exception password, while forced
  password change remains backend-persisted truth surfaced through `login/bootstrap`.
- Fixed the current stage responsibility split so `web` owns the post-login restricted-state blocking path and future
  stronger server-side middleware remains an explicit hardening step rather than accidental scope creep.

## 2026-05-16 startup governance minimum migration

- Landed the minimum startup-governance migration in repository truth by making the root `AGENTS.md` the only
  authoritative source for startup preflight, startup receipt, resume/restart revalidation, and minimum subagent
  inheritance requirements.
- Reduced `graft-boot` to an executor of `AGENTS.md` startup rules instead of a parallel boot definition, and updated
  `graft-multi-agent-batch` so subagent delegation now requires inherited startup context rather than objective-only
  dispatch.
- Repositioned `ai-plan/README.md`, `ai-plan/public/README.md`, and `ai-plan/design/AI任务追踪与恢复设计.md` as
  recovery-system documents only, explicitly separating repository recovery from repository startup governance.
- Synchronized the parent topic tracking entry so future governance drift can be observed through the active topic
  instead of reintroducing ad-hoc startup notes elsewhere.

## Next Step

- Continue MVP work through the relevant subtopic, keeping `/api/auth/bootstrap` stable while expanding real
  `server + web` page hookups instead of widening backend-only governance behavior or restoring demo auth/menu paths.
