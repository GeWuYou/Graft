# User Page Permission Governance Trace

## 2026-05-27 Batch 0 topic initialized and current-state map recorded

- Reused the inherited startup receipt under root `AGENTS.md` for a `web` round inside `graft-multi-agent-loop`.
- Read:
  - `AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/public/README.md`
  - `ai-plan/public/rbac-visibility-governance/README.md`
  - `ai-plan/public/rbac-visibility-governance/todos/rbac-visibility-governance-tracking.md`
  - `ai-plan/public/rbac-visibility-governance/traces/rbac-visibility-governance-trace.md`
  - `ai-plan/design/AI任务追踪与恢复设计.md`
- Confirmed the shared recovery index currently lists no active topic and keeps archived RBAC visibility governance as
  the immediate predecessor for this work.
- Created the new topic document set:
  - `ai-plan/public/user-page-permission-governance/README.md`
  - `ai-plan/public/user-page-permission-governance/todos/user-page-permission-governance-tracking.md`
  - `ai-plan/public/user-page-permission-governance/traces/user-page-permission-governance-trace.md`
- Audited the canonical permission implementation already available in owned frontend scope:
  - `web/src/store/modules/permission.ts` provides bootstrap-backed `hasPermission`, `hasAnyPermission`, and
    `hasAllPermissions`
  - `web/src/app/bootstrap/permission-directive.ts` implements `v-permission` by removing disallowed elements from the
    DOM
  - `web/src/app/bootstrap/route-guards.ts` keeps route visibility on the bootstrap-session path rather than page-local
    permission state
  - `web/src/modules/rbac/pages/index.vue` is the closest current reference page for `v-permission`-first action
    visibility
- Audited `web/src/modules/user/pages/index.vue` and mapped the remaining drift:
  - page-local permission computed booleans:
    - `canCreateUsers`
    - `canUpdateUsers`
    - `canDisableUsers`
    - `canReadUserRoles`
    - `canAssignUserRoles`
    - `canShowOperationColumn`
  - template-level permission visibility still uses those page-local booleans on:
    - header create button
    - empty-state create button
    - row assign-roles button
    - row edit button
  - permission semantics still appear as visible-but-disabled controls on:
    - batch assign-roles action
    - row dropdown `toggle-status`
    - row dropdown `reset-password`
    - row dropdown `delete`
  - operation column visibility is bundled through `canShowOperationColumn` rather than explicit canonical action
    visibility
  - privileged action handlers currently lack local runtime permission guard checks and rely on UI state plus backend
    denial:
    - `openUserDrawer`
    - `toggleUserStatus`
    - `confirmDeleteUser`
    - `handleUserMoreAction`
    - `handleOpenUserRoleDrawer`
    - `openBatchUserRoleDrawer`
    - `submitUserRoleAssignment`
- Identified non-permission disabled state that should remain outside the visibility-convergence work:
  - role assignment drawer controls disabled during loading
  - role assignment submit disabled when there is no effective change
  - batch enable/disable placeholders that are not part of the current permission-governance target
- Decided not to update `ai-plan/public/README.md` in Batch 0.
  - Basis: the recovery index explicitly says it should stay short and list only active topics, but no repository rule
    found in the read scope required initialization-only workers to register an active topic immediately before any
    implementation batch.
  - This can be revisited by the outer loop during archive-closeout if topic indexing needs a final update.
- Kept the round doc-only; no business code was modified.

## 2026-05-27 Batch 1 converged user-page permission visibility and runtime guards

- Reused the inherited startup receipt under root `AGENTS.md` for the Batch 1 `web` round.
- Read and rechecked:
  - `web/AGENTS.md`
  - `ai-plan/public/user-page-permission-governance/README.md`
  - `web/src/modules/user/pages/index.vue`
  - `web/src/modules/rbac/pages/index.vue`
  - `web/src/store/modules/permission.ts`
  - `web/src/app/bootstrap/permission-directive.ts`
- Kept the implementation inside owned scope:
  - `web/src/modules/user/pages/index.vue`
  - `web/src/modules/user/locales/zh-CN.json`
  - `web/src/modules/user/locales/en-US.json`
  - topic tracking and trace docs
