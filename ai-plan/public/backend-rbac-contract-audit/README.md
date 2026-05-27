# Backend RBAC Contract Audit

## Status

- Topic: `backend-rbac-contract-audit`
- Status: `active`
- Branch: `feat/wt-rbac-further-development`
- Task class: `cross-boundary`
- Loop mode: `topic-completion-loop`

## Goal

Audit the current RBAC contract closure without changing runtime code:

- backend permission registry
- backend menu declaration
- backend RBAC API routes
- backend request guard wiring
- frontend permission constants
- frontend bootstrap route and menu visibility usage
- frontend page and action permission usage

This topic is an audit-first cross-boundary contract topic. Batch 0 establishes the topic, records the initial
inventory, and drafts the first audit matrix.

## Hard Constraints

- Do not modify runtime code in Batch 0.
- Do not change database schema or migrations.
- Do not modify OpenAPI or generated contract files unless a proven blocking mismatch is found first.
- Do not widen into capability snapshot, denial-reason contract, data permission, tenant permission, or broad RBAC
  redesign.

## Scope

- `ai-plan/public/backend-rbac-contract-audit/**`
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

## Recovery Sources

- `ai-plan/public/README.md`
- archived `rbac-visibility-governance`
- archived `user-page-permission-governance`
- archived `frontend-permission-code-cleanup`
- current RBAC backend implementation
- current RBAC frontend implementation

## Batch 0 Audit Inventory

### Backend Permission Registry Inventory

Canonical RBAC backend permission contracts registered by `server/plugins/rbac/plugin_registration.go`:

| Code | Name | Category | Observed ownership |
| --- | --- | --- | --- |
| `role.read` | `Read Roles` | `api` | role list/detail |
| `role.create` | `Create Roles` | `api` | role create |
| `role.update` | `Update Roles` | `api` | role update |
| `role.status.update` | `Update Role Status` | `api` | role status mutation |
| `role.delete` | `Delete Roles` | `api` | role delete |
| `role.permission.assign` | `Assign Role Permissions` | `api` | role-permission write routes |
| `permission.read` | `Read Permissions` | `api` | permission list/detail and role-permission binding snapshot |
| `user.role.read` | `Read User Roles` | `api` | user-role snapshot |
| `user.role.assign` | `Assign User Roles` | `api` | user-role write routes |

Batch 0 conclusion:

- `server/plugins/rbac/contract/permission.go` and `registerRBACPermissions(...)` are the current canonical owned
  permission source in read scope.
- All owned RBAC permissions are typed on the backend and emitted into the platform permission registry as plain string
  codes at the registration boundary.
- Batch 0 did not find a second owned backend alias for the above nine codes.

### Backend Menu Declaration Inventory

Owned backend menu declarations registered by `registerRBACMenu(...)`:

| Menu code | Path | Title key | Permission | Notes |
| --- | --- | --- | --- | --- |
| `access-control.root` | `/access-control` | `menu.access_control.title` | blank | root grouping node |
| `access-control.overview` | `/access-control/overview` | `menu.access_control.overview.title` | blank | overview child |
| `role.list` | `/access-control/roles` | `menu.access_control.roles.title` | `role.read` | role list entry |
| `permission.list` | `/access-control/permissions` | `menu.access_control.permissions.title` | `permission.read` | permission list entry |

Batch 0 conclusion:

- RBAC plugin owned menu declarations in current read scope do not declare `/access-control/users`.
- Frontend owned scope does register `/access-control/users` as a page route, so cross-plugin menu ownership for that
  entry remains a Batch 3 consistency question rather than a Batch 0 fix.

### Backend RBAC API Route Inventory

Owned RBAC route fragments declared in `server/plugins/rbac/contract/route.go` and wired in
`server/plugins/rbac/route_registration.go`:

| Method | Path | Guard |
| --- | --- | --- |
| `GET` | `/api/roles` | `role.read` |
| `GET` | `/api/roles/:id` | `role.read` |
| `GET` | `/api/roles/:id/permissions` | `permission.read` |
| `POST` | `/api/roles` | `role.create` |
| `POST` | `/api/roles/:id/update` | `role.update` |
| `POST` | `/api/roles/:id/status` | `role.status.update` |
| `POST` | `/api/roles/:id/delete` | `role.delete` |
| `POST` | `/api/roles/:id/permissions/replace` | `role.permission.assign` |
| `POST` | `/api/roles/:id/permissions/add` | `role.permission.assign` |
| `POST` | `/api/roles/:id/permissions/remove` | `role.permission.assign` |
| `GET` | `/api/permissions` | `permission.read` |
| `GET` | `/api/permissions/:id` | `permission.read` |
| `GET` | `/api/users/:id/roles` | `user.role.read` |
| `POST` | `/api/users/:id/roles/replace` | `user.role.assign` |
| `POST` | `/api/users/:id/roles/add` | `user.role.assign` |
| `POST` | `/api/users/:id/roles/remove` | `user.role.assign` |
| `POST` | `/api/users/roles/replace` | `user.role.assign` |
| `POST` | `/api/users/roles/add` | `user.role.assign` |
| `POST` | `/api/users/roles/remove` | `user.role.assign` |

