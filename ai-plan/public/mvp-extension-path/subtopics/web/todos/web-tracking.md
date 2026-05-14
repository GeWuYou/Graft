# MVP Extension Path Web Tracking

## Subtopic

- Parent Topic: `mvp-extension-path`
- Subtopic: `web`
- Scope: `web` admin shell, route/menu/page/api/permission frontend path, i18n UI surface, tests, and frontend
  governance/toolchain follow-up

## Goal

- Keep frontend recovery material separate from backend iteration while preserving the parent `mvp-extension-path`
  topic as the default MVP entrypoint.

## Current Recovery Point

- The user has decided to abandon the current incremental frontend migration path because the current `web` pages now
  contain widespread bugs and are effectively unusable.
- The active frontend direction is now to let `web` directly adopt the full project shape of
  `web/ai-libs/tdesign-vue-next-starter` as a temporary runtime baseline, instead of continuing the current
  shell-only migration strategy.
- This temporary baseline reset is documented as a controllability decision: replacing the broken half-migrated state
  is safer than continuing to patch scattered page defects on top of an unstable shell.
- The baseline reset does not change Graft's target contract. After the starter full-project baseline is running
  again, the next phase must still reattach backend-driven `menu + route + page + api + permission` semantics,
  locale propagation, and shared auth/permission boundaries.
- Frontend command execution truth remains explicit: in WSL-based development, all `web` install, validation, build,
  preview, and dev commands must use the configured host Windows Bun, and WSL Bun must not refresh `web/node_modules`.
- Theme workbench host/state cleanup now treats `showThemeWorkbench` as the single intended visible-state source in the
  `setting` store, while `showSettingPanel` remains only as a compatibility mirror for legacy reads.
- `web/src/layouts/setting.vue` is now mounted once from `web/src/App.vue` as the global workbench host. Dock display
  follows the current route so login pages no longer need their own host instance.
- The latest frontend slice continues aligning the theme workbench with the official TDesign Starter presentation,
  tightening visual hierarchy, spacing density, and interaction rhythm across the dock, right-side panel, and
  configuration editing area while still keeping one `tvision-color + CSS variables + Pinia persisted state` theme
  path instead of forking a second theme system.
- The theme workbench follow-up now also closes two visual-regression gaps from that alignment pass:
  the right-side panel uses responsive drawer/card sizing so the mode cards no longer collapse into one cramped row,
  and the bottom dock restores the active-pill expansion pattern so selected quick actions can reveal their labels.
- The dock entry contract is now tighter as well:
  the global “自定义主题” trigger stays icon-only by default and expands only after activation,
  while the bottom quick-action icons reuse the same icon language as the right-side group navigation.
- The latest fix slice corrects icon regressions in the theme workbench by switching the dock and group navigation to
  icon names that exist in the current `tdesign-icons-vue-next` package, so the overview, semantic, and font entries
  no longer render blank placeholders.
- The same slice also removes the floating footer action area from the right-side panel:
  the redundant “复制完整配置” action and its copy pipeline are deleted, and “恢复默认主题” now lives directly under
  the `元素开关` block to keep the action near the configuration it resets.
- The latest dock-alignment follow-up corrects a remaining visual-centering issue in the floating toolbar:
  the bottom dock keeps its overall center anchored while active pills expand, and the active icon + label content is
  now centered within each expanded button instead of left-biased.

## Active Risks

- Future frontend work must continue to align with backend-driven menus, permissions, and shared i18n contracts instead
  of drifting into a long-lived frontend-only policy after the starter baseline is copied in.
- The temporary baseline will likely bring starter demo routes, mock data flows, and frontend-only assumptions back
  into the tree, so the reattachment plan must remove or fence them quickly.
- The current shell-level bug density means the repository has less confidence in any partial migration artifact that
  remains in `web`, so mainline implementation needs a clear replacement boundary instead of mixed old/new pages.
- Mixed WSL Bun and host Windows Bun dependency installs can still break Windows IDE startup until the working tree is
  reinstalled with host Windows Bun after this rule change lands.

## Latest Validation

- The official-alignment slice for the theme workbench validated with host Windows Bun using:
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run typecheck`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run build`
- The build still emits existing non-blocking warnings from `@vueuse/core` pure annotations and large Vite chunks, but
  the validation completed successfully.
- The icon and footer cleanup follow-up should again validate with host Windows Bun using:
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run typecheck`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run build`
- The dock-centering follow-up should validate with host Windows Bun using:
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run typecheck`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run build`
- After the mainline implementation actually replaces the current frontend baseline, it should still validate with host
  Windows Bun using at least:
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe install --force`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run typecheck`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run build`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run check`

## Immediate Next Step

- Let the mainline frontend work replace the current broken incremental shell with a starter full-project baseline
  first, then reattach the real backend auth/menu/permission contracts in a controlled second step without
  reintroducing frontend-only policy.
- For the theme workbench follow-up, continue improving grouped token editors and layout-preview fidelity on top of the
  current `setting store + token/runtime底座 + dock/panel 壳层` path.
- Do not fork a second theme system outside the existing `tvision-color + CSS variables + Pinia persisted state`
  path, and avoid adding another shell-level host or a parallel visible-state flag.
