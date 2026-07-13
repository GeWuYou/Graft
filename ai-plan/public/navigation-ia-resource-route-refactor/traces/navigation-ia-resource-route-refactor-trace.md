# Navigation IA And Resource Route Refactor Trace

## 2026-07-13 Batch State Reconciliation

- Reconciled the active tracking file after PR review: `metadata-driven-sidebar-navigation-refinement` is recorded in both the completed checklist and `completed_batches`, preserving the topic's single recovery state.

## 2026-07-13 Metadata-Driven Sidebar Navigation Refinement

- Replaced the shell's path-based wide-table sidebar-motion allowlist with route metadata: paged list routes derive the mode from `pageKind` and `pageSurface`, while exceptional dense non-list tables declare `sidebarMotion: 'wide-table'` at their bootstrap registration.
- Made global menu search prefer an entry's canonical `navigationTargetPath`, preventing internal bootstrap menu-code paths from becoming a second navigation target.
- Updated shell, route registration, and menu-search coverage with the corresponding frontend governance rules.

## 2026-07-12 IA-Aligned UI Route Policy

- Product decision superseded the previous resource-oriented UI route policy: all visible menu URLs now mirror their IA domain using `/<domain>/<resource>`.
- HTTP API paths remain domain-resource contracts and are not migrated.
- The Runtime page is renamed to Service Status and the Module Runtime menu is renamed to Modules; technical module-runtime snapshot contracts remain unchanged.

## 2026-07-12 IA-Aligned Route Migration

- Updated the navigation authority and `graft-navigation-route-governance` so every visible menu entry uses its canonical IA prefix.
- Migrated server menu/bootstrap values and web route, deep-link, tab, breadcrumb, authentication-return and dashboard fixtures to `/applications/**`, `/infrastructure/**`, `/observability/**`, `/security/**`, and `/platform/**`; no UI aliases or redirects were added.
- `cd server && go run ./cmd/graft validate backend` passed. `cd web && bun run check` passed all static gates and 1,270 tests, but one unrelated container-detail mount-copy test failed once; its isolated 101-test retry passed. Keep the topic active until a stable full-Web retry completes.

## 2026-07-12 Navigation Design, Topic, And Skill

- Work Intake classified the navigation refactor as long-running, with a design and active topic but no roadmap or ADR.
- Established `ai-plan/design/architecture/导航与资源路由信息架构规范.md` as the repository authority for seven navigation domains and stable resource-oriented UI routes.
- Added `graft-navigation-route-governance`; unresolved placement or resource boundary now blocks menu and UI route edits until the user decides.
- Preserved the rule that UI route migration has no aliases or redirects and excludes HTTP API endpoint changes.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill"],
  "pending_batches": [
    "backend-navigation-contract",
    "frontend-navigation-route-migration",
    "cross-boundary-validation-closeout"
  ],
  "current_batch": "navigation-design-topic-and-skill",
  "next_batch": "backend-navigation-contract",
  "closeout_status": "batch-1-complete"
}

## 2026-07-12 Backend Navigation Contract

