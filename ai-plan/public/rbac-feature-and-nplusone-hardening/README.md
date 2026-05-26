# RBAC Feature And N+1 Hardening

## 当前状态摘要

- 本轮是 `cross-boundary` 只读审计建图，不直接扩大实现范围。
- 当前 RBAC 真值主要分布在：
  - `server/plugins/rbac/**`
  - `server/plugins/user/**` 中用户角色分配消费面
  - `openapi/**`
  - `web/src/modules/rbac/**`
  - `web/src/modules/user/**` 中用户角色 UI / API
  - `web/src/store/modules/permission.ts`
  - `web/src/app/bootstrap/route-guards.ts`
  - `web/src/utils/route/bootstrap.ts`
- 当前后端 RBAC 已经具备最小管理能力：
  - `Role` 列表、详情、创建、更新、启停、删除
  - `Permission` 列表、详情
  - `RolePermission` 读/写（`replace` / `add` / `remove`）
  - `UserRole` 读/写（单用户与批量 `replace` / `add` / `remove`）
  - 当前登录用户权限判定
  - bootstrap 菜单与权限消费链路
- 当前仍缺少完整 RBAC 管理面：
  - `Permission search/filter` 后端接口
  - `Permission create/update/delete`
  - 角色删除生命周期的更强前端引导与审计视图
- 当前已修复的首要 N+1 风险是前端用户列表角色摘要；列表不再逐条调用 `GET /api/users/{id}/roles`，而是消费 `GET /api/users` 返回的最小 `roles` 摘要。

## RBAC 功能覆盖矩阵

### Permission

| 维度 | 现状 | 证据 | 说明 |
| --- | --- | --- | --- |
| list | 已有 | `server/plugins/rbac/read_service.go` `ListPermissions`；`server/plugins/rbac/route_read_handlers.go` `handleListPermissions`；`openapi/paths/permissions.list.yaml`；`web/src/modules/rbac/api/rbac.ts` `getPermissions`；`web/src/modules/rbac/pages/permissions/index.vue` | 后端/前端/OpenAPI 全链路已接通 |
| detail | 已有 | `server/plugins/rbac/route_read_handlers.go` `handleGetPermission`；`openapi/paths/permissions.detail.yaml`；`web/src/modules/rbac/api/rbac.ts` `getPermissionDetail`；`web/src/modules/rbac/pages/permissions/index.vue` | 权限页已接入详情抽屉与失败回退 |
| search/filter | 后端缺失，前端本地过滤 | `permissions.list.yaml` 无 query 参数；`web/src/modules/rbac/pages/permissions/index.vue` 在前端 `computed` 里按 keyword/category 过滤 | 现状是全量拉取再本地筛选 |
| create | 不应在当前 MVP 默认存在 | `web/src/modules/rbac/pages/permissions/index.vue` 明确 `readonlyNotice` / `readonlyDescription`；`server/plugins/rbac/route_registration.go` 只注册 `PermissionReadPermission` | 当前权限元数据以注册中心/插件声明为 canonical source，不应先做后台 CRUD |
| update | 不应在当前 MVP 默认存在 | 同上 | 否则会与插件声明式权限真值冲突 |
| delete | 不应在当前 MVP 默认存在 | 同上 | 删除声明式权限同样不是当前管理面应承担的真值 |

### Role

