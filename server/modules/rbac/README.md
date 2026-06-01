# rbac plugin

## 用途

`server/modules/rbac` 提供 MVP 阶段最小可用的后端授权与管理能力，用来把用户与权限仓储基线接到真实请求鉴权链路，并为 `web` 暴露真实的角色、权限和用户角色绑定入口。

## 职责边界

这个模块负责：

* 暴露 `pluginapi.Authorizer`
* 基于稳定仓储接口判断请求主体是否拥有所需权限
* 在 `Register` 阶段向统一 `server/internal/i18n` facade 注册插件内建菜单标题 message，并通过共享菜单 contract 持有 `title_key`
* 注册 RBAC 只读权限元数据与菜单元数据
* 提供 `GET /api/roles`、`GET /api/roles/:id`、`GET /api/permissions`、`GET /api/permissions/:id`、`GET /api/roles/:id/permissions` 与 `GET /api/users/:id/roles` 最小只读接口
* 提供 `POST /api/roles`、`POST /api/roles/:id/update`、`POST /api/roles/:id/status`、`POST /api/roles/:id/delete`、`POST /api/roles/:id/permissions/replace|add|remove`、`POST /api/users/:id/roles/replace|add|remove` 与批量 `POST /api/users/roles/replace|add|remove` 写接口
* `POST /api/users/:id/roles/replace` 与批量 `POST /api/users/roles/replace` 在“当前登录用户修改自己”时增加后端硬保护：如果当前角色快照仍包含 builtin `admin`，replace 写入后也必须继续保留该角色；违反时返回 `403 RBAC_CANNOT_REMOVE_OWN_ADMIN_ROLE`

这个模块不负责：

* 超出当前治理的权限元数据 CRUD、复杂审批流或 `super_admin` bypass
* 认证登录、token 签发与刷新
* 把具体存储实现泄漏给其它插件

## 主要入口

* `doc.go`：插件用途说明
* `plugin_registration.go`：插件生命周期与授权服务注册
* `route_read_handlers.go`：角色/权限/详情只读路由与绑定快照路由
* `route_write_handlers.go`：角色状态/删除、角色权限、用户角色与批量用户角色写接口路由
* `read_service.go`：插件内只读管理服务收口
* `write_service.go`：插件内管理写接口服务收口

## 维护提示

后续如果接入完整 RBAC 管理能力，应继续保持：

* handler 不直接扩散 repository 或 ORM 细节
* 权限判断继续统一走 `pluginapi.Authorizer` 与 `httpx.RequirePermission`
* 用户角色最小读面只返回稳定 `role_ids` 快照，角色详情继续由 `GET /api/roles` 持有
* 角色权限和用户角色写入继续保持 `replace | add | remove` 三种稳定语义，并继续把 `permission_ids` / `role_ids` 作为稳定请求字段
* builtin 角色允许更新展示字段，但不允许通过写接口修改稳定名称
* 角色/权限写操作单独通过插件内 service/usecase 收敛，而不是在路由层散落规则
* 目标用户不存在、角色/权限 ID 无效，以及 TOCTOU 场景下已删除 ID 的错误映射，继续保持当前 focused tests 已覆盖的稳定契约
* 当前权限列表仍只暴露 `display` / `description` 回退文案；未来若增加 `display_key`，必须以 additive contract 方式显式演进，不能把当前返回值描述成已经 key-first
* RBAC 现阶段使用插件本地 SQL repository 直连共享 `*sql.DB`，不再通过 alias layer 反向依赖 `server/internal/ent/*`