### Backend Guard Inventory

Batch 0 observed two owned guard layers:

1. `server/plugins/rbac/plugin_registration.go`
   - constructs `managementGuards`
   - binds every RBAC management route to an explicit `httpx.RequirePermission(...)`
2. `server/internal/httpx/authz.go`
   - canonical request guard flow:
     - parse bearer token
     - resolve current user
     - skip authorization only when permission code is blank
     - return `403` with denied `permission` detail on authorization failure

Batch 0 conclusion:

- Backend guard semantics are explicit and centralized.
- Blank permission is currently accepted only for authenticated-only routes; all owned RBAC management routes in current
  read scope use non-blank permission codes.

### Frontend Permission Constant Inventory

Owned frontend permission constants:

- `web/src/modules/rbac/contract/permissions.ts`
  - `role.read`
  - `role.create`
  - `role.update`
  - `role.status.update`
  - `role.delete`
  - `role.permission.assign`
  - `permission.read`
  - `user.role.read`
  - `user.role.assign`
- `web/src/modules/user/contract/permissions.ts`
  - `user.read`
  - `user.create`
  - `user.update`
  - `user.disable`
  - `user.session.read`
  - `user.session.revoke`
- `web/src/store/modules/permission.ts`
  - canonical bootstrap-backed helpers:
    - `hasPermission`
    - `hasAnyPermission`
    - `hasAllPermissions`

Batch 0 conclusion:

- The frontend owned scope now converges on canonical permission names; the historical
  `ROLE_PERMISSION_MANAGE` alias is no longer present.
- Frontend permission truth remains bootstrap-snapshot driven rather than page-local.

### Frontend Route And Menu Visibility Inventory

Owned frontend bootstrap route registrations:

| Module | Menu path | Route name | Page |
| --- | --- | --- | --- |
| `user` | `/access-control/users` | `UserList` | `web/src/modules/user/pages/index.vue` |
| `rbac` | `/access-control/roles` | `RoleList` | `web/src/modules/rbac/pages/index.vue` |
| `rbac` | `/access-control/permissions` | `PermissionList` | `web/src/modules/rbac/pages/permissions/index.vue` |

Owned frontend route visibility observations:

- `web/src/utils/route/bootstrap.ts` derives dynamic routes only from bootstrap menus.
- A bootstrap menu path mounts only when `getBootstrapRouteRegistration(menuPath)` resolves to an owned registration.
- `/access-control/overview` is present in backend-owned menu declarations and in route-transform tests, but Batch 0 did
  not find an owned page registration for it in current `rbac` or `user` modules.
- `USER_ROUTE_PATH.LEGACY_LIST = '/users'` still exists as a contract constant comment, but Batch 0 did not find an
  owned runtime bootstrap registration using it.

### Frontend Page And Action Permission Usage Inventory

Owned page-level permission usage observed in current scope:

- `web/src/modules/rbac/pages/index.vue`
  - `v-permission="ROLE_CREATE"` on create entry
  - `v-permission="{ allOf: [PERMISSION_READ, ROLE_PERMISSION_ASSIGN] }"` on assign-permissions entry
  - `v-permission="ROLE_UPDATE"` on edit entry
  - local computed guards still drive delete/status/action-column behavior
- `web/src/modules/rbac/pages/permissions/index.vue`
  - read-only page in owned scope
  - no additional button-level permission gate observed inside the page body
- `web/src/modules/user/pages/index.vue`
  - `v-permission="CREATE"` on create entry
  - `v-permission="{ allOf: [USER_ROLE_READ, USER_ROLE_ASSIGN] }"` on single-row and batch role-management entries
  - `v-permission="UPDATE"` on edit entry
  - row dropdown options are filtered through permission-store checks for `DISABLE` and `UPDATE`
  - privileged handlers retain local runtime permission guards via `ensureUserPermission(...)`

