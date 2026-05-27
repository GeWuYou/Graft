# Frontend Permission Code Cleanup Tracking

## Topic

- Topic: `frontend-permission-code-cleanup`
- Status: `active`
- Goal: unify frontend RBAC permission-code naming with canonical backend permission values while preserving current
  visibility behavior.
- Recovery source:
  - `ai-plan/public/README.md`
  - archived `rbac-visibility-governance`
  - archived `user-page-permission-governance`
  - current RBAC frontend implementation
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-rbac-further-development`
- Branch: `feat/wt-rbac-further-development`
- Loop mode: `topic-completion-loop`

## Scope

- Owned scope:
  - `ai-plan/public/frontend-permission-code-cleanup/**`
  - `ai-plan/public/README.md`
  - `web/src/modules/rbac/**`
  - `web/src/modules/user/**`
  - `web/src/store/modules/permission.ts`
  - `web/src/constants/**`
  - `web/src/types/**`
  - bounded tests/types only if directly required
- Forbidden scope:
  - `server/**`
  - OpenAPI/generated contracts
  - backend permission registry / seed / migration / auth contract
  - unrelated refactor/reformat
  - UI redesign
  - route architecture rewrite

## Repository Truth

- `AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`

## Governance Guardrails

- Frontend permission-code cleanup only.
- No backend permission change.
- No new alias compatibility layer.
- No visibility-behavior change.
- If canonical/frontend mismatch requires backend action, stop as blocked instead of widening scope.

## Current Recovery Point

- Batch 0 initialized the topic docs.
- Batch 0 confirmed the current owned-scope canonical permission-code map.
- Batch 0 identified `RBAC_PERMISSION_CODE.ROLE_PERMISSION_MANAGE` as the only confirmed historical alias drift in the
  active owned permission contracts.
- Batch 0 confirmed the alias currently resolves to the canonical backend value `role.permission.assign`.
- Batch 0 did not modify runtime behavior.

## Batch Plan

1. Batch 0: topic initialization and permission-code map. Status: completed.
2. Batch 1: canonical permission-code alignment. Status: completed.
3. Batch 2: regression audit. Status: pending.
4. Batch 3: archive-ready closeout. Status: pending.

## Batch 0 Findings

- canonical RBAC contract currently exposes both:
  - `ROLE_PERMISSION_ASSIGN`
  - `ROLE_PERMISSION_MANAGE`
- both names resolve to the same backend canonical value:
  - `role.permission.assign`
- owned-scope alias usage found in active pages:
  - `web/src/modules/rbac/pages/index.vue`
    - `v-permission` all-of check for assign-permissions button
    - `canAssignPermissions` computed guard
    - `canShowOperationColumn` computed guard
- current user-page permission usage already references canonical names for role assignment:
  - `USER_ROLE_READ`
  - `USER_ROLE_ASSIGN`
- current frontend permission helper contract remains canonical and unchanged:
  - `permissionStore.hasPermission`
  - `permissionStore.hasAnyPermission`
  - `permissionStore.hasAllPermissions`
  - `v-permission`

## Batch 1 Entry Conditions

- replace all owned-scope alias usage with `ROLE_PERMISSION_ASSIGN`
- delete the alias definition once no owned-scope reference remains
- remove alias-only wrapper logic if found during implementation
- keep behavior equivalent:
  - authorized users still see and use the same RBAC/user actions
  - unauthorized users remain hidden or guarded exactly as before

## Batch 1 Result

- replaced owned-scope `RBAC_PERMISSION_CODE.ROLE_PERMISSION_MANAGE` references with
  `RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN`
- removed the alias definition from `web/src/modules/rbac/contract/permissions.ts`
- updated RBAC page permission guards without changing the underlying canonical permission value
- updated directly affected RBAC page tests to grant `ROLE_PERMISSION_ASSIGN`
- acceptance status:
  - no remaining `ROLE_PERMISSION_MANAGE` usage in owned frontend runtime scope
  - no duplicate symbolic naming remains for `role.permission.assign` in owned frontend contract definitions
  - visibility behavior remains equivalent because the canonical value is unchanged

## Required Validation

- Batch 0:
  - `git diff --check`
- Batch 1 and later:
  - `cd web && bun run check`
  - `git diff --check`

## Commit Plan

- Batch 0:
  - `docs(frontend-permission-cleanup): initialize governance topic`
- Batch 1:
  - `fix(frontend-permission-cleanup): align permission codes with canonical naming`
- Batch 2:
  - `docs(frontend-permission-cleanup): record regression audit`
- Batch 3:
  - `docs(frontend-permission-cleanup): archive governance topic`

## Batch 2 Result

- ran owned-scope regression audit for:
  - `ROLE_PERMISSION_MANAGE`
  - canonical naming around `role.permission.assign`
  - obsolete alias-helper or deprecated permission-constant patterns
- audit conclusion:
  - no remaining `ROLE_PERMISSION_MANAGE` usage exists in owned frontend runtime/type scope
  - `web/src/modules/rbac/contract/permissions.ts` exposes only canonical
    `ROLE_PERMISSION_ASSIGN -> role.permission.assign`
  - no obsolete alias helper or deprecated permission constant pattern was found in owned runtime helpers
  - no same-semantic residual fix was required after Batch 1
- visibility-check coverage confirmed in owned scope:
  - RBAC page assign-permission button and related computed guards use `ROLE_PERMISSION_ASSIGN`
  - user page role-management visibility remains canonical through
    `userRoleManagePermissionCodes = [USER_ROLE_READ, USER_ROLE_ASSIGN]`
  - route visibility logic in `web/src/store/modules/permission.ts` remains bootstrap-driven and unchanged by this topic
  - dropdown/action visibility checks in owned RBAC and user pages still flow through `hasPermission`,
    `hasAnyPermission`, or `v-permission`
- acceptance status:
  - alias drift is closed within owned runtime scope
  - no topic expansion was needed
  - Batch 2 is docs-only