| 维度 | 现状 | 证据 | 说明 |
| --- | --- | --- | --- |
| list | 已有 | `ListRoles` / `handleListRoles` / `openapi/paths/roles.list.yaml` / `web/src/modules/rbac/pages/index.vue` | 已接通 |
| detail | 已有 | `managementReader.GetRole`；`handleGetRole`；`openapi/paths/roles.detail.yaml`；`web/src/modules/rbac/api/rbac.ts` `getRoleDetail` | 角色页详情抽屉优先消费 detail 接口，失败时回退列表快照 |
| create | 已有 | `managementWriter.CreateRole`；`handleCreateRoleRoute`；`roles.list.yaml` `post`；`web/src/modules/rbac/api/rbac.ts` `createRole` | 已接通 |
| update | 已有 | `managementWriter.UpdateRole`；`handleUpdateRoleRoute`；`openapi/paths/roles.update.yaml`；`web/src/modules/rbac/api/rbac.ts` `updateRole` | 路径采用 `/api/roles/{id}/update` |
| delete | 已有 | `managementWriter.SoftDeleteRole`；`handleDeleteRoleRoute`；`openapi/paths/roles.delete.yaml`；`web/src/modules/rbac/api/rbac.ts` `deleteRole` | 后端要求“custom + disabled + 无绑定”才能删除；前端现已在提交前提示该生命周期规则 |
| status/enable/disable | 已有 | `managementWriter.SetRoleStatus`；`handleUpdateRoleStatusRoute`；`openapi/paths/roles.status.yaml`；`web/src/modules/rbac/api/rbac.ts` `updateRoleStatus` | 角色页已接通启用/停用操作，删除链路依赖该状态真值 |
| role permissions read | 已有 | `ListRolePermissionBindings`；`handleListRolePermissionBindings`；`openapi/paths/roles.permissions.yaml`；`getRolePermissionBindings` | 返回的是 `permission_ids` 快照 |
| role permissions write | 已有 | `ReplacePermissionsForRole`；`handleAssignRolePermissionsRoute`；`openapi/paths/roles.assign-permissions.yaml`；`assignRolePermissions` | 现状是 replace，不是增量 add/remove |

### UserRole

| 维度 | 现状 | 证据 | 说明 |
| --- | --- | --- | --- |
| 用户拥有的角色 | 已有 | `ListRoleIDsByUserID`；`handleListUserRoleBindings`；`openapi/paths/users.roles.yaml`；`web/src/modules/user/api/user-roles.ts` `getUserRoleBindings` | 返回 `role_ids` 快照 |
| 给用户分配角色 | 已有 | `AddRolesToUser` / `ReplaceRolesForUser`；对应 route handler 与 OpenAPI path；`web/src/modules/user/pages/index.vue` | 前端已显式区分 `replace/add/remove` 语义 |
| 移除用户角色 | 已有 | `RemoveRolesFromUser`；对应 route handler 与 OpenAPI path；`web/src/modules/user/pages/index.vue` | 当前有独立 remove 语义 |
| 替换用户角色 | 已有 | 同上 | 继续保留为完整快照替换 |
| 批量操作 | 已有 | 批量 `users.roles.(replace|add|remove)` path；`web/src/modules/user/pages/index.vue` | 前端批量抽屉已补语义提示，尤其覆盖空 `replace` 的清空风险 |

### 权限菜单 / bootstrap

| 维度 | 现状 | 证据 | 说明 |
| --- | --- | --- | --- |
| 当前用户权限 | 已有 | `server/plugins/rbac/plugin_registration.go` `authorizer.Authorize` 读取 `ListPermissionsByUserID`；`web/src/store/modules/permission.ts` 消费 `bootstrapSnapshot.permissions` | 权限快照由 bootstrap 提供给前端，服务端按请求态重新判定 |
| 当前用户菜单 | 已有 | `web/src/store/modules/permission.ts` `buildAsyncRoutes`；`web/src/utils/route/bootstrap.ts` | 菜单来自 bootstrap `menus` |
| 动态菜单 `title_key` / fallback | 已有兼容治理 | `web/src/utils/route/bootstrap.ts` 以 `title_key` 为主，保留 `title` 兼容回退；`ai-plan/design/前端架构设计.md` 与 `契约治理` 已冻结规则 | 现状不是第二真值，但仍保留兼容 fallback |

## 接口覆盖矩阵