## Initial RBAC Contract Audit Matrix Draft

| Surface | Canonical permission | Backend registry | Backend menu / route entry | Frontend visibility / route usage | Backend API guard | Batch 0 note |
| --- | --- | --- | --- | --- | --- | --- |
| access-control root | none | n/a | menu declared | dynamic group route derived from bootstrap menus | none | structural shell node |
| access-control overview | none | n/a | menu declared | no owned page registration found in current scope | none in owned scope | candidate cross-boundary drift to verify later |
| user management page | `user.read` for menu visibility, action-specific codes for operations | outside owned RBAC registry scope for `user.read` | frontend route registration exists at `/access-control/users` | page exists and uses canonical `v-permission` / runtime guards | outside current Batch 0 owned backend read scope | keep as consistency item, not Batch 0 fix |
| role management page | `role.read` | yes | menu declared | route registration exists at `/access-control/roles` | yes | current closure present |
| permission management page | `permission.read` | yes | menu declared | route registration exists at `/access-control/permissions` | yes | current closure present |
| role-permission binding snapshot | `permission.read` | yes | no menu; API-only route | surfaced through role page assign-permissions workflow | yes | read/write split differs intentionally from `role.permission.assign` |
| role-permission mutation | `role.permission.assign` | yes | no menu; API-only route | role page assign-permissions entry uses all-of visibility | yes | current closure present |
| user-role snapshot | `user.role.read` | yes | no menu; API-only route | user page role-management entry requires all-of visibility | yes | paired with assign in frontend |
| user-role mutation | `user.role.assign` | yes | no menu; API-only route | user page role-management entry requires all-of visibility | yes | current closure present |

## Batch 0 Conclusions

- Topic docs are now initialized for a dedicated backend RBAC contract audit line.
- The owned RBAC backend scope already exposes explicit typed permission contracts, explicit menu declarations, and
  explicit request guard wiring.
- The owned frontend scope already uses canonical permission constants and bootstrap-driven route visibility.
- Batch 0 identified two concrete follow-up questions for later audit batches without widening scope:
  - `/access-control/overview` has backend menu presence but no owned page registration found in current scope
  - `/access-control/users` has owned frontend route registration but its backend menu owner is not inside current
    Batch 0 backend read scope

## Planned Batches

1. Batch 0: topic initialization and audit inventory.
2. Batch 1: backend permission, menu, API, and guard audit.
3. Batch 2: frontend permission, route, and action audit.
4. Batch 3: cross-boundary contract consistency audit.
5. Batch 4: MVP-stable decision and archive closeout.

## Batch 1 Backend Audit

### Backend Audit Matrix

