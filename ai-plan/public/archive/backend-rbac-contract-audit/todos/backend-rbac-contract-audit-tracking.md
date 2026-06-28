# Backend RBAC Contract Audit Tracking

## Topic

- Topic: `backend-rbac-contract-audit`
- Status: `archived`
- Goal: audit the current RBAC permission/menu/API/guard contract closure across `server` and `web` without modifying
  runtime code in Batch 0.
- Branch: `feat/wt-rbac-further-development`
- Task class: `cross-boundary`
- Loop mode: `topic-completion-loop`

## Scope

- Owned scope:
  - `ai-plan/public/archive/backend-rbac-contract-audit/**`
  - `ai-plan/public/README.md`
  - read-only audit of:
    - `server/plugins/rbac/**`
    - `server/internal/permission/**`
    - `server/internal/menu/**`
    - `server/internal/httpx/**`
    - `web/src/modules/rbac/**`
    - `web/src/modules/user/**`
    - `web/src/store/modules/permission.ts`
    - `web/src/utils/route/**`
- Forbidden scope:
  - broad runtime redesign or out-of-scope runtime changes
  - database schema / migrations
  - unrelated plugin code
  - OpenAPI/generated contract mutation without a recorded blocking mismatch
  - capability snapshot, denial reason, row-level permission, org/tenant model expansion

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `.ai/environment/tools.ai.yaml`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- archived:
  - `rbac-visibility-governance`
  - `user-page-permission-governance`
  - `frontend-permission-code-cleanup`

## Governance Guardrails

- Batch 0 is docs-only.
- Batch 0 must not modify runtime or test code.
- Record confirmed inventory separately from later audit questions.
- If a later batch discovers a real contract mismatch that requires broader change, record it before proposing any fix.

## Current Recovery Point

- Batch 0 completed topic initialization.
- Batch 0 created:
  - `ai-plan/public/archive/backend-rbac-contract-audit/README.md`
  - `ai-plan/public/archive/backend-rbac-contract-audit/todos/backend-rbac-contract-audit-tracking.md`
  - `ai-plan/public/archive/backend-rbac-contract-audit/traces/backend-rbac-contract-audit-trace.md`
- Batch 0 updated `ai-plan/public/README.md` to register this topic as the active recovery entry.
- Batch 0 recorded the initial audit inventory and draft matrix for backend and frontend RBAC contract surfaces.
- Batch 1 completed the backend-only permission/menu/API/guard audit without changing runtime code.
- Batch 1 confirmed the current backend owned scope does not need a low-risk runtime fix.
- Batch 2 completed the frontend permission/route/action audit and applied one bounded RBAC page visibility fix inside
  owned frontend scope.

## Batch Plan

1. Batch 0: topic initialization and audit inventory. Status: completed.
2. Batch 1: backend permission/menu/API/guard audit. Status: completed.
3. Batch 2: frontend permission/route/action audit. Status: completed.
4. Batch 3: cross-boundary consistency audit. Status: completed.
5. Batch 4: MVP-stable decision and archive closeout. Status: completed.

## Batch 0 Findings

- backend RBAC owned permission registry currently exposes nine canonical permission codes:
  - `role.read`
  - `role.create`
  - `role.update`
  - `role.status.update`
  - `role.delete`
  - `role.permission.assign`
  - `permission.read`
  - `user.role.read`
  - `user.role.assign`
- backend RBAC owned menu declarations currently expose four entries:
  - `/access-control`
  - `/access-control/overview`
  - `/access-control/roles`
  - `/access-control/permissions`
- backend RBAC owned route registration currently wires explicit guards for:
  - role read/write routes
  - permission read routes
  - user-role snapshot and mutation routes
- backend guard semantics currently centralize through `httpx.RequirePermission(...)` and return denied
  `permission` detail on `403`.
- frontend owned permission constants currently converge on canonical names in:
  - `web/src/modules/rbac/contract/permissions.ts`
  - `web/src/modules/user/contract/permissions.ts`
- frontend owned bootstrap route registrations currently exist for:
  - `/access-control/users`
  - `/access-control/roles`
  - `/access-control/permissions`
- Batch 0 observed two follow-up consistency questions:
  - `/access-control/overview` backend menu exists but no owned page registration was found in current scope
  - `/access-control/users` frontend route exists but its backend menu owner is outside current Batch 0 RBAC backend
    read scope

## Immediate Next Step

- No continuation required for this topic.
- If a follow-up is needed:
  - open a new bugfix-only topic for a proven defect inside the audited RBAC MVP scope
  - open a new future-scope topic for data permission, org permission, tenant permission, or observability work

## Required Validation

- Batch 0:
  - `git diff --check`
- Batch 1:
  - `git diff --check`
- Batch 2:
  - `cd web && bun run check`
  - `git diff --check`
- Batch 3:
  - docs-only closeout this round relied on prior Batch 1 backend validation and Batch 2 web validation
  - `git diff --check`
- Batch 4:
  - `git status --short`
  - `git branch --show-current`
  - `cd web && bun run check`
  - `cd server && go run ./cmd/graft validate backend`
  - `git diff --check`

## Commit Plan

- Batch 0:
  - `docs(rbac-contract-audit): initialize audit topic`
