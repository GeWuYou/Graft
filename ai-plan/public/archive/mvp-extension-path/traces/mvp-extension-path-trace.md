# MVP Extension Path Trace

## 2026-05-12 topic bootstrap

- Established `mvp-extension-path` as the first long-lived active topic for Graft and bound it to branch
  `feat/mvp-extension-path`.
- Migrated repository-wide design and roadmap truth from `plan/` into `ai-plan/`.
- Added `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md` and aligned `AGENTS.md`, `README.md`, and `graft-boot` with the new
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
- Repositioned `ai-plan/README.md`, `ai-plan/public/README.md`, and `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md` as
  recovery-system documents only, explicitly separating repository recovery from repository startup governance.
- Synchronized the parent topic tracking entry so future governance drift can be observed through the active topic
  instead of reintroducing ad-hoc startup notes elsewhere.

## 2026-05-16 restricted-session forced-password-change slice

- Completed the next bounded cross-boundary forced-password-change slice on top of the existing `login / refresh / bootstrap` contract.
- Fixed the current-state semantics so `must_change_password=true` now means “authenticated but restricted” rather than “logged out”, and the `web` guard keeps tokens while blocking business routes through a dedicated restricted-session entry.
- Fixed the default-admin first-login path so `server` only allows empty `current_password` when the backend can prove the actor is still the restricted default admin using the initialization-only password, while the frontend re-enters normal navigation only after a fresh `bootstrap`.

## 2026-05-16 shared authorizer convergence

- Cleared the previously recorded backend test-lint backlog and restored `graft validate backend` to a clean local
  completion baseline.
- Removed the last active authz dual-truth point on the backend by making `user` route guards bind the shared
  `pluginapi.Authorizer` from `rbac` during `Boot`, instead of maintaining a plugin-local authorization copy.
- Kept the repository-wide next-step focus on cross-boundary web-shell cleanup rather than reopening backend-only
  governance drift.

## 2026-05-16 contract-governance phase-1 foundation

- Added `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md` as the repository design truth for canonical contract ownership,
  typed boundaries, lifecycle, compatibility windows, and phase-1 magic-value governance.
- Extended `AGENTS.md`, `项目设计`, `前端架构设计`, and `插件与依赖注入设计` so high-risk contract changes now
  explicitly require canonical reuse, lifecycle clarity, and same-change doc alignment across `server` / `web`.
- Added a repository-local scanner under `scripts/magic_value/` plus local hook and CI wiring so new high-risk
  literals can be checked through one phase-1 automation path instead of ad-hoc manual review.

## 2026-05-16 first live auth contract convergence

- Added the first live canonical auth contract surface under `server/internal/contract/**` and `web/src/contracts/**`,
  covering auth/common response codes, message keys, auth scheme, request headers, auth API paths, restricted-session
  route metadata, and the persisted session/locale storage keys used by the current auth runtime.
- Rewired the current `server/internal/httpx`, `server/internal/i18n`, `web` request/session/bootstrap/route recovery
  path to consume those canonical contracts instead of maintaining parallel literals inside each runtime consumer.
- Updated the contract-governance scanner so canonical contract owner files count as definition context and the
  cross-end drift report now reads server/web auth contracts from their real owner files rather than the older
  consumer-side literal locations.

## Next Step

- Continue MVP work by pushing the next contract-governance slice into `server/plugins/user` permission/message-key/
  auth-route hotspots and the remaining shared auth literals still outside the new canonical contract surface.

## 2026-05-16 user-plugin contract-governance follow-up

- Completed the planned `server/plugins/user` follow-up by moving runtime permission and auth-route hotspots onto the
  plugin-local `contract` package and by switching user-plugin runtime error wiring to canonical `message.Key`
  consumers instead of raw strings.
- Extended the server-side canonical message contract and default i18n catalogs with shared `common.conjunction` and
  `common.copyright`, eliminating the scanner-reported shared message-key drift exposed by current `web` runtime use.
- Revalidated the slice with focused `server` tests plus a fresh contract-governance report and confirmed the targeted
  runtime findings disappeared from `plugin.go`, `plugin_routes.go`, and `server/internal/contract/message/key.go`.

## 2026-05-17 boot orchestration and mandatory closeout alignment

- Extended the boot-governance chain so `graft-boot` now orchestrates not only startup preflight but also the required
  post-boot workflow hooks: assess `graft-multi-agent-batch`, then route slice endings through `graft-task-closeout`.
- Reclassified `graft-task-closeout` as the default slice-end path for boot-started work and made it evaluate commit
  eligibility through `graft-commit` rules before any handoff can claim a safe closure.
- Updated the parent tracking/design truth so `AGENTS.md`, the four repository skills, and `ai-plan` governance notes
  no longer imply that commit/closeout logic is only available through manual skill mentions.
