# MVP Extension Path Web Trace

## 2026-05-12 frontend baseline

- Added the first-pass `web` admin shell with Vue 3, TypeScript, Vite, TDesign Vue Next, static auth, baseline
  layouts, and route/store scaffolding.
- Fixed the initial UnoCSS package-version issues and the `vue-router` augmentation issue so the shell could pass
  typecheck and production build.

## 2026-05-13 frontend governance and i18n path

- Reserved the frontend side of the shared i18n path for locale state, message lookup, and request header
  propagation.
- Tightened repository truth so the `web` governance baseline has one explicit quality chain and the local
  `web/ai-libs/tdesign-vue-next-starter` area remains a reference source rather than runtime truth.
- Converted the relevant frontend route/store boundary comments to Chinese where repository governance requires them.

## 2026-05-14 frontend PR follow-up fixes

- Fixed the TDesign `t-menu` stub so tests emit the expected `change` event.
- Split the 404 navigation test into isolated mounts.
- Guarded locale-store `localStorage` access against read/write failures.

## 2026-05-14 subtopic extraction

- Extracted frontend recovery state from the overloaded parent `mvp-extension-path` topic into this dedicated `web`
  subtopic.
- Left cross-boundary direction, shared risks, and shared validation summaries in the parent topic.

## 2026-05-14 starter shell migration

- Migrated the reusable shell layer from `web/ai-libs/tdesign-vue-next-starter` into the real `web` app without
  copying mock routes, frontend-only permission bypass, tabs-router, or demo business pages.
- Split `AuthLayout` and `BasicLayout` into dedicated layout-shell components while preserving the existing
  auth/navigation/i18n semantics and the backend-driven `menu + route + page + api + permission` direction.
- Refreshed the login page, dashboard page, and 403/404 result pages to match the starter-style admin shell while
  keeping current static auth and route-guard behavior unchanged.
- Added brand assets, favicon, auth/result visual layers, global style tokens, and a corrected TDesign input test stub
  so the real shell can pass the documented frontend quality chain.
- Validated the migration with focused tests, targeted build checks, and one full `cd web && bun run check` pass.

## 2026-05-14 frontend baseline reset decision

- The current incremental frontend migration path has been abandoned because the real `web` pages now show widespread
  bugs and the current application state is effectively unusable.
- Repository design truth is now updated to allow `web` to directly adopt the full project shape of
  `web/ai-libs/tdesign-vue-next-starter` as a temporary runtime baseline instead of continuing a shell-only reuse
  strategy.
- The decision is explicitly framed as a control and recovery measure: replacing the broken half-migrated baseline is
  considered safer than continuing to patch scattered defects across the existing page tree.
- The target contract does not change. After the starter baseline is made runnable again, the next implementation stage
  still needs to reconnect backend-driven `menu + route + page + api + permission`, auth, and locale semantics.

## 2026-05-15 Next Step

- Replace the current `web` baseline with a starter full-project baseline first, then stage Graft contract
  reattachment and later optimization work on top of that recovered runtime path.

## 2026-05-14 theme workbench foundation

- Expanded the `setting` store so the frontend can track theme workbench group state, preset selection, custom token
  overrides, and copyable theme configuration without changing the existing layout shell yet.
- Extended the existing `tvision-color` + CSS variable injection path from brand-only output to multi-group token
  output, while keeping the runtime rooted in the same `theme-color` / `theme-mode` attribute mechanism.
- Added pure logic helpers for theme token composition, workbench snapshot export, and future UI integration so the
  later floating toolbar and right-side panel can stay thin.

## 2026-05-14 theme workbench host/state cleanup

- Removed the old mount-time bridge that auto-opened the new workbench from persisted `showSettingPanel` state, so
  legacy state no longer causes the panel to appear by default.
- Narrowed theme workbench visibility semantics in the `setting` store: `showThemeWorkbench` is the intended source of
  truth, while `showSettingPanel` remains only as a compatibility mirror for not-yet-migrated readers.
- Added a `themeWorkbenchRuntimeReady` guard and then moved `web/src/layouts/setting.vue` to the `App` root as the
  single global workbench host, so runtime initialization and dock visibility no longer depend on separate login/admin
  host instances.

