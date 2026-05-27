# User Page Permission Governance

## Status

- Topic: `user-page-permission-governance`
- Status: `archived`
- Branch: `feat/wt-rbac-further-development`
- Task class: `web`

## Goal

Close the remaining user-management page permission drift inside `web/src/modules/user/pages/index.vue` by aligning it
with the existing canonical frontend permission pattern:

- `bootstrap permissions -> permission store -> v-permission visibility -> runtime action guard`

This topic is implementation-convergence only. It must not change backend contracts, permission codes, or user
management business flows.

## Hard Constraints

- No new backend API.
- No new or modified backend permission code.
- No capability snapshot, denial reason, or observability contract.
- No global UI refactor.
- No business-behavior change for user management.
- If a required permission code is missing, record the gap as risk or future topic instead of widening scope.

## Scope

- `ai-plan/public/user-page-permission-governance/**`
- `ai-plan/public/README.md`
- `web/src/modules/user/**`
- `web/src/modules/rbac/**`
- `web/src/store/modules/permission.ts`
- `web/src/utils/route/**`
- `web/src/app/bootstrap/**`
- bounded tests/types only if directly required

## Forbidden Scope

- `server/**`
- OpenAPI/generated contract files
- RBAC backend permission registry
- unrelated layout/theme/i18n refactors

## Canonical Frontend Permission Path

- `web/src/store/modules/permission.ts`
  - canonical permission truth comes from bootstrap snapshot and `permissionStore.hasPermission/hasAnyPermission/hasAllPermissions`
- `web/src/app/bootstrap/permission-directive.ts`
  - canonical element visibility directive is `v-permission`
  - unsupported permission state removes the element from DOM rather than showing a disabled control
- `web/src/app/bootstrap/route-guards.ts`
  - route bootstrap and session recovery are shell-owned
  - route access depends on bootstrap menus and mounted runtime routes, not page-local booleans
- `web/src/modules/rbac/pages/index.vue`
  - current owned reference page for critical action visibility
  - create, edit, and assign-permission entries use `v-permission`
  - disabled state remains only for non-permission business constraints such as built-in role protection or unchanged selection state

## Batch 0 Current-State Map

### User Page Permission Drift In `web/src/modules/user/pages/index.vue`

- Page-local permission computed values:
  - `canCreateUsers`
  - `canUpdateUsers`
  - `canDisableUsers`
  - `canReadUserRoles`
  - `canAssignUserRoles`
  - `canShowOperationColumn`
- Template-level permission visibility currently uses page-local booleans instead of `v-permission`:
  - create button in page header
  - create button in empty state
  - edit button in operation column
  - assign-roles button in operation column
- Template-level permission semantics currently drift into disabled controls:
  - batch assign-roles button stays visible and uses `:disabled="!canAssignUserRoles"`
  - row dropdown options stay visible and use `disabled: !canDisableUsers.value` or `disabled: !canUpdateUsers.value`
- Table operation visibility is still governed by page-local `canShowOperationColumn`, which bundles visibility semantics
  into page code instead of following the element-level `v-permission` path used on RBAC pages.
- Action functions do not currently include explicit permission guards:
  - `openUserDrawer`
  - `handleUserMoreAction`
  - `toggleUserStatus`
  - `confirmDeleteUser`
  - `handleOpenUserRoleDrawer`
  - `openBatchUserRoleDrawer`
  - `submitUserRoleAssignment`
  - they rely on UI state and backend API denial rather than a centralized runtime guard helper
- Disabled states that appear business-driven and should survive Batch 1:
  - role assignment drawer selection toggles disabled during loading or before selection is ready
  - role assignment submit disabled when no effective role mutation exists
  - batch enable/disable buttons are disabled placeholders and are not permission-governance targets in this topic

### Intended Batch 1 Convergence

- Prefer `v-permission` for create, edit, manage-roles, and row/batch action entry visibility.
- Keep disabled state only when it expresses business state rather than missing permission.
- Retain or add minimal runtime guard checks for privileged actions that can still be triggered outside direct template
  visibility.
- Avoid changing table data shape, API calls, modal flows, or user-management copy.

## Decision Record

- Batch 0 initializes the topic docs and records the current-state map only.
- Batch 0 does not update `ai-plan/public/README.md` yet.
  - Basis: the current recovery index says it should stay short and list only active topics, but the provided batch
    contract did not require immediate registration and no stronger repository rule was found that initialization alone
    must edit the shared active-topic index before implementation starts.
  - The index may be updated in the archive-closeout batch if the outer loop chooses that path under the existing topic
    governance pattern.

## Planned Batches

1. Batch 0: topic initialization and current-state map.
2. Batch 1: user page permission implementation.
3. Batch 2: regression audit and consistency check.
4. Batch 3: archive-ready closeout.

## Final Archive Record

- Status: `archived`
- Archive reason: the topic completed its planned frontend-only convergence loop and removed the remaining
  user-management page permission drift without requiring backend, OpenAPI, or permission-code changes.
- Final result:
  - `web/src/modules/user/pages/index.vue` now uses the canonical permission path for user action visibility:
    `bootstrap permissions -> permission store -> v-permission visibility -> runtime action guard`
  - header, empty-state, row, and batch action entry points no longer depend on page-local permission computed wrappers
  - permission-only visible-disabled semantics were removed from the user-page dropdown and batch role-assignment entry
    points
  - disabled state remains only for business/loading constraints instead of missing-permission presentation
  - privileged user actions retain local runtime guard checks so template visibility is not the only protection layer
- Final validation summary:
  - `git status --short`
  - `git branch --show-current`
  - `cd web && bun run check`
  - `git diff --check`
- Remaining risks:
  - `ROLE_PERMISSION_MANAGE` remains a frontend alias of `role.permission.assign` in the RBAC module and should be
    handled by a future frontend permission-code cleanup topic rather than reopened here
  - if a future user-management capability needs a backend permission code that does not already exist, that gap should
    be handled by a separate RBAC contract topic
- Next-session prompt: `No continuation required.`