- Removed the page-local permission computed wrappers that previously only mirrored permission codes:
  - `canCreateUsers`
  - `canUpdateUsers`
  - `canDisableUsers`
  - `canReadUserRoles`
  - `canAssignUserRoles`
  - `canShowOperationColumn`
- Switched user-page visibility entry points to canonical permission handling:
  - header create button now uses `v-permission="userPermissionCodes.CREATE"`
  - empty-state create button now uses `v-permission="userPermissionCodes.CREATE"`
  - row edit button now uses `v-permission="userPermissionCodes.UPDATE"`
  - row manage-roles button now uses `v-permission="{ allOf: userRoleManagePermissionCodes }"`
  - batch manage-roles button now uses `v-permission="{ allOf: userRoleManagePermissionCodes }"`
- Removed visible-but-disabled permission semantics from batch and row action entry points:
  - batch manage-roles no longer stays visible only to be disabled for missing permission
  - row dropdown option builder now filters out unauthorized actions instead of rendering disabled permission-only
    options
  - row dropdown trigger is hidden when no authorized row actions remain
- Kept disabled state only for non-permission business/loading constraints:
  - role-assignment toolbar still disables during submit/load
  - checkbox group and assignment cards still disable while selection data is not ready
  - assignment submit still depends on effective mutation state rather than permission-only booleans
- Added local runtime permission guards to privileged handlers without introducing a new global abstraction:
  - `openUserDrawer`
  - `openResetPasswordDialog`
  - `toggleUserStatus`
  - `confirmDeleteUser`
  - `handleOpenUserRoleDrawer`
  - `openBatchUserRoleDrawer`
  - `submitUserRoleAssignment`
- Chose the managed-role visibility requirement as `USER_ROLE_READ + USER_ROLE_ASSIGN` to match the page's editable
  role-assignment flow and remove the prior drift where a role-management entry could stay visible but land in a
  permission-disabled editing surface.
- Did not change:
  - API entrypoints
  - permission codes
  - table data shape
  - drawer/dialog business flow
  - unrelated shell, layout, or route behavior

## 2026-05-27 Batch 2 completed regression audit and consistency check

- Reused the inherited startup receipt under root `AGENTS.md` for the Batch 2 `web` round.
- Did not use `graft-multi-agent-batch`; this remained a single-worker audit round.
- Used `graft-task-closeout` style acceptance logic for owned-scope verification, validation, and commit eligibility.
- Verified the worktree was clean before the audit and confirmed Batch 1 had already been committed as:
  - `fae5058` `fix(user-page-permission): align user actions with centralized guards`
- Ran targeted `rg` audits across the owned `user` and `rbac` frontend scope:
  - `rg -n "hasPermission|can(Create|Update|Delete|Assign|Read)|v-permission|disabled:" web/src/modules/user web/src/modules/rbac`
  - `rg -n "handleUserMoreAction|handleOpenUserRoleDrawer|openBatchUserRoleDrawer|submitUserRoleAssignment|hasVisibleUserOperationActions" web/src/modules/user/pages/index.vue`
- Audit conclusions for `web/src/modules/user/pages/index.vue`:
  - no page-local `canCreate/canUpdate/canDelete/canAssign` computed wrappers remain that only mirror permission
    codes for template visibility
  - create, edit, and manage-role entry points remain on `v-permission`
  - batch manage-roles visibility remains on `v-permission` rather than visible-disabled permission semantics
  - row dropdown options are permission-filtered by `userRowMoreOptions(...)` and the dropdown trigger disappears when
    the filtered option set is empty
  - privileged handlers still contain runtime guards:
    - `openUserDrawer`
    - `openResetPasswordDialog`
    - `toggleUserStatus`
    - `confirmDeleteUser`
    - `handleOpenUserRoleDrawer`
    - `openBatchUserRoleDrawer`
    - `submitUserRoleAssignment`
- Audit conclusions for `web/src/modules/rbac/pages/index.vue`:
  - it still matches the topic's intended canonical strategy for critical action visibility, using `v-permission` on
    create, edit, and permission-management entry points
  - remaining computed permission state and disabled controls in RBAC stay unchanged and are not a user-page
    regression; Batch 2 did not widen scope into a cross-page refactor
- No code fix was required in Batch 2.
- Updated only topic tracking/trace docs to record the regression-audit result.
- Validation passed:
  - `cd web && bun run check`
  - `git diff --check`
