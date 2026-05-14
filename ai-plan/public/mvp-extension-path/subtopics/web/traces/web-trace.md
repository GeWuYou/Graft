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

## Next Step

- Continue the workbench slice by improving panel behavior, layout preview fidelity, and token-editor ergonomics
  without reintroducing a second host component or another parallel visibility flag.