| 语义 | Method | Path | 后端处理 | 前端消费 | 状态 |
| --- | --- | --- | --- | --- | --- |
| List roles | `GET` | `/api/roles` | `handleListRoles` | `modules/rbac/api/rbac.ts` `getRoles`；角色页；用户角色页角色目录 | 已有 |
| Create role | `POST` | `/api/roles` | `handleCreateRoleRoute` | `createRole`；角色页创建抽屉 | 已有 |
| Update role | `POST` | `/api/roles/{id}/update` | `handleUpdateRoleRoute` | `updateRole`；角色页编辑抽屉 | 已有 |
| Read role permissions | `GET` | `/api/roles/{id}/permissions` | `handleListRolePermissionBindings` | `getRolePermissionBindings`；角色页权限抽屉 | 已有 |
| Replace role permissions | `POST` | `/api/roles/{id}/permissions/assign` | `handleAssignRolePermissionsRoute` | `assignRolePermissions`；角色页权限抽屉 | 已有 |
| List permissions | `GET` | `/api/permissions` | `handleListPermissions` | `getPermissions`；权限页；角色页权限目录 | 已有 |
| Read user roles | `GET` | `/api/users/{id}/roles` | `handleListUserRoleBindings` | `getUserRoleBindings`；用户列表摘要；用户角色抽屉 | 已有 |
| Replace user roles | `POST` | `/api/users/{id}/roles/assign` | `handleAssignUserRolesRoute` | `assignUserRoles`；用户角色抽屉 | 已有 |
| Get role detail | `GET` | `/api/roles/{id}` | `handleGetRole` | `getRoleDetail`；角色页详情抽屉 | 已有 |
| Update role status | `POST` | `/api/roles/{id}/status` | `handleUpdateRoleStatusRoute` | `updateRoleStatus`；角色页更多操作 | 已有 |
| Delete role | `POST` | `/api/roles/{id}/delete` | `handleDeleteRoleRoute` | `deleteRole`；角色页更多操作 | 已有 |
| Get permission detail | `GET` | `/api/permissions/{id}` | `handleGetPermission` | `getPermissionDetail`；权限页详情抽屉 | 已有 |
| Permission write | - | - | 未实现 | 未消费 | 按当前治理不应优先存在 |
| User role add/remove delta | - | - | 未实现 | 未消费 | 缺失 |
| User role bulk | - | - | 未实现 | 未消费 | 缺失 |

## OpenAPI 覆盖矩阵

| Path | OperationId | 语义 | 现状 |
| --- | --- | --- | --- |
| `/api/roles` | `getRoles` | 角色列表 | 已覆盖 |
| `/api/roles` | `postRoles` | 创建角色 | 已覆盖 |
| `/api/roles/{id}/update` | `postRoleUpdate` | 更新角色 | 已覆盖 |
| `/api/roles/{id}/permissions` | `getRolePermissions` | 读取角色权限快照 | 已覆盖 |
| `/api/roles/{id}` | `getRole` | 角色详情 | 已覆盖 |
| `/api/roles/{id}/status` | `postRoleStatus` | 更新角色状态 | 已覆盖 |
| `/api/roles/{id}/delete` | `postRoleDelete` | 删除角色 | 已覆盖 |
| `/api/roles/{id}/permissions/replace` | `postRolePermissionsReplace` | 替换角色权限 | 已覆盖 |
| `/api/roles/{id}/permissions/add` | `postRolePermissionsAdd` | 追加角色权限 | 已覆盖 |
| `/api/roles/{id}/permissions/remove` | `postRolePermissionsRemove` | 移除角色权限 | 已覆盖 |
| `/api/permissions` | `getPermissions` | 权限列表 | 已覆盖 |
| `/api/permissions/{id}` | `getPermission` | 权限详情 | 已覆盖 |
| `/api/users/{id}/roles` | `getUserRoles` | 读取用户角色快照 | 已覆盖 |
| `/api/users/{id}/roles/replace` | `postUserRolesReplace` | 替换用户角色 | 已覆盖 |
| `/api/users/{id}/roles/add` | `postUserRolesAdd` | 追加用户角色 | 已覆盖 |
| `/api/users/{id}/roles/remove` | `postUserRolesRemove` | 移除用户角色 | 已覆盖 |
| `/api/users/roles/replace` | `postUsersRolesReplace` | 批量替换用户角色 | 已覆盖 |
| `/api/users/roles/add` | `postUsersRolesAdd` | 批量追加用户角色 | 已覆盖 |
| `/api/users/roles/remove` | `postUsersRolesRemove` | 批量移除用户角色 | 已覆盖 |

OpenAPI 缺口：

- 未覆盖 permission write path
- 未覆盖权限搜索/过滤 query contract

## generated type consumption 覆盖矩阵

### Server generated consumption

| 消费点 | generated 类型 | 作用 |
| --- | --- | --- |
| `server/plugins/rbac/route_read_handlers.go` | `generated.RoleListResponse` | 角色列表响应 |
| `server/plugins/rbac/route_read_handlers.go` | `generated.PermissionListResponse` | 权限列表响应 |
| `server/plugins/rbac/route_read_handlers.go` | `generated.RolePermissionBindingResponse` | 角色权限快照响应 |
| `server/plugins/rbac/route_read_handlers.go` | `generated.UserRoleBindingResponse` | 用户角色快照响应 |
| `server/plugins/rbac/route_write_handlers.go` | `rbacopenapi.PostRolesJSONRequestBody` | 创建角色请求体 |
| `server/plugins/rbac/route_write_handlers.go` | `rbacopenapi.PostRoleUpdateJSONRequestBody` | 更新角色请求体 |
| `server/plugins/rbac/route_write_handlers.go` | `rbacopenapi.PostRolePermissionAssignJSONRequestBody` | 替换角色权限请求体 |
| `server/plugins/rbac/route_write_handlers.go` | `rbacopenapi.PostUserRolesAssignJSONRequestBody` | 替换用户角色请求体 |