## 2026-05-14 theme workbench regression fixes

- Reworked the right-side workbench shell so the drawer width, card grids, and mode-preview sizing now degrade
  responsively instead of forcing three fixed cards into the narrow overview column.
- Restored the dock interaction rhythm expected by the official reference: inactive quick actions stay icon-only,
  while the active action expands into a pill button and reveals its label without clipping the surrounding controls.
- Tightened the dock entry details so the global trigger also defaults to an icon-only state before activation, and the
  quick-action icons now match the right-side group-navigation icons instead of using a separate icon vocabulary.
- Kept both fixes inside the existing `ThemeWorkbenchPanel.vue` and `ThemeWorkbenchDock.vue` shell files so the
  runtime still depends on the same `setting` store state and theme token pipeline.

## 2026-05-14 theme workbench icon and footer cleanup

- Replaced the workbench dock and group-navigation icon names with entries that are actually present in the current
  `tdesign-icons-vue-next` dependency, fixing the blank icon placeholders in the overview, semantic, and font-related
  controls.
- Removed the right-panel floating footer action area, deleted the redundant copy-config action path, and moved the
  reset-theme action into the `元素开关` section so the panel no longer shows suspended bottom buttons over content.
- Trimmed the now-unused copy-config locale strings, store getters, and helper types/functions so the feature surface
  matches the visible UI again.

## 2026-05-14 theme workbench dock centering fix

- Adjusted the floating dock shell so its fixed-position container still centers around the viewport while button pills
  expand and collapse.
- Reworked the active dock-button alignment to keep the icon + label content centered inside each expanded pill instead
  of using a left-aligned layout that made the controls look visually off-center.
- Kept the change local to `ThemeWorkbenchDock.vue`, preserving the existing store actions, group switching semantics,
  and drawer interaction path.

## 2026-05-14 frontend quality-source tightening

- Clarified repository truth so frontend completion follows the documented host Windows Bun CLI chain, while JetBrains
  Inspection, TS suggestion diagnostics, and local spell-check output remain IDE-local guidance unless mirrored by a
  repository rule.
- Tightened `web/eslint.config.js` with `eqeqeq` and a `no-console` policy that blocks `console.log`-style debug
  output while still allowing `console.warn` and `console.error` for operational diagnostics.
- Removed the existing debug-only log calls and one leftover commented-out debug line from the current starter/demo
  `web` pages so the stricter lint rule applies immediately without local exemptions.

## 2026-05-14 frontend logger design documentation

- Updated repository design truth for a planned frontend logger infrastructure before code implementation begins.
- Locked the core model to `LoggerCore + LogEvent + Transport`, with `consola` as the default transport and
  `NoopTransport` as the silent path.
- Fixed the usage boundary so business code only depends on `createLogger`, `child()`, and `withContext()` instead of
  transport-specific APIs.
- Recorded governance requirements for stable `moduleName`, serializable `meta/context`, sensitive-data restrictions,
  explicit separation between logger output and UI messages, no silent swallowing after `logger.error` in `catch`,
  temporary debug lifecycle cleanup, and AI debug-noise control.

## 2026-05-15 commit message hook tightening

- Reused the existing repository `.husky/commit-msg` path instead of adding a new hook system, keeping commit-message
  enforcement aligned with the current `web`-local `commitlint` dependency layout.
- Tightened the minimum Conventional Commit contract by requiring an explicit `scope` in `web/commitlint.config.mjs`.
- Added `scripts/validate-commit-message.mjs` so commit messages now reject literal escaped control text such as
  `\n`, `\t`, and `\r`, forcing automation to emit real multi-line text before `git commit`.
- Validated the slice with one passing sample commit message, one failing missing-scope sample through `commitlint`,
  and one failing escaped-control-text sample through the new script.

## 2026-05-16 restricted-session forced-password-change hookup