| Surface | Canonical code | Owner module | Menu usage | API guard usage | CRUD/action meaning | Error semantics | Batch 1 conclusion |
| --- | --- | --- | --- | --- | --- | --- | --- |
| role list/detail | `role.read` | `server/plugins/rbac` | `/access-control/roles` menu requires `role.read` | `GET /api/roles`, `GET /api/roles/:id` use `guards.roleRead` | role directory and single-role snapshot read | invalid query values -> `400 common.invalid_argument`; missing role -> `404 role.not_found`; denied -> `403 auth.forbidden` | read guard and menu reference align |
| role create | `role.create` | `server/plugins/rbac` | none | `POST /api/roles` uses `guards.roleCreate` | create non-builtin role | invalid body/name/display -> `400 common.invalid_argument`; name conflict -> `400 common.invalid_argument`; denied -> `403 auth.forbidden` | dedicated write guard present |
| role update | `role.update` | `server/plugins/rbac` | none | `POST /api/roles/:id/update` uses `guards.roleUpdate` | update role metadata | invalid id/body/name/display -> `400`; missing role -> `404 role.not_found`; builtin rename -> `400 field=name`; denied -> `403` | write guard present and builtin-name protection remains at service layer |
| role status update | `role.status.update` | `server/plugins/rbac` | none | `POST /api/roles/:id/status` uses `guards.roleStatus` | enable/disable role lifecycle | invalid id/body/status -> `400`; missing role -> `404`; builtin immutable -> `409 common.invalid_argument`; denied -> `403` | status mutation has dedicated permission, not folded into `role.update` |
| role delete | `role.delete` | `server/plugins/rbac` | none | `POST /api/roles/:id/delete` uses `guards.roleDelete` | soft-delete disabled unbound roles | invalid id -> `400`; missing role -> `404`; builtin/enabled/bound role -> `409 common.invalid_argument`; denied -> `403` | destructive path has dedicated guard and service/repository lifecycle checks |
| role-permission snapshot | `permission.read` | `server/plugins/rbac` | none | `GET /api/roles/:id/permissions` uses `guards.permissionRead` | read current permission bindings for one role | invalid id -> `400`; missing role -> `404 role.not_found`; denied -> `403` with denied permission detail | read semantics intentionally reuse permission catalog read permission |
| role-permission write | `role.permission.assign` | `server/plugins/rbac` | none | `POST /api/roles/:id/permissions/{replace|add|remove}` use `guards.rolePermissionAssign` | mutate role-permission bindings | invalid id/body/ids -> `400`; missing role -> `404`; deleted or missing permission IDs -> `400 field=permission_ids`; denied -> `403` | all role-permission writes guarded consistently |
| permission list/detail | `permission.read` | `server/plugins/rbac` | `/access-control/permissions` menu requires `permission.read` | `GET /api/permissions`, `GET /api/permissions/:id` use `guards.permissionRead` | permission catalog read | invalid query -> `400`; missing permission detail -> `404 permission.not_found`; denied -> `403` | menu and guard references align |
| user-role snapshot | `user.role.read` | `server/plugins/rbac` | none | `GET /api/users/:id/roles` uses `guards.userRoleRead` | read one user's role IDs | invalid id -> `400`; missing user -> `404 user.not_found`; denied -> `403` | read path guarded as expected |
| user-role write | `user.role.assign` | `server/plugins/rbac` | none | `POST /api/users/:id/roles/{replace|add|remove}` and `POST /api/users/roles/{replace|add|remove}` use `guards.userRoleAssign` | mutate single-user or batch user-role bindings | invalid id/body/role_ids -> `400`; missing user -> `404 user.not_found`; deleted or missing role IDs -> `400 field=role_ids`; removing own builtin admin -> `403 rbac.cannot_remove_own_admin_role` | all single and batch writes guarded consistently |

### Batch 1 Evidence-Backed Answers

- Permission code uniqueness and stability:
  - RBAC permission contracts are defined as typed `PermissionCode` constants in `server/plugins/rbac/contract/permission.go`.
  - The file exports both `*Permission` names and consumer aliases like `RoleRead`, but all wire values collapse to the same nine stable strings; no second wire-format code was found.
  - Registration order in `rbacPermissionItems(...)` and `plugin_test.go` snapshots the same nine codes.
- Menu permission references all exist:
  - `registerRBACMenu(...)` references only `role.read` and `permission.read`.
  - Both codes are present in `rbacPermissionItems(...)`.
  - Root and overview menu entries intentionally leave `Permission` blank as authenticated shell nodes, not missing references.
- API guard permission references all exist:
  - `managementGuards` is built only from the nine typed RBAC contract codes.
  - `registerManagementRoutes(...)` uses those guards on every owned RBAC route in scope.
- RBAC write interfaces all have guard:
  - Role create/update/status/delete, role-permission replace/add/remove, user-role single-user replace/add/remove, and batch replace/add/remove all register explicit non-blank guards.
- RBAC read interfaces have expected guard:
  - Role list/detail use `role.read`.
  - Permission list/detail and role-permission binding snapshot use `permission.read`.
  - User-role snapshot uses `user.role.read`.
- Role-permission and user-role read/write semantics are guarded correctly:
  - Role-permission read/write is intentionally split between `permission.read` and `role.permission.assign`.
  - User-role read/write is intentionally split between `user.role.read` and `user.role.assign`.
- Builtin role or privileged semantics remain enforced at service/repository layer:
  - builtin role rename is blocked in `managementWriter.UpdateRole(...)`
  - builtin role disable/delete and enabled-or-bound delete restrictions are enforced by repository lifecycle checks
  - self-removal of the actor's builtin admin role is blocked in user-role write service logic, including batch replace/remove flows
- `403` forbidden uses standard `httpx` auth/RBAC semantics:
  - `httpx.RequirePermission(...)` maps permission denial to `auth.forbidden` and echoes the denied permission in `details.permission`.
  - RBAC self-lockout protection intentionally uses a dedicated domain-level `403` (`rbac.cannot_remove_own_admin_role`) after the route-level permission check passes.
- `404` and `400` semantics are coherent:
  - missing role, permission, and user resources map to dedicated not-found contracts
  - malformed path/body/query input maps to `common.invalid_argument`
  - TOCTOU-style deleted role/permission IDs are intentionally normalized to `400` argument errors instead of leaking storage-level misses as `404`

### Batch 1 Conclusions