说明：

- server 端 generated 主要用于 operation wrapper / request body / response shape 对齐。
- 当前 handler 里保留了“generated-operation 空实现 wrapper”作为 compile-time 对齐点，但没有引入 runtime SDK。

### Web generated consumption

| 消费点 | generated 类型 | 作用 |
| --- | --- | --- |
| `web/src/modules/rbac/api/rbac.ts` | `paths[...]` | RBAC API request/response typing |
| `web/src/modules/user/api/user-roles.ts` | `paths[...]` | 用户角色 API request/response typing |
| `web/src/modules/rbac/contract/role.ts` | `components['schemas']` | `RoleListItem` / `RoleListResponse` / `UserRoleBindingResponse` |
| `web/src/modules/rbac/types/rbac.ts` | `components['schemas']` | 创建/更新角色、替换角色权限 DTO |
| `web/src/modules/rbac/types/permission.ts` | `components['schemas']` | 权限列表 DTO |
| `web/src/modules/auth/contract/types.ts` | `components['schemas']['BootstrapResponse']` 等 | bootstrap 菜单/权限快照消费 |

结论：

- generated type 已经在 web 里进入真实消费面，不只是旁路类型。
- RBAC 模块与用户角色 API 基本遵守“模块 API 层消费 generated 类型，页面不直调 request.ts”治理。

## 前端页面 / API 覆盖矩阵

| 前端面 | 页面 / 文件 | API 调用 | 状态 |
| --- | --- | --- | --- |
| 角色管理页 | `web/src/modules/rbac/pages/index.vue` | `getRoles` / `getPermissions` / `getRolePermissionBindings` / `createRole` / `updateRole` / `assignRolePermissions` | 已有 |
| 角色状态/删除反馈 | `web/src/modules/rbac/pages/index.vue` | `updateRoleStatus` / `deleteRole` | 已有；详情抽屉与删除前提示会直接暴露 lifecycle 规则 |
| 权限管理页 | `web/src/modules/rbac/pages/permissions/index.vue` | `getPermissions` / `getPermissionDetail` | 已有，只读详情 |
| 用户管理页角色摘要 | `web/src/modules/user/pages/index.vue` | 消费 `getUsers` 返回的 `roles` 摘要；抽屉仍单次调用 `getUserRoleBindings` | 已修复列表级 N+1 |
| 用户角色分配抽屉 | `web/src/modules/user/pages/index.vue` | `getRoles` / `getUserRoleBindings` / `assignUserRoles` | 已有 |
| RBAC 动态路由注册 | `web/src/modules/rbac/bootstrap-routes.ts` | 消费 bootstrap 路径 | 已有 |
| 权限快照 store | `web/src/store/modules/permission.ts` | 消费 bootstrap `permissions` / `menus` | 已有 |
| 路由守卫 bootstrap 恢复 | `web/src/app/bootstrap/route-guards.ts` | `ensureBootstrap` / `bootstrap` | 已有 |

前端缺口：

- 没有权限写操作 UI，且按当前治理不应先补
- 角色删除前只有最小提示，仍缺少更强的跨页审计视图

## N+1 风险矩阵