- Added a dedicated restricted-session route and route-guard behavior so `must_change_password=true` now blocks business navigation without clearing tokens or forcing logout.
- Kept request semantics unchanged for real auth failures, while explicitly ensuring `AUTH_PASSWORD_CHANGE_REQUIRED` is not treated as a token-invalid path and replayed requests still stop after one refresh attempt.
- Wired the forced-password-change dialog to complete the backend-owned sequence `change-password -> bootstrap(true) -> rebuild routes -> restore navigation`, and hid the restricted route from tabs so the temporary entry does not pollute normal shell navigation.

## 2026-05-15 signals evaluation documentation

- Updated `ai-plan/design/前端架构设计.md` to lock `Pinia` as the only formal shared state layer and to constrain
  any future `signals` exploration to a document-first, `setting/theme`-only boundary.
- Added `ai-plan/public/mvp-extension-path/subtopics/web/design/signals-theme-runtime-evaluation.md` to capture the
  evidence review, the current no-go conclusion, the forbidden domains, and the future admission/exit criteria for a
  smallest-possible `alien-signals` POC.
- Explicitly concluded that the current `theme runtime` does not yet show enough evidence that
  `computed / watch / store action` maintenance has failed, so no POC should begin now.

## 2026-05-15 real auth/bootstrap hookup

- Replaced the `web` mock auth path with real `POST /api/auth/login`, `POST /api/auth/refresh`, and
  `GET /api/auth/bootstrap` integration while preserving the existing starter-shell login page and guard flow.
- Added request-level access-token and locale-header propagation so the frontend now sends `Authorization` and
  `X-Graft-Locale` consistently through the shared Axios wrapper.
- Narrowed the first dynamic-menu slice to bootstrap-returned menus that already have real page implementations in
  `web`, and wired `/users` as the first backend-driven route instead of reusing the starter demo menu tree.
- Validated the slice with one full host Windows Bun `bun run check` pass after adding direct route-transform coverage
  for the bootstrap-menu mapper.

## 2026-05-15 PR #9 review follow-up

- Re-checked the latest PR #9 open threads against local HEAD and kept the current `web` work focused on still-valid
  behavior, contract, and documentation issues instead of stale AI suggestions.
- Kept the existing refresh-based route-guard recovery path, but added the Chinese contract comments required by
  repository governance around dynamic-route initialization, bootstrap recovery, and silent refresh fallback.
- Hardened the shell logout path so router navigation back to `/login` now runs in a `finally` block even if the
  logout request fails after local session cleanup.
- Narrowed the bootstrap route mapper to one explicit `RouteRecordRaw` adaptation boundary instead of the previous
  `as unknown as RouteRecordRaw[]` escape, updated locale-header propagation to replace every underscore in persisted
  locale tags, and normalized the active trace/tracking docs to remove duplicate headings and machine-specific paths.

## 2026-05-15 local git root cleanup

- Removed the extra JetBrains VCS mapping that treated `web/ai-libs/tdesign-vue-next-starter` as a second Git root
  inside the current workspace.
- Reconfirmed that `web/ai-libs/` stays ignored by the main repository and is only a local reference area for starter
  patterns, not an independently managed repository in this project workspace.
- Kept the cleanup scoped to local IDE metadata plus active-topic recovery notes so future recovery does not mistake the
  reference starter tree for repository history owned by `Graft`.

## 2026-05-15 real `/users` page hookup

- Replaced the old starter/profile-style `/users` page content with a minimal real TDesign table that consumes
  `GET /api/users` and shows the current backend-driven user list snapshot.
- Removed the page-local demo constants and chart dataset that previously made `/users` look connected while still
  rendering static starter data.
- Removed the leftover static `/user/index` route module and repointed the header dropdown entry to `/users`, so the
  shell no longer exposes two competing user-page entry paths.
- Revalidated the focused slice with `cd web && bun run typecheck` before the full completion-state check.

## 2026-05-15 login console error cleanup

- Split the frontend API host semantics into two explicit roles: browser requests stay on relative `/api/...` paths in
  proxy mode, while `VITE_API_TARGET` now serves only as the Vite development proxy target or the direct backend host
  when proxy mode is disabled.
- Removed the login-page `vue-i18n` compilation noise by replacing the copyright string's literal `@` token with a
  plain text variant, so the shell no longer trips linked-message parsing on first render.
