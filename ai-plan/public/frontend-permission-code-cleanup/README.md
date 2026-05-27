# Frontend Permission Code Cleanup

## Status

- Topic: `frontend-permission-code-cleanup`
- Status: `archived`
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-rbac-further-development`
- Branch: `feat/wt-rbac-further-development`
- Task class: `web`
- Loop mode: `topic-completion-loop`
- Archive date: `2026-05-27`

## Goal

Close the remaining frontend-only permission-code naming drift inside owned `web` scope so permission-code usage aligns
with the existing backend canonical values.

Primary target:

- remove the historical frontend alias `ROLE_PERMISSION_MANAGE`
- converge all equivalent frontend usage onto the canonical permission code `role.permission.assign`
- keep current page visibility and action behavior unchanged

This topic is permission-code governance only. It must not widen into backend contract, OpenAPI, or permission-system
redesign work.

## Hard Constraints

- Do not modify backend permission codes.
- Do not add backend permissions.
- Do not modify OpenAPI or generated contracts.
- Do not introduce capability snapshot or denial-reason semantics.
- Do not add a new alias compatibility layer.
- Keep all edits inside the frontend permission-code usage layer.

## Scope

- `ai-plan/public/frontend-permission-code-cleanup/**`
- `ai-plan/public/README.md`
- `web/src/modules/rbac/**`
- `web/src/modules/user/**`
- `web/src/store/modules/permission.ts`
- `web/src/constants/**`
- `web/src/types/**`
- bounded tests/types only if directly required

## Forbidden Scope

- `server/**`
- OpenAPI/generated contracts
- RBAC backend permission registry / seed / migration
- auth contract
- unrelated refactor or reformat
- UI redesign
- route architecture rewrite

## Recovery Sources

- `ai-plan/public/README.md`
- archived `rbac-visibility-governance`
- archived `user-page-permission-governance`
- current RBAC frontend implementation

## Canonical Frontend Permission Path

- `web/src/store/modules/permission.ts`
  - canonical permission truth comes from the bootstrap snapshot
  - stable helpers are `hasPermission`, `hasAnyPermission`, and `hasAllPermissions`
- `web/src/app/bootstrap/permission-directive.ts`
  - canonical element visibility primitive is `v-permission`
- module permission contracts
  - `web/src/modules/rbac/contract/permissions.ts`
  - `web/src/modules/user/contract/permissions.ts`
  - frontend permission-code literals should converge here instead of being duplicated in pages

## Batch 0 Audit

### `ROLE_PERMISSION_MANAGE` Origin

- Source file:
  - `web/src/modules/rbac/contract/permissions.ts`
- Current definition:
  - `ROLE_PERMISSION_ASSIGN: 'role.permission.assign'`
  - `ROLE_PERMISSION_MANAGE: 'role.permission.assign'`
- Current conclusion:
  - `ROLE_PERMISSION_MANAGE` is a frontend-only historical alias
  - it does not represent a second backend permission code
  - its canonical target is `RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN`

### Canonical Permission-Code Map In Owned Scope

- RBAC permission codes:
  - `ROLE_READ` -> `role.read`
  - `ROLE_CREATE` -> `role.create`
  - `ROLE_UPDATE` -> `role.update`
  - `ROLE_STATUS_UPDATE` -> `role.status.update`
  - `ROLE_DELETE` -> `role.delete`
  - `ROLE_PERMISSION_ASSIGN` -> `role.permission.assign`
  - `PERMISSION_READ` -> `permission.read`
  - `USER_ROLE_READ` -> `user.role.read`
  - `USER_ROLE_ASSIGN` -> `user.role.assign`
- User permission codes:
  - `READ` -> `user.read`
  - `CREATE` -> `user.create`
  - `UPDATE` -> `user.update`
  - `DISABLE` -> `user.disable`
  - `SESSION_READ` -> `user.session.read`
  - `SESSION_REVOKE` -> `user.session.revoke`

### Alias Drift Map

- confirmed alias drift:
  - `RBAC_PERMISSION_CODE.ROLE_PERMISSION_MANAGE`
    - actual value: `role.permission.assign`
    - canonical replacement: `RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN`
    - drift type: duplicate symbolic name for one canonical permission code
- no second owned-scope alias was found in permission contract definitions during Batch 0 audit
- no backend-value mismatch was found in owned-scope permission literals during Batch 0 audit

### Relevant Usage Points

- `hasPermission` usage relevant to this topic:
  - `web/src/modules/rbac/pages/index.vue`
    - `canCreateRoles`
    - `canDeleteRoles`
    - `canToggleRoleStatus`
    - `canReadPermissions`
    - `canAssignPermissions`
  - `web/src/modules/user/pages/index.vue`
    - user create/update/disable action guards
    - role-assignment action guards
- `v-permission` usage relevant to this topic:
  - `web/src/modules/rbac/pages/index.vue`
    - create action
    - assign-permission action
    - edit action
  - `web/src/modules/user/pages/index.vue`
    - create action
    - assign-role action
    - edit action
- owned-scope alias usage found in Batch 0:
  - `web/src/modules/rbac/pages/index.vue`
    - `v-permission="{ allOf: [permissionCodes.PERMISSION_READ, permissionCodes.ROLE_PERMISSION_MANAGE] }"`
    - `permissionStore.hasPermission(permissionCodes.ROLE_PERMISSION_MANAGE)`
    - `permissionStore.hasAnyPermission([... permissionCodes.ROLE_PERMISSION_MANAGE])`

### Batch 1 Decision Boundaries

- direct-delete candidates:
  - `RBAC_PERMISSION_CODE.ROLE_PERMISSION_MANAGE` definition after all owned-scope references switch to
    `ROLE_PERMISSION_ASSIGN`
- bounded migration candidates:
  - RBAC page `v-permission` checks, computed guards, dropdown/action visibility checks, and any directly affected
    tests/types inside owned scope
- forbidden expansions in this topic:
  - adding a new alias for backward compatibility
  - changing backend permission values or introducing a frontend/backend mapping layer
  - widening into route/menu/auth/OpenAPI governance

## Batch Plan

1. Batch 0: initialize topic docs and record canonical map plus alias drift map.
2. Batch 1: replace owned-scope alias usage with canonical permission naming and remove alias-only wrapper logic.
3. Batch 2: run regression audit, record search evidence, and fix only bounded same-semantic leftovers.
4. Batch 3: archive-ready closeout, final validation, recovery-index update, and archive record.

## Final Archive Record

- Final status: `archive-ready`
- Archive summary:
  - owned frontend RBAC permission-code naming now uses the canonical symbolic name
    `RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN`
  - the historical duplicate alias `ROLE_PERMISSION_MANAGE` is no longer present in owned frontend scope
  - page visibility and action-guard behavior remain unchanged because the canonical backend permission value stayed
    `role.permission.assign`
- Final validations:
  - `git status --short`
  - `git branch --show-current`
  - `cd web && bun run check`
  - `git diff --check`
- Remaining risks:
  - future backend RBAC contract topic
  - future permission observability topic
- Continuation:
  - no continuation required for this topic
