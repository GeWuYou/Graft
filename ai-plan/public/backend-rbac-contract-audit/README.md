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
