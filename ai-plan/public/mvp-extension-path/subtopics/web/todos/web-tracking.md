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

- `web` has now switched its visible shell direction from a lightweight custom admin page toward a starter-led
  TDesign admin shell, while keeping Graft's `menu + route + page + api + permission` contract in place.
- `AuthLayout` and `BasicLayout` are now backed by Graft-owned shell components that reuse starter-style side-nav,
  top-nav, page-header, settings-panel, theme switching, auth/result visual language, and global shell tokens.
- The current shell state is intentionally narrow: theme mode, sidebar collapsed state, and settings panel visibility
  live in a dedicated shell store, while auth, permissions, locale, and navigation truth still stay in Graft stores.
- The dashboard page has been replaced with a starter-style local example slice under `web/src/pages/dashboard/`,
  using page-local sample data and components instead of importing starter business stores or mock APIs.
- Static routing, mock auth, route guards, locale propagation, and navigation metadata still preserve the future
  backend-driven menu and permission path instead of introducing a parallel frontend-only access model.
- Frontend command execution truth remains explicit: in WSL-based development, all `web` install, validation, build,
  preview, and dev commands must use the configured host Windows Bun, and WSL Bun must not refresh `web/node_modules`.

## Active Risks

- Future frontend work must continue to align with backend-driven menus, permissions, and shared i18n contracts instead
  of drifting into frontend-only policy.
- The current shell still relies on staged/mock auth behavior until the real auth + RBAC path lands on the backend.
- The starter-style shell has not yet absorbed list/form/detail/result example pages beyond the dashboard slice, so AI
  page generation still lacks a full in-repo CRUD page baseline.
- The login background PNG assets are currently large, and the production build still reports oversized chunks; both
  should be reduced in a later optimization slice without weakening the current shell contract.
- Mixed WSL Bun and host Windows Bun dependency installs can still break Windows IDE startup until the working tree is
  reinstalled with host Windows Bun after this rule change lands.

## Latest Validation

- Historical frontend validation commands before the subtopic split are preserved in the parent-topic archive.
- The latest starter-led shell migration should validate with host Windows Bun using:
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run test:run -- src/layouts/BasicLayout.test.ts src/pages/LoginPage.test.ts src/pages/UnauthorizedPage.test.ts src/pages/NotFoundPage.test.ts src/app/route-guards.test.ts src/pages/DashboardPage.test.ts`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run typecheck`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run build`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run check`
- The environment-rule fix should additionally validate with host Windows Bun:
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe install --force`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run dev`
  - `C:\\Users\\gewuyou\\.bun\\bin\\bun.exe run check`

## Immediate Next Step

- Keep the starter-led shell stable, continue migrating list/form/detail page baselines into `web`, then wire the real
  backend auth/menu/permission contracts into the existing `menu + route + page + api + permission` path without
  reintroducing frontend-only policy.