- Kept the change scoped to the shared request/env layer and locale resources, preserving the current
  `auth + refresh + bootstrap` contract and route-guard flow.

## 2026-05-15 frontend env template alignment

- Stopped tracking the real `web/.env.development` file and aligned the frontend env workflow with the repository's
  existing server-side convention: keep `web/.env.example` in Git as the shared template, while local
  `web/.env.*` runtime files remain ignored.
- Updated the repository README and active web recovery notes so contributors now have one explicit path for local web
  startup and do not need to infer whether machine-specific proxy targets belong in version control.

## 2026-05-15 auth response convergence

- Completed the first frontend-side auth response convergence pass so the shared request layer now consumes the stable
  `AUTH_*` code contract instead of falling back to localized message text.
- Added direct Vitest coverage for `request.ts` and `user` store auth recovery behavior, including
  `AUTH_TOKEN_EXPIRED -> refresh + replay`, `AUTH_TOKEN_INVALID / AUTH_TOKEN_MISSING -> single cleanup exit + login redirect`,
  and the rule that refresh must not recurse on its own failure path.
- Replaced the earlier request-layer dynamic import of `store/index.ts` with an explicit auth session bridge registered
  by the `user` store, removing the Vite dynamic-import warning and keeping request/store auth synchronization explicit.
- Revalidated the slice with focused Vitest + typecheck, then one full host Windows Bun `bun run check` pass with zero
  unresolved warnings.

## 2026-05-15 Follow-up Next Step

- Continue reconnecting the starter shell to the real backend `auth + current user + menu + permission + locale`
  contracts by expanding from the new bootstrap baseline instead of restoring mock auth/menu paths or expanding the
  standalone theme/runtime surface.
- Keep `signals` at the document-only candidate stage unless a future review can prove that `theme runtime`
  maintenance has materially failed under the current `Pinia + computed + composable` approach.
- When the logger slice is scheduled for implementation, land it as a focused frontend infrastructure change first and
  keep business modules on the `createLogger` boundary rather than binding them directly to `consola` or UI feedback.

## 2026-05-15 PR #10 review follow-up

- Applied the remaining CodeRabbit follow-up fixes for the current PR: refreshed the visible copyright year,
  kept the user-page style deep selector on the Stylelint-supported `:deep` syntax, and added an explicit request
  sequence assertion so `AUTH_TOKEN_INVALID / AUTH_TOKEN_MISSING` cannot silently regain a refresh path.
- Applied the missed `greptile-apps[bot]` follow-up fixes for the same PR: moved the `/users` page copy onto i18n
  keys, replaced literal backtick display with semantic `<code>` markup, prevented duplicate client-session cleanup
  when refresh fails on invalid or missing auth state, and blocked `ensureBootstrap` from issuing a second refresh
  after the session has already been cleared.
- Revalidated the updated slice with the required host Windows Bun full frontend chain and kept the change scoped to
  the existing `web` recovery path.

## 2026-05-15 forced-password-change web docs sync

- Recorded the next bounded frontend auth-governance slice before implementation so the recovery point stays aligned
  with the backend contract.
- Fixed repository truth so the future forced-password-change UI must consume backend `login/bootstrap` truth instead
  of guessing from username, default password, or localized message text.
- Fixed the MVP responsibility split so `web` will own the post-login restricted-state modal and refresh-safe
  recovery path, while stronger server-side global interception remains out of scope for this slice.

## 2026-05-16 RBAC permission-consumption foundation

- Started the first frontend RBAC MVP implementation wave by keeping the UI scope on the existing real `/users`
  entrypoint instead of opening `/roles` or a second batch of dynamic menu routes.
- Extended the frontend bootstrap contract so `web` can consume backend-owned `roles` alongside the existing
  `permissions` and `menus` snapshot, avoiding another round of empty-role placeholders or local role inference.
- Added the shared permission-consumption foundation in `web` through permission-store helpers and a global
  `v-permission` directive registration path, so later pages can reuse one UI-level permission surface instead of
  scattering raw `includes()` checks.
- Kept the slice intentionally narrow: the current route/bootstrap mapping stays centered on `/users`, and the wider
  RBAC management UI continues in parallel rather than reopening starter demo menus or role pages.