- No clear low-risk backend runtime gap was proven inside the owned scope, so Batch 1 stays audit-only.
- Backend RBAC management routes currently have explicit guard closure for all owned read and write surfaces.
- Guarded permission semantics are specific enough to distinguish:
  - role metadata read
  - permission catalog read
  - role-permission mutation
  - user-role read
  - user-role mutation
- The backend still relies on documentation and tests, not registry-level runtime enforcement, for:
  - permission code uniqueness
  - menu-permission reference validity
  - duplicate registration detection
- That limitation is a governance note for later contract-hardening work, not a Batch 1 fix:
  - current registries append declarations without duplicate or cross-reference validation
  - current owned tests snapshot the intended permission and menu closure

## Batch 2 Frontend Audit

### Frontend Audit Matrix

| Surface | Frontend canonical code | Backend canonical owner | Route / menu visibility | Page / action usage | Runtime guard usage | Expected backend API guard | Batch 2 conclusion |
| --- | --- | --- | --- | --- | --- | --- | --- |
| access-control overview | none | backend menu in `server/plugins/rbac`, frontend route in `web/src/modules/access-control` | backend bootstrap menu `/access-control/overview` maps to `AccessControlOverview` page | shell overview only, no owned RBAC action surface in this batch | none in owned page scope | none | Batch 0 missing-page question is closed; frontend page registration exists in adjacent owned module |
| user management page | `user.read` | `server/plugins/user/contract/permission.go` | `/access-control/users` route registration matches user plugin menu owner and `user.read` menu permission | create button uses `user.create`; edit uses `user.update`; role actions require all-of `user.role.read + user.role.assign`; more menu uses `user.disable` / `user.update` | privileged handlers re-check create, edit, disable, and manage-roles before opening dialogs or mutating | list/read guarded by user plugin; role read/write guarded by RBAC plugin | route/menu/action closure is aligned |
| role management page | `role.read` | `server/plugins/rbac/contract/permission.go` | `/access-control/roles` route registration matches RBAC menu owner and `role.read` menu permission | create button uses `role.create`; edit uses `role.update`; assign-permissions uses all-of `permission.read + role.permission.assign`; row more menu now hides permission-missing actions | permission drawer submit gated by `canAssignPermissions`; destructive/status handlers still re-check runtime permission plus lifecycle | role list/read uses `role.read`; permission snapshot uses `permission.read`; permission write uses `role.permission.assign`; status/delete use dedicated write guards | one low-risk drift fixed: permission-only disabled row actions now hide instead of rendering disabled |
| permission management page | `permission.read` | `server/plugins/rbac/contract/permission.go` | `/access-control/permissions` route registration matches RBAC menu owner and `permission.read` menu permission | read-only page only exposes refresh, filters, detail drawer; no write affordance | no extra page-local permission wrapper found | permission list/detail guarded by `permission.read` | closure is aligned |
| user-role snapshot and mutation | `user.role.read`, `user.role.assign` | `server/plugins/rbac/contract/permission.go` | no direct menu; surfaced from user page row and batch actions | both single-user and batch role-management entries require all-of read+assign | `canManageUserRoles()` plus `ensureUserPermission(...)` re-check before dialog open and submit | snapshot `GET /api/users/:id/roles` uses `user.role.read`; all write routes use `user.role.assign` | no visible-but-forbidden drift found |
| role-permission snapshot and mutation | `permission.read`, `role.permission.assign` | `server/plugins/rbac/contract/permission.go` | no direct menu; surfaced from role page assign-permissions action | entry requires all-of permission read + assign | page re-checks `canAssignPermissions` before submit; snapshot load falls back to warning when unavailable | snapshot `GET /api/roles/:id/permissions` uses `permission.read`; write routes use `role.permission.assign` | intentional read/write split is preserved in frontend |

### Batch 2 Evidence-Backed Answers

- All owned frontend permission constants in current scope resolve to canonical backend owners:
  - `web/src/modules/rbac/contract/permissions.ts` maps to `server/plugins/rbac/contract/permission.go`
  - `web/src/modules/user/contract/permissions.ts` maps to `server/plugins/user/contract/permission.go`
- No leftover historical alias or duplicate naming drift remains in owned scope:
  - the historical RBAC frontend alias cleanup remains intact; no `ROLE_PERMISSION_MANAGE`-style alias was reintroduced
  - user-role action gates consistently reuse `userRoleManagePermissionCodes = [user.role.read, user.role.assign]`
