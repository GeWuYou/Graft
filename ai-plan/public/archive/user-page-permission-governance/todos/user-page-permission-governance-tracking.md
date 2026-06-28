# User Page Permission Governance Tracking

## Topic

- Topic: `user-page-permission-governance`
- Status: `archived`
- Goal: converge the user-management page onto the existing centralized frontend permission visibility and runtime
  guard model without changing backend contracts or business behavior.
- Recovery source:
  - `ai-plan/public/README.md`
  - current RBAC/user management implementation
  - latest `rbac-visibility-governance` archive record
- Branch: `feat/wt-rbac-further-development`

## Scope

- Owned scope:
  - `ai-plan/public/archive/user-page-permission-governance/**`
  - `ai-plan/public/README.md`
  - `web/src/modules/user/**`
  - `web/src/modules/rbac/**`
  - `web/src/store/modules/permission.ts`
  - `web/src/utils/route/**`
  - `web/src/app/bootstrap/**`
  - bounded tests/types only if directly required
- Forbidden scope:
  - `server/**`
  - OpenAPI/generated contract files
  - RBAC backend permission registry
  - unrelated layout/theme/i18n refactors

## Repository Truth

- `AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/public/archive/rbac-visibility-governance/**`

## Governance Guardrails

- No backend API change.
- No backend permission-code change.
- No capability snapshot / denial reason / observability contract.
- No global UI restructuring.
- No user-management business behavior change.
- Missing permission-code gaps are recorded as risks, not solved cross-boundary in this topic.

## Current Recovery Point

- Batch 0 completed topic initialization.
- Batch 1 completed the user-page permission convergence and was committed as `fae5058` (`fix(user-page-permission):
align user actions with centralized guards`).
- Topic docs now exist under `ai-plan/public/archive/user-page-permission-governance/**`.
- The current codebase already has the canonical frontend permission building blocks:
  - bootstrap snapshot permission truth in `web/src/store/modules/permission.ts`
  - `v-permission` in `web/src/app/bootstrap/permission-directive.ts`
  - RBAC page reference usage in `web/src/modules/rbac/pages/index.vue`
- Batch 2 regression audit confirms the owned user-management scope is now aligned with the canonical permission path:
  - user-page visibility entry points use `v-permission`
  - row dropdown permission-only disabled semantics were removed in favor of visibility filtering
  - privileged handlers retain local runtime guards through `ensureUserPermission(...)`
  - no new same-pattern drift remains in the owned `web/src/modules/user/**` scope

## Batch Plan

1. Batch 0: topic initialization and current-state map. Status: completed.
2. Batch 1: user page permission implementation. Status: completed.
3. Batch 2: regression audit and consistency check. Status: completed.
4. Batch 3: archive-ready closeout. Status: completed.

## Batch 0 Findings Summary

- Canonical permission truth:
  - `permissionStore.hasPermission/hasAnyPermission/hasAllPermissions` are the stable helpers
  - `v-permission` is the stable element-visibility primitive
  - route visibility continues to flow from bootstrap menus and mounted runtime routes
- RBAC page reference pattern:
  - create/edit/assign-permission actions already use `v-permission`
  - disabled state remains for business-state constraints, not as the primary permission signal
- User page drift map:
  - header create and empty-state create depend on `canCreateUsers`
  - row assign-roles depends on `canReadUserRoles`
  - row edit depends on `canUpdateUsers`
  - batch assign-roles remains visible but disabled by `!canAssignUserRoles`
  - row dropdown action items remain visible but disabled by `!canDisableUsers` or `!canUpdateUsers`
  - operation-column visibility is aggregated by `canShowOperationColumn`
  - privileged action handlers do not enforce a local runtime guard before API invocation

## Batch 2 Audit Outcome

- `web/src/modules/user/pages/index.vue`
  - no page-local `canCreate/canUpdate/canDelete/canAssign` computed wrappers remain for template visibility
  - no permission-only visible-but-disabled batch or dropdown action remains
  - runtime guards remain present on create, edit, reset-password, status toggle, delete, and role-management handlers
- `web/src/modules/rbac/pages/index.vue`
  - still serves as the same reference pattern for `v-permission`-first critical action visibility
  - retains local permission computed state and some disabled controls, but those are unchanged in this topic and were
    not widened because Batch 2 found no user-page regression requiring cross-file refactor
- Validation:
  - `cd web && bun run check`
  - `git diff --check`

## Immediate Next Step

- None. The topic is archived.

## Validation

- Required for Batch 0:
  - `git diff --check`
- Required for Batch 1:
  - `cd web && bun run check`
  - `git diff --check`

## Commit Eligibility Note

- Batch 1 owned scope is clear, but the worktree still includes uncommitted topic-initialization docs from Batch 0 plus
  Batch 1 tracking updates.
- A scoped commit remains possible, but the outer loop should decide whether to:
  - keep Batch 0/1 together in one owned commit, or
  - preserve batch readability and leave commit creation to a later closeout step

## Final Archive Summary

- Archive status:
  - `archived`
- Archive-ready reason:
  - Batch 0 established the topic docs and current-state map.
  - Batch 1 aligned the user page with canonical `v-permission` visibility and local runtime guards.
  - Batch 2 confirmed no same-pattern permission drift remained in the owned `user` scope.
  - Batch 3 completed recovery-index and topic-doc closeout with final validation.
- Final validations:
  - `git status --short`
  - `git branch --show-current`
  - `cd web && bun run check`
  - `git diff --check`
- Remaining risks:
  - `ROLE_PERMISSION_MANAGE` alias cleanup remains a separate future frontend topic.
  - any missing backend permission code discovered by future user-management work remains a separate RBAC contract
    topic, not part of this closed frontend slice.