| 链路 | 风险级别 | 现状 | 证据 | 判断 |
| --- | --- | --- | --- | --- |
| 角色列表是否逐条查权限 | 低 | 单次 `ListRoles` 查询；权限数量与用户数量通过 SQL 子查询聚合，不逐条再发 query | `server/plugins/rbac/storeent/repository.go` `ListRoles` | 不是典型 N+1 |
| 用户列表是否逐条查角色 | 低 | 用户列表直接消费 `GET /api/users` 的内嵌最小 `roles` 摘要；角色抽屉保留单用户读取 | `server/plugins/user/route_user_handlers.go`；`web/src/modules/user/pages/index.vue` | 列表级 N+1 已消除 |
| bootstrap 是否逐条查菜单/权限 | 低 | 前端消费单个 bootstrap snapshot；未见前端逐条菜单/权限再请求 | `web/src/store/modules/permission.ts` / `web/src/app/bootstrap/route-guards.ts` | 当前前端侧不是 N+1 |
| role detail 是否重复查 permission | 中 | 角色页加载时并发一次 `getRoles + getPermissions`；打开权限抽屉时再单独 `getRolePermissionBindings(role.id)` | `web/src/modules/rbac/pages/index.vue` | 不是列表级 N+1，但 detail 抽屉每次打开都会额外读绑定快照 |
| user role assignment 是否重复查 role | 中 | 用户页会先拉一次 role catalog；打开用户角色抽屉还会再读一次当前用户 role bindings | `web/src/modules/user/pages/index.vue` | 单用户操作可接受，不是最紧急风险 |
| 服务端权限判定是否逐请求重复查 permission | 中 | `authorizer.Authorize` 每次鉴权时调用 `ListPermissionsByUserID` | `server/plugins/rbac/plugin_registration.go` | 这是按请求级重复读取，不是单请求内 N+1，但后续可能是热路径性能点 |
| `ReplaceRolesForUser` / `ReplacePermissionsForRole` 是否存在逐 ID roundtrip | 中 | 事务内对每个 relation ID 都做 `bindingExists` 检查并可能 insert | `server/plugins/rbac/storeent/repository.go` `replaceStableAssignments` 调用配置 | 写入链路可能随 ID 数量线性放大，但不是当前最先暴露的读侧 N+1 |

## 推荐后续批次顺序

### Batch 1: 消除最明确的用户列表角色摘要 N+1

- 目标：
  - 用一个聚合读接口替换用户列表逐条 `GET /api/users/{id}/roles`
- 建议方向：
  - 优先让用户列表接口直接返回最小 `role summary`
  - 或新增受控的 `batch user-role summary` 接口
- 原因：
  - 这是当前最直观、最稳定、最容易验证的性能缺口
  - 不要求先扩角色/权限完整 CRUD
- 当前结果：
  - 已完成。`GET /api/users` 现在返回最小 `roles` 摘要，后端通过单次批量查询读取当前列表用户的角色集合。
- Guardrail：
  - 用户列表页不得重新引入逐行 `getUserRoleBindings(user.id)`、`Promise.allSettled(userItems.map(...))` 或等价的每行角色读取扇出。
  - 列表态角色摘要必须继续来自 `GET /api/users` 的内嵌 `roles` 字段；`GET /api/users/{id}/roles` 只保留给单用户抽屉/详情态读取。

### Batch 2: 补齐 Role detail canonical read

- 目标：
  - 增加 `GET /api/roles/{id}` 或等价 detail contract
- 原因：
  - 当前前端 detail 只是列表行抽屉，缺少独立详情真值
  - 后续 delete/status/audit 等能力都更适合以 detail 为中心扩展

### Batch 3: 统一 UserRole 语义命名与接口形态

- 目标：
  - 明确当前 `assign` 实际是 `replace`
  - 决定是否需要补 `add/remove` 增量接口
- 原因：
  - 现在 OpenAPI / 前端命名有“assign”字样，但语义是快照替换
  - 这是治理与可维护性问题，先于批量操作

### Batch 4: 评估是否需要 Role delete / status

- 目标：
  - 基于产品语义决定是补 delete，还是补 disable，还是都不补
- 原因：
  - 当前没有 role status 字段；若直接加 delete/status，需先明确 lifecycle 语义

### Batch 5: 批量用户角色操作

- 目标：
  - 仅在前几批接口语义稳定后，再补 batch assign / replace
- 原因：
  - 用户页已有 disabled batch bar，说明需求存在，但不应在基础 query / detail / 语义未稳前先做

## 审计结论

- 当前 RBAC 已经脱离“只做后端授权判定”的极小状态，具备一条最小管理闭环。
- 但它仍是“管理面半成品”：
  - `Role` 只有 list/create/update
  - `Permission` 只有只读 list
  - `UserRole` 只有 snapshot read/replace
- 权限管理只读是合理的当前治理选择，不应被误判成缺陷。
- 当前最值得优先落地的不是扩更多 CRUD，而是先消除用户列表角色摘要的 N+1，并补一个 canonical `role detail` 读接口。