- Route visibility and menu semantics stay aligned in owned scope:
  - `web/src/utils/route/bootstrap.ts` only mounts routes backed by bootstrap menu paths and local registrations
  - `/access-control/users` is now confirmed to be backend-owned by `server/plugins/user/plugin_registration.go`
  - `/access-control/overview` is now confirmed to be frontend-owned by `web/src/modules/access-control/bootstrap-routes.ts`
- RBAC page buttons and actions use the expected visibility semantics:
  - create/edit entrypoints are hidden via `v-permission`
  - permission assignment entrypoint requires both read and assign permissions
  - Batch 2 removed one residual permission-only disabled pattern from the role row `More` dropdown
- User page RBAC-related operations match backend guards:
  - role-management entrypoints require both `user.role.read` and `user.role.assign`
  - local submit/open handlers still re-check the same combined permission before mutating
- No obvious frontend-visible but backend-always-forbidden drift was found in owned scope after the role-row fix.
- No obvious owned-scope backend permission exists without a frontend entry where this batch would expect one:
  - `user.session.read` and `user.session.revoke` stay outside this page scope and are not treated as omissions here
  - RBAC read/write management permissions present the expected page or action entrypoints in current scope
- No page-local computed wrappers were found that merely restate canonical permission codes without adding behavior:
  - remaining computed helpers such as `canAssignPermissions()` and `canManageUserRoles()` express multi-permission or
    lifecycle-aware behavior rather than duplicating a single canonical check pointlessly

### Batch 2 Conclusions

- Frontend route, menu, and action permission usage is largely closed against the current backend canonical registry.
- Batch 2 proved and fixed one low-risk owned-scope drift:
  - `web/src/modules/rbac/pages/index.vue` no longer renders RBAC row `More` dropdown entries that are disabled only
    because the viewer lacks the required permission
  - business-state disabled behavior remains intact for builtin roles and delete lifecycle constraints
- The remaining consistency questions are now narrowed for Batch 3:
  - verify the full cross-boundary closure across `access-control` shell routes, user plugin menus, RBAC plugin menus,
    and bootstrap ordering
  - confirm no shared contract drift remains between frontend API paths, menu paths, and backend route/menu owners

## Batch 3 Cross-Boundary Consistency Audit

### Cross-Boundary Closure Matrix

