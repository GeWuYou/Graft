# Frontend Permission Code Cleanup Trace

## 2026-05-27 Batch 0 initialized topic docs and recorded permission-code drift

- Reused the inherited startup receipt under root `AGENTS.md` for a `web` round inside `graft-multi-agent-loop`.
- Did not use `graft-multi-agent-batch`.
  - Reason: this round was a single-slice owned-scope audit plus documentation update; internal delegation had no
    practical payoff.
- Used `graft-task-closeout` style acceptance logic for owned-scope verification, validation, and commit eligibility.
- Read:
  - `.ai/environment/tools.ai.yaml`
  - `web/AGENTS.md`
  - `ai-plan/public/README.md`
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
  - `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
  - `ai-plan/public/archive/rbac-visibility-governance/README.md`
  - `ai-plan/public/archive/user-page-permission-governance/README.md`
  - `web/src/modules/rbac/contract/permissions.ts`
  - `web/src/modules/user/contract/permissions.ts`
  - `web/src/modules/rbac/pages/index.vue`
  - `web/src/modules/user/pages/index.vue`
  - `web/src/store/modules/permission.ts`
- Confirmed the worktree was clean before edits and the current branch is:
  - `feat/wt-rbac-further-development`
- Created the new topic document set:
  - `ai-plan/public/archive/frontend-permission-code-cleanup/README.md`
  - `ai-plan/public/archive/frontend-permission-code-cleanup/todos/frontend-permission-code-cleanup-tracking.md`
  - `ai-plan/public/archive/frontend-permission-code-cleanup/traces/frontend-permission-code-cleanup-trace.md`
- Audited owned-scope permission contracts and confirmed the canonical frontend permission-code map:
  - RBAC canonical value for role-permission assignment is `role.permission.assign`
  - user-role management permissions remain `user.role.read` and `user.role.assign`
  - user-management permissions remain `user.read`, `user.create`, `user.update`, `user.disable`,
    `user.session.read`, and `user.session.revoke`
- Identified the historical alias drift:
  - `RBAC_PERMISSION_CODE.ROLE_PERMISSION_MANAGE` is defined in `web/src/modules/rbac/contract/permissions.ts`
  - it resolves to the same value as `RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN`
  - it is therefore a duplicate symbolic name rather than a distinct backend permission
- Identified owned-scope alias usage relevant to Batch 1:
  - `web/src/modules/rbac/pages/index.vue`
    - assign-permission button `v-permission` all-of check
    - `canAssignPermissions` computed guard
    - `canShowOperationColumn` computed guard
- Confirmed no runtime behavior change was needed in Batch 0.
- Chose Batch 1 bounds:
  - replace owned-scope alias references with `ROLE_PERMISSION_ASSIGN`
  - remove `ROLE_PERMISSION_MANAGE` after references are eliminated
  - avoid any new alias compatibility layer
- Kept `ai-plan/public/README.md` unchanged in Batch 0.
  - Basis: this round only initializes topic-local recovery material and the provided batch contract did not require a
    recovery-index mutation at initialization time.

## 2026-05-27 Batch 1 aligned frontend permission codes with canonical naming

- Reused the inherited startup receipt under root `AGENTS.md` for a `web` round inside `graft-multi-agent-loop`.
- Did not use `graft-multi-agent-batch`.
  - Reason: the round was a single bounded implementation slice inside one owned frontend module area.
- Used `graft-task-closeout` style acceptance logic for owned-scope validation and commit eligibility.
- Updated owned-scope RBAC permission usage:
  - `web/src/modules/rbac/contract/permissions.ts`
    - removed `ROLE_PERMISSION_MANAGE`
  - `web/src/modules/rbac/pages/index.vue`
    - switched assign-permission `v-permission` check to `ROLE_PERMISSION_ASSIGN`
    - switched `canAssignPermissions` guard to `ROLE_PERMISSION_ASSIGN`
    - switched `canShowOperationColumn` permission set to `ROLE_PERMISSION_ASSIGN`
  - `web/src/modules/rbac/pages/index.test.ts`
    - replaced granted-code fixtures that referenced the removed alias
- Kept permission behavior unchanged:
  - the old alias and the canonical symbol both resolved to `role.permission.assign`
  - replacing the symbolic name did not change the underlying permission value tested by `hasPermission`,
    `hasAnyPermission`, or `v-permission`
- Batch 1 validations to run:
  - `cd web && bun run check`
  - `git diff --check`
- Batch 1 closeout target:
  - create scoped commit `fix(frontend-permission-cleanup): align permission codes with canonical naming` only if both
    validations pass

## 2026-05-27 Batch 2 recorded regression audit evidence

- Reused the inherited startup receipt under root `AGENTS.md` for a `web` round inside `graft-multi-agent-loop`.
- Did not use `graft-multi-agent-batch`.
  - Reason: the round remained a single owned-scope regression audit plus topic-doc update; no parallel internal slice
    was justified.
- Used `graft-task-closeout` style acceptance logic for audit verification, validation, and scoped commit eligibility.
- Confirmed the worktree was clean before the Batch 2 doc edits.
- Ran owned-scope search audit commands:
  - `rg -n "ROLE_PERMISSION_MANAGE" web/src/modules/rbac web/src/modules/user web/src/store/modules/permission.ts web/src/types -S`
    - result: no matches
  - `rg -n "role\\.permission\\.assign|ROLE_PERMISSION_ASSIGN" web/src/modules/rbac web/src/modules/user web/src/store/modules/permission.ts web/src/types -S`
    - result: canonical RBAC references only
    - `web/src/modules/rbac/contract/permissions.ts`
    - `web/src/modules/rbac/pages/index.vue`
    - `web/src/modules/rbac/pages/index.test.ts`
    - `web/src/modules/rbac/contract/permission-copy.ts`
  - `rg -n "alias|deprecated|ROLE_PERMISSION_" web/src/modules/rbac web/src/modules/user web/src/store/modules/permission.ts web/src/types -S`
    - result: no obsolete alias helper or deprecated permission-constant pattern in owned runtime/type helpers
- Reviewed owned visibility checkpoints:
  - `web/src/modules/rbac/pages/index.vue`
    - assign-permission action uses
      `v-permission="{ allOf: [permissionCodes.PERMISSION_READ, permissionCodes.ROLE_PERMISSION_ASSIGN] }"`
    - `canAssignPermissions` uses `permissionStore.hasPermission(permissionCodes.ROLE_PERMISSION_ASSIGN)`
    - operation-column visibility still uses `permissionStore.hasAnyPermission([... permissionCodes.ROLE_PERMISSION_ASSIGN])`
  - `web/src/modules/user/pages/index.vue`
    - batch and row role-assignment actions continue to use `v-permission="{ allOf: userRoleManagePermissionCodes }"`
    - user action guards remain canonical through `userPermissionCodes.*` and `userRoleManagePermissionCodes`
  - `web/src/store/modules/permission.ts`
    - route visibility path remains bootstrap-menu driven
    - permission visibility helpers remain `hasPermission`, `hasAnyPermission`, and `hasAllPermissions`
- Determined no runtime patch was required in Batch 2.
  - Basis: the audit found no remaining alias drift in the allowed owned runtime/type scope.
- Batch 2 validations to run:
  - `cd web && bun run check`
  - `git diff --check`
- Batch 2 closeout target:
  - create scoped commit `docs(frontend-permission-cleanup): record regression audit` only if both validations pass

## 2026-05-27 Batch 3 archived the governance topic

- Reused the inherited startup receipt under root `AGENTS.md` for a `web` round inside `graft-multi-agent-loop`.
- Did not use `graft-multi-agent-batch`.
  - Reason: the round was a final owned-scope archive-closeout slice with no justified internal delegation.
- Used `graft-task-closeout` style acceptance logic for terminal archive readiness, validation, and scoped commit
  eligibility.
- Final verification commands for this round:
  - `git status --short`
  - `git branch --show-current`
  - `cd web && bun run check`
  - `git diff --check`
- Updated owned recovery docs only:
  - `ai-plan/public/README.md`
    - moved `frontend-permission-code-cleanup` into the archived recovery index
  - `ai-plan/public/archive/frontend-permission-code-cleanup/README.md`
    - marked the topic archived and added the final archive record
  - `ai-plan/public/archive/frontend-permission-code-cleanup/todos/frontend-permission-code-cleanup-tracking.md`
    - marked Batches 2 and 3 completed and recorded the archive-closeout result
  - `ai-plan/public/archive/frontend-permission-code-cleanup/traces/frontend-permission-code-cleanup-trace.md`
    - recorded this final closeout round
- Kept remaining risks intentionally narrow:
  - future backend RBAC contract topic
  - future permission observability topic
- Batch 3 closeout target:
  - create scoped commit `docs(frontend-permission-cleanup): archive governance topic` only if validation passes and
    ownership remains clean