- Added explicit `code`, `parent_code`, and `kind` graph semantics to the backend menu registry; bootstrap validates malformed graphs before projection and prunes empty domain groups after permission/config filtering.
- Registered the seven canonical domain groups in core and re-parented current menu entries without deriving hierarchy from UI paths.
- Migrated menu UI paths to the design authority, including `/projects`, `/containers`, `/system/**`, `/users`, `/roles`, `/permissions`, `/scheduled-tasks`, `/system-config`, and `/announcements`; HTTP API URIs remain unchanged.
- Extended the OpenAPI bootstrap menu schema and generated server/web contract consumers with explicit navigation metadata.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract"],
  "pending_batches": ["frontend-navigation-route-migration", "cross-boundary-validation-closeout"],
  "current_batch": "backend-navigation-contract",
  "next_batch": "frontend-navigation-route-migration",
  "closeout_status": "batch-2-complete"
}
```

## 2026-07-12 Frontend Navigation Route Migration

- Split Web navigation graph construction from Vue Router registration: groups are graph-only nodes, entries register only their canonical resource routes, and no path ancestry fallback remains.
- Migrated Web route constants and bootstrap registrations to `/projects/**`, `/containers/**`, `/system/**`, `/users`, `/roles`, `/permissions`, `/scheduled-tasks`, `/system-config`, and `/announcements`; the Notification Center remains a global menu-external route.
- Added root locale ownership for the seven domain titles and focused graph/routing coverage. Persisted removed legacy URLs are rejected by the existing route restoration validity check rather than aliased.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract", "frontend-navigation-route-migration"],
  "pending_batches": ["cross-boundary-validation-closeout"],
  "current_batch": "frontend-navigation-route-migration",
  "next_batch": "cross-boundary-validation-closeout",
  "closeout_status": "batch-3-complete"
}
```

## 2026-07-12 Cross-Boundary Validation Closeout

- Verified the backend bootstrap/OpenAPI/Web schema contract for `code`, `parent_code`, `kind`, group nodes without paths, and entry nodes with canonical resource paths.
- Repaired the Web shell gap: bootstrap routes now carry explicit localized navigation ancestors; breadcrumbs and tab titles consume that graph context without path-based ancestry inference. Global Project, Container, and announcement detail routes declare their bootstrap parent resource explicitly; Notification Center remains menu-external.
- Updated affected route contract assertions to canonical `/containers`, `/system/**`, `/scheduled-tasks`, `/system-config`, and `/my-announcements` paths. Active registration code has no `/ops` or `/server` UI hierarchy.
- Validation passed: `cd server && go run ./cmd/graft validate backend`; focused Web navigation suite (28 tests); Web typecheck, ESLint, Stylelint, OpenAPI frontend governance, i18n governance, production Vite build; `git diff --check`; and AI plan/governance checks.
- Browser evidence: authenticated shell bootstrap at `http://172.21.235.129:3002` rendered Application, Infrastructure, Observability, Security, and Platform only; Build and Resources were absent because empty. Artifacts: `.ai/artifacts/browser/navigation-ia-auth-closeout`.
- At this batch's closeout, `bun run hygiene:check` reported the unused `BLANK_LAYOUT` export; Batch 5 removed it. The full Web suite retains intermittent Monaco RAF timeout and container log-stream disconnect assertion failures. The route-contract failures discovered during this batch were repaired and pass in the focused suite.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract", "frontend-navigation-route-migration", "cross-boundary-validation-closeout"],
  "pending_batches": [],
  "current_batch": "cross-boundary-validation-closeout",
  "next_batch": null,
  "closeout_status": "batch-4-complete-pending-archive-readiness"
}
```

## 2026-07-12 Web Hygiene And Final Validation

- Removed the unused `BLANK_LAYOUT` route export and its wholly unreferenced `layouts/blank.vue` asset after a repository-wide reference and shared-asset registry preflight. The `route-bootstrap-utils` registry entry remains valid because the shared route utility directory remains in use; no registry update is required.
- Full Web validation ran through all static checks and reached the test suite, where one existing flaky Monaco RAF test timed out: `src/modules/project/shared/project-monaco.test.ts > resolves relayout after the scheduled animation frame runs` (1 failed, 1,271 passed). An isolated retry passed all 11 tests, confirming this batch does not introduce the failure; therefore `bun run check` is not claimed as passed.
- Focused route tests passed (4 files, 11 tests). Shared asset registry and AI governance validation passed.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract", "frontend-navigation-route-migration", "cross-boundary-validation-closeout", "web-hygiene-and-final-validation"],
  "pending_batches": [],
  "current_batch": "web-hygiene-and-final-validation",
  "next_batch": null,
  "closeout_status": "batch-5-complete-pending-archive-readiness"
}
```

## 2026-07-12 Final Archive-Readiness Retry Blocked

- After commit `0bb2910c`, reran the full Web completion entrypoint: `cd web && bun run check`.
- Format, typecheck, OpenAPI frontend governance, i18n governance, ESLint, Stylelint, hygiene, and 1,271 tests passed before the test runner ended with one failure.
- Exact failing test: `src/modules/project/shared/project-monaco.test.ts > project-monaco relayout bridge > resolves relayout after the scheduled animation frame runs`.
- Exact failure: `Error: Test timed out in 20000ms.` at `src/modules/project/shared/project-monaco.test.ts:121:3`; final runner summary was `1 failed | 192 passed (193)` test files and `1 failed | 1271 passed (1272)` tests.
- The failure matches the pre-existing Monaco RAF flake recorded in Batch 5. This archive batch did not modify that module, so it remains outside the safe repair scope. The active topic must not move to `ai-plan/public/archive/` until the full Web check passes or a later, explicitly authorized owner resolves the validation policy.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract", "frontend-navigation-route-migration", "cross-boundary-validation-closeout", "web-hygiene-and-final-validation"],
  "pending_batches": [],
  "current_batch": "navigation-final-validation-and-archive",
  "next_batch": null,
  "closeout_status": "blocked-validation-failed"
}
```