| Capability | Backend permission code | Backend API guard | Backend menu declaration | Frontend constant | Frontend route visibility | Frontend page/action visibility | Error semantics | Tests / validation evidence | Status |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| role list/detail/create/update/status/delete | `role.read`, `role.create`, `role.update`, `role.status.update`, `role.delete` | `GET /api/roles`, `GET /api/roles/:id`, `POST /api/roles`, `POST /api/roles/:id/update`, `POST /api/roles/:id/status`, `POST /api/roles/:id/delete` all use explicit RBAC guards | `/access-control/roles` requires `role.read` in RBAC menu registry | `RBAC_PERMISSION_CODE.ROLE_*` in `web/src/modules/rbac/contract/permissions.ts` | bootstrap menu `/access-control/roles` maps to `RoleList` via RBAC bootstrap route registration and `transformBootstrapMenusToRoutes(...)` | role page uses `v-permission` for create/edit, hides permission-missing row actions, and keeps lifecycle-only disabled semantics for builtin or not-deletable roles | unauthenticated -> `401 auth.token_*`; permission denied -> `403 auth.forbidden`; missing role -> `404 role.not_found`; malformed or stale IDs -> `400 common.invalid_argument`; builtin/enabled/bound lifecycle conflicts remain service-layer rejections | `server/plugins/rbac/plugin_test.go`, `server/internal/httpx/authz_test.go`, `web/src/modules/rbac/pages/index.test.ts`; prior Batch 1 backend validation and Batch 2 web validation | `aligned` |
| permission list/detail/filter/read-only behavior | `permission.read` | `GET /api/permissions`, `GET /api/permissions/:id`, and role-permission snapshot `GET /api/roles/:id/permissions` use `permission.read` | `/access-control/permissions` requires `permission.read` in RBAC menu registry | `RBAC_PERMISSION_CODE.PERMISSION_READ` | bootstrap menu `/access-control/permissions` maps to `PermissionList`; route mounting still depends on bootstrap menu presence | permission page is read-only; role page assignment entry requires `permission.read` together with write permission before opening mutation flow | unauthenticated -> `401`; permission denied -> `403 auth.forbidden`; missing permission detail -> `404 permission.not_found`; malformed query or stale referenced IDs -> `400 common.invalid_argument` | `server/plugins/rbac/plugin_test.go`, `server/internal/httpx/authz_test.go`, `web/src/modules/rbac/pages/index.test.ts` | `aligned` |
| role-permission list/replace/add/remove | `permission.read` for snapshot, `role.permission.assign` for writes | `GET /api/roles/:id/permissions` uses `permission.read`; `POST /api/roles/:id/permissions/{replace|add|remove}` use `role.permission.assign` | none; API-only capability | `RBAC_PERMISSION_CODE.PERMISSION_READ`, `RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN` | no standalone menu; surfaced from the role page under `/access-control/roles` | role page requires all-of read + assign to show assign-permissions entry; drawer submit path preserves `replace | add | remove` semantics | denied -> `403 auth.forbidden`; missing role -> `404 role.not_found`; invalid or stale `permission_ids` -> `400 common.invalid_argument`; this remains a read/write split by design, not drift | `server/plugins/rbac/plugin_test.go`, `web/src/modules/rbac/pages/index.test.ts` cover replace/add/remove UI submission semantics and error localization | `aligned` |
| user-role list/replace/add/remove/batch | `user.role.read` for snapshot, `user.role.assign` for writes | `GET /api/users/:id/roles` uses `user.role.read`; single-user and batch `POST /api/users(/:id)?/roles/{replace|add|remove}` use `user.role.assign` | none; API-only capability | `RBAC_PERMISSION_CODE.USER_ROLE_READ`, `RBAC_PERMISSION_CODE.USER_ROLE_ASSIGN` | no standalone menu; surfaced from the user page under `/access-control/users` | user page manage-roles entry requires all-of read + assign for single-row and batch flows; dialog keeps `replace | add | remove` semantics aligned with backend routes | denied -> `403 auth.forbidden`; self-removal of builtin admin -> dedicated `403 rbac.cannot_remove_own_admin_role`; missing user -> `404 user.not_found`; invalid or stale `role_ids` -> `400 common.invalid_argument` | `server/plugins/rbac/plugin_test.go`, `web/src/modules/user/pages/index.test.ts` cover single-user and batch replace/add/remove flows plus error handling | `aligned` |
| user management manage-roles entry | `user.role.read` + `user.role.assign` for RBAC operations, `user.read` for page visibility | user-role snapshot and mutation routes stay in RBAC plugin guard scope; user list/read stays in user plugin guard scope | `/access-control/users` menu is declared by user plugin with `user.read` | `USER_PERMISSION_CODE.READ`, `RBAC_PERMISSION_CODE.USER_ROLE_READ`, `RBAC_PERMISSION_CODE.USER_ROLE_ASSIGN` | `/access-control/users` bootstrap route registration matches the user plugin menu owner and is filtered by bootstrap permissions | user page create/edit/more entries use user-plugin permissions; manage-roles entry and submit path re-check RBAC read + assign before mutating | route/login failures still separate `401` vs `403`; page-level manage-roles failures localize into dialog or toast without masking backend domain errors | `server/plugins/user/plugin_registration.go`, `server/plugins/user/bootstrap.go`, `web/src/modules/user/bootstrap-routes.ts`, `web/src/modules/user/pages/index.test.ts` | `aligned` |
| dynamic menu bootstrap | permission-specific per menu item; root and overview intentionally blank | menu exposure is bootstrap snapshot filtering, not request guard middleware | backend menu owners are split intentionally: RBAC owns `/access-control`, `/access-control/overview`, `/access-control/roles`, `/access-control/permissions`; user plugin owns `/access-control/users` | frontend consumes backend permission strings through bootstrap snapshot rather than duplicating menu contracts | `transformBootstrapMenusToRoutes(...)` mounts only backend-returned menu paths that also have local route registrations | visible routes and menus are therefore the intersection of backend menu snapshot and owned frontend registrations | menu filtering treats blank permissions as authenticated shell nodes and guarded menus as permission-gated nodes; no denial-reason contract is surfaced in MVP | `server/plugins/user/bootstrap.go`, `server/plugins/user/plugin_test.go`, `web/src/utils/route/bootstrap.test.ts` | `aligned-with-risk` |
| route visibility | inherited from bootstrap menu permissions and local registrations | no direct backend route guard here; actual API protection remains separate and explicit | backend menu path truth is `/access-control/*` with stable ordering rooted in bootstrap snapshot | frontend route constants stay in module contracts and `access-control` bootstrap contract | `/access-control/overview`, `/users`, `/roles`, `/permissions` all have confirmed backend menu owners and frontend route registrations | page entry visibility therefore follows bootstrap menu filtering; actions still need their own permission checks | route visibility is intentionally not the same as action authorization; a visible page can still surface finer-grained action denial | `web/src/utils/route/bootstrap.ts`, `web/src/utils/route/bootstrap.test.ts`, `server/plugins/user/bootstrap.go` | `aligned` |
| button/action visibility | action-specific RBAC and user permission codes | backend still enforces all writes even when frontend hides actions | no menu-level ownership for action-only capabilities | `RBAC_PERMISSION_CODE.*`, `USER_PERMISSION_CODE.*` | no extra route effect | RBAC and user pages now hide permission-missing actions instead of showing permission-only disabled affordances; remaining disabled states reflect business lifecycle only | forbidden API responses still remain the canonical backstop for stale or race-case permissions | `web/src/modules/rbac/pages/index.test.ts`, `web/src/modules/user/pages/index.test.ts`; prior Batch 2 fix confirmed the governed hiding rule | `aligned` |
| builtin role / permission protection | uses ordinary RBAC permissions plus service-layer lifecycle rules | write guards allow authorized callers through to service checks; service/repository layer blocks builtin rename, builtin lifecycle mutation, and self-lockout | none | frontend constants do not invent separate builtin-protection permissions | no separate route; protection applies within visible role and user-role actions | frontend surfaces builtin state as read-only or lifecycle hints; it does not attempt to model a second permission system | self-lockout remains dedicated `403 rbac.cannot_remove_own_admin_role`; builtin rename/delete/disable conflicts stay backend-owned domain errors rather than frontend-only hiding | `server/plugins/rbac/write_service.go`, `server/plugins/rbac/plugin_test.go`, `web/src/modules/rbac/pages/index.test.ts` | `aligned` |
| auth forbidden / unauthorized separation | n/a | `httpx.RequirePermission(...)` separates missing or invalid token from permission denial before handler execution | n/a | frontend consumes backend envelope codes, not its own guessed status semantics | route visibility comes from bootstrap after authentication, while APIs still distinguish authn and authz | page dialogs preserve backend error surfaces for `403` and `404` cases instead of coercing everything into generic UI copy | missing token or invalid token -> `401 auth.token_missing|auth.token_invalid|auth.token_expired`; authenticated but not authorized -> `403 auth.forbidden` with `details.permission` | `server/internal/httpx/authz_test.go`; page tests verify local error handling does not overwrite backend message surfaces | `aligned` |

