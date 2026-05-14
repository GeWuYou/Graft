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

## Next Step

- Continue frontend iteration against the real backend auth/menu/permission contracts, then optimize large login
  background assets and current build chunk warnings without undoing the starter-aligned shell structure.
