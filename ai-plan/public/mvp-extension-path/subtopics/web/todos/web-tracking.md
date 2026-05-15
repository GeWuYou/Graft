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
- The latest frontend governance follow-up now also closes the quality-chain warning cleanup:
  `web/vite.config.ts` only mounts `vite-plugin-mock` in mock/development modes, release/test builds use explicit
  vendor chunk boundaries for `tdesign` / `tdesign-icons` / `echarts` / `vue` / shared utils, and the current starter
  baseline accepts a `chunkSizeWarningLimit` aligned to that temporary full-TDesign shell so the host Windows Bun
  `bun run check` path can finish without warning output.
- The same slice also refreshed the lowest-risk runtime dependency update in the current tree by moving `axios` to
  `^1.16.1`, while leaving cross-major framework, UI-library, and tooling upgrades for later dedicated validation
  slices instead of mixing them into this warning-cleanup pass.
- The latest frontend governance tightening now also makes the quality-source boundary explicit:
  repository completion and merge decisions follow the documented CLI chain headed by host Windows Bun `bun run check`,
  while JetBrains Inspection, TS suggestion diagnostics, and local spell-check output remain IDE-local guidance unless
  a matching rule is promoted into the repository CLI/tooling contract.
- The same slice also raises one concrete CLI rule instead of relying on IDE yellow warnings:
  `web/eslint.config.js` now rejects `console.log`-style debug output while still allowing `console.warn` and
  `console.error`, and the current starter/demo pages remove the existing debug-only log calls so the stricter lint
  policy can pass without hidden local exceptions.
- The planned frontend logger infrastructure is now documented before implementation:
  it follows a `LoggerCore + LogEvent + Transport` model, defaults to a `consola` transport, supports a silent
  `NoopTransport`, and requires business code to depend on `createLogger` plus `child()` / `withContext()` instead of
  transport-specific APIs.
- The same planned logger slice also establishes governance boundaries up front:
  stable `moduleName` ownership, serializable `meta/context`, explicit sensitive-data restrictions, a strict
  separation between logger output and UI message responsibility, no silent swallowing after `logger.error` in `catch`,
  temporary debug lifecycle cleanup rules, and AI debug-noise limits for generated frontend code.
- The latest commit-governance slice now lands the minimum repository hook enforcement for commit messages without
  touching the active backend line:
  the existing `.husky/commit-msg` hook still reuses `web`'s local `commitlint`, `scope` is now mandatory in the
  Conventional Commit header, and a new repository script rejects literal escaped control text such as `\n`, `\t`,
  and `\r` inside commit messages so automation must emit real multi-line text.

## Active Risks

- Future frontend work must continue to align with backend-driven menus, permissions, and shared i18n contracts instead
  of drifting into a long-lived frontend-only policy after the starter baseline is copied in.
- The temporary baseline will likely bring starter demo routes, mock data flows, and frontend-only assumptions back
  into the tree, so the reattachment plan must remove or fence them quickly.
- The current shell-level bug density means the repository has less confidence in any partial migration artifact that
  remains in `web`, so mainline implementation needs a clear replacement boundary instead of mixed old/new pages.
- Mixed WSL Bun and host Windows Bun dependency installs can still break Windows IDE startup until the working tree is
  reinstalled with host Windows Bun after this rule change lands.
- The warning-cleanup slice now reaches a zero-warning completion state, but the current vendor-size strategy still
  depends on the temporary full-TDesign starter baseline and a raised `chunkSizeWarningLimit`, so deeper bundle-size
  optimization should be treated as a future performance task instead of silently regressing back into warning debt.
- IDE 问题面板仍可能继续提示未使用导出、JSDoc、拼写、commented-out code 一类本地检查项；后续若团队希望把其中某类
  提升为正式阻塞规则，必须先提供等价 CLI 校验入口，而不是把 JetBrains 专属 Inspection 直接当作仓库唯一标准。
- If the logger plan is implemented without the documented `moduleName` / serializable-context / sensitive-data
  boundaries, frontend debug output could quickly drift into inconsistent module naming, unstructured payloads, and
  accidental leakage of user or credential data.

## Latest Validation

- The frontend governance baseline now treats host Windows Bun `bun run check` as the required completion-state
  validation entrypoint for `web`.
- The latest implementation validation snapshot is:
  - `bun outdated`
  - `bun add axios@1.16.1`
  - `bun run test:run -- --reporter=hanging-process`
  - `bun run test:run`
  - `bun run check`
- `bun run check` 当前通过，`format:check`、`typecheck`、`lint`、`stylelint`、`test:run`、`build` 均无未处理 warning。
- The current build-cleanup strategy is explicit:
  - `@vueuse/core` pure annotation noise is filtered by exact source match in `vite.config.ts`, so the repository does
    not suppress unrelated Rollup warnings.
  - `vite-plugin-mock` is only mounted in mock/development modes, preventing Vitest from leaving watcher handles open
    in completion-state validation.
  - Chunk-warning output is closed by stable vendor chunk boundaries plus the current `1600` threshold that matches
    the temporary full-TDesign starter baseline; future bundle-size optimization remains a separate follow-up concern.
- The latest governance-tightening validation snapshot additionally requires:
  - host Windows Bun `bun run check`
  - no new `console.log`-style debug output under `web/src`
- The latest commit-governance validation snapshot is:
  - `cd web && ./node_modules/.bin/commitlint --config commitlint.config.mjs --edit <valid-message-file>`
  - `cd web && ./node_modules/.bin/commitlint --config commitlint.config.mjs --edit <invalid-scope-message-file>` and expect `scope-empty`
  - `node scripts/validate-commit-message.mjs <invalid-escaped-message-file>` and expect rejection of literal `\n` / `\t` / `\r`

## Immediate Next Step

- Let the mainline frontend work replace the current broken incremental shell with a starter full-project baseline
  first, then reattach the real backend auth/menu/permission contracts in a controlled second step without
  reintroducing frontend-only policy.
- For the theme workbench follow-up, continue improving grouped token editors and layout-preview fidelity on top of the
  current `setting store + token/runtime底座 + dock/panel 壳层` path.
- Do not fork a second theme system outside the existing `tvision-color + CSS variables + Pinia persisted state`
  path, and avoid adding another shell-level host or a parallel visible-state flag.
- Keep future `web` slices on the host Windows Bun `bun run check` completion gate, and treat any later bundle-size
  or dependency-major upgrade work as dedicated follow-up tasks instead of reintroducing warning noise into the
  completion path.
- When the frontend logger slice moves from documentation to implementation, keep it infrastructure-scoped first:
  add the core logger path and governance boundary without coupling it to UI message flows or a broader remote logging
  platform in the same change.
- If the repository later wants stronger commit-message semantics, extend the same `commit-msg` hook path with
  body-bullet verb checks or summary-language checks instead of introducing a second parallel hook stack.