### Batch 3 Evidence-Backed Answers

- The current MVP cross-boundary closure is runtime-aligned in owned scope:
  - backend permission contracts, backend menu declarations, frontend permission constants, bootstrap route registrations, and page/action visibility checks all point to the same stable `/access-control/*` and permission-code semantics.
- The previously open cross-boundary menu-owner questions are now fully closed:
  - `/access-control/overview` is an RBAC-owned backend menu with an `access-control` frontend route registration.
  - `/access-control/users` is a user-plugin backend menu with a user-module frontend route registration.
- Dynamic menu bootstrap is intentionally split across plugins but still coherent:
  - RBAC owns access-control shell root, overview, roles, and permissions menu declarations.
  - user plugin owns the users menu declaration.
  - user bootstrap code imposes a stable access-control sort order before the frontend turns menus into routes.
- No runtime drift inside the owned scope justified a Batch 3 code fix:
  - role CRUD, permission read-only behavior, role-permission mutation, user-role single-user mutation, user-role batch mutation, and user manage-roles entry all already matched current backend contracts and frontend behavior.
  - the only proven mismatch found this round was documentation drift in `server/plugins/rbac/README.md`, which still described deprecated `assign` wording and replace-only semantics.
- One tiny non-runtime fix was justified and applied:
  - `server/plugins/rbac/README.md` now matches the actual `replace | add | remove` user-role and role-permission contract surface.
- Residual risk remains documented rather than widened into contract hardening:
  - runtime registries and bootstrap assembly still rely on canonical ownership plus tests, not registry-level duplicate or reference enforcement.
  - that is a governance-hardening follow-up, not an MVP blocker for this topic.

### Batch 3 Conclusions

- Batch 3 closes the main cross-boundary consistency question for current MVP RBAC scope.
- Current owned runtime behavior is stable enough to move to Batch 4 archive-readiness and MVP decision work.
- Remaining notes are risk-only, not active drift:
  - registry/menu closure still depends on tests and disciplined contract ownership
  - route visibility remains bootstrap-driven and intentionally separate from per-action authorization
