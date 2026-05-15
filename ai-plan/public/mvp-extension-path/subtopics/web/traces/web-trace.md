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

## Next Step

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

## Next Step

- Continue the workbench slice by improving grouped token-editor ergonomics and layout-preview fidelity without
  reintroducing a second host component or another parallel visibility flag.
- When the logger slice is scheduled for implementation, land it as a focused frontend infrastructure change first and
  keep business modules on the `createLogger` boundary rather than binding them directly to `consola` or UI feedback.