- Batch 1:
  - `docs(rbac-contract-audit): record backend guard audit`
- Batch 2:
  - `fix(rbac-contract-audit): align frontend permission usage`
- Batch 3:
  - `docs(rbac-contract-audit): record cross-boundary audit`
- Batch 4:
  - `docs(rbac-contract-audit): archive MVP-stable audit`

## Batch 1 Findings

- owned backend RBAC permission surfaces still converge on nine stable wire-format codes even though the contract file
  also exports same-value consumer aliases
- owned backend menu permission references are currently closed inside RBAC scope:
  - `/access-control/roles` -> `role.read`
  - `/access-control/permissions` -> `permission.read`
- owned backend API guards are currently closed inside RBAC scope:
  - all role writes use dedicated write guards
  - role-permission writes use `role.permission.assign`
  - permission and role-permission reads use `permission.read`
  - user-role reads use `user.role.read`
  - user-role writes, including batch writes, use `user.role.assign`
- builtin and privileged lifecycle protections remain enforced below the page layer:
  - builtin role rename blocked in write service
  - builtin/active/bound role mutation constraints enforced in repository lifecycle checks
  - actor self-lockout from builtin admin role blocked in user-role write service
- `403`, `404`, and `400` semantics are coherent in current owned scope:
  - authz denial -> `403 auth.forbidden` with denied permission detail
  - self-lockout protection -> dedicated `403 rbac.cannot_remove_own_admin_role`
  - missing role/user/permission resources -> dedicated `404`
  - malformed input or stale referenced IDs -> `400 common.invalid_argument`
- no low-risk backend runtime gap was proven in owned scope
- current registries still do not enforce uniqueness or menu-permission reference validity at runtime; tests and
  contract discipline remain the current safeguard

## Batch 2 Findings

- owned frontend permission constants converge on canonical backend-owned permission codes across:
  - `web/src/modules/rbac/contract/permissions.ts`
  - `web/src/modules/user/contract/permissions.ts`
- owned frontend bootstrap route registrations now have confirmed menu owners for all Batch 2 paths:
  - `/access-control/overview` -> frontend `access-control` module route registration, backend RBAC menu owner
  - `/access-control/users` -> frontend `user` module route registration, backend user plugin menu owner
  - `/access-control/roles` -> frontend `rbac` module route registration, backend RBAC menu owner
  - `/access-control/permissions` -> frontend `rbac` module route registration, backend RBAC menu owner
- owned RBAC and user pages use expected permission closure:
  - role page create/edit actions use `role.create` / `role.update`
  - role permission assignment entry requires `permission.read` + `role.permission.assign`
  - user page role-management entry requires `user.role.read` + `user.role.assign`
  - user page create/edit/more actions map to `user.create` / `user.update` / `user.disable`
- no obvious frontend-visible but backend-always-forbidden drift remains in owned scope after the Batch 2 fix
- one proven low-risk owned-scope drift was corrected:
  - RBAC role-row `More` dropdown no longer exposes permission-missing actions as disabled entries
  - builtin/lifecycle disabled semantics remain intact because they encode business state, not permission absence

## Batch 3 Findings

- Batch 1 and Batch 2 evidence now merge into one cross-boundary closure path for current MVP scope:
  - backend permission registry
  - backend API guards
  - backend menu declarations
  - frontend permission constants
  - frontend bootstrap route registrations
  - frontend page and action visibility
- Required capability surfaces are aligned in current owned scope:
  - role list/detail/create/update/status/delete
  - permission list/detail/filter/read-only behavior
  - role-permission list/replace/add/remove
  - user-role list/replace/add/remove/batch
  - user management manage-roles entry
  - dynamic menu bootstrap
  - route visibility
  - button/action visibility
  - builtin role/permission protection
  - auth forbidden / unauthorized separation
- No new runtime drift was proven in Batch 3.
- One tiny owned-scope documentation drift was corrected:
  - `server/plugins/rbac/README.md` no longer describes the stale `.../roles/assign` path or replace-only write semantics
- Remaining note is risk-only rather than blocker:
  - registry and menu closure still rely on tests plus canonical ownership, not runtime duplicate/reference enforcement

## Batch 4 Final Decision

- Final closeout status: `archive-ready`
- MVP-stable decision: `mvp-stable-with-risks`
- Decision basis:
  - Batch 1 backend audit found no proven runtime guard or permission-contract mismatch in owned scope
  - Batch 2 frontend audit closed the only proven owned-scope visibility drift
  - Batch 3 cross-boundary audit confirmed current backend and frontend RBAC contract closure is aligned for MVP scope
  - final required backend and web validation passed in Batch 4
- Archive policy:
  - this topic line is closed for proactive feature expansion
  - later work is bugfix-only unless a new topic is opened
  - data permission / row-level permission, organization permission, tenant permission, and observability remain
    future topics instead of reopen triggers for this archive
- Residual risks:
  - registry and menu closure still depend on tests plus disciplined canonical ownership, not runtime
    duplicate/reference enforcement
  - this residual risk is non-blocking for current MVP archive readiness, so the terminal decision remains
    `mvp-stable-with-risks` rather than `blocked`
