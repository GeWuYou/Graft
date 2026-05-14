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

- This update is documentation-only and has not executed any `web` runtime command.
- Direct validation for this change should be limited to consistency checks across `ai-plan/design/前端架构设计.md`,
  this tracking file, and `ai-plan/public/mvp-extension-path/subtopics/web/traces/web-trace.md`.
- After the mainline implementation actually replaces the current frontend baseline, it should validate with host
  Windows Bun using at least:
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe install --force`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run typecheck`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run build`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run check`

## Immediate Next Step

- Let the mainline frontend work replace the current broken incremental shell with a starter full-project baseline
  first, then reattach the real backend auth/menu/permission contracts in a controlled second step without
  reintroducing frontend-only policy.
