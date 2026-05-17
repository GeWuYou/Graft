# rbac plugin

## 用途

`server/plugins/rbac` 提供 MVP 阶段最小可用的后端授权与管理能力，用来把用户与权限仓储基线接到真实请求鉴权链路，并为 `web` 暴露真实的角色、权限和用户角色绑定入口。

## 职责边界

这个模块负责：

* 暴露 `pluginapi.Authorizer`
* 基于稳定仓储接口判断请求主体是否拥有所需权限
* 在 `Register` 阶段向统一 `server/internal/i18n` facade 注册插件内建菜单标题 message，并通过共享菜单 contract 持有 `title_key`
* 注册 RBAC 只读权限元数据与菜单元数据
* 提供 `GET /api/roles`、`GET /api/permissions`、`GET /api/roles/:id/permissions` 与 `GET /api/users/:id/roles` 最小只读接口
* 提供 `POST /api/roles`、`POST /api/roles/:id/update`、`POST /api/roles/:id/permissions/assign` 与 `POST /api/users/:id/roles/assign` 最小写接口

这个模块不负责：

* 超出最小范围的 RBAC 写接口，例如删除角色、禁用用户、批量治理策略或 `super_admin` bypass
* 认证登录、token 签发与刷新
* 把具体存储实现泄漏给其它插件

## 主要入口

* `doc.go`：插件用途说明
* `plugin.go`：插件生命周期与授权服务注册
* `plugin_routes.go`：角色/权限只读路由与角色权限快照路由
* `plugin_write_routes.go`：用户角色只读快照路由与最小写接口路由
* `read_service.go`：插件内只读管理服务收口
* `write_service.go`：插件内最小写接口服务收口

## 维护提示

后续如果接入完整 RBAC 管理能力，应继续保持：

* handler 不直接扩散 repository 或 ORM 细节
* 权限判断继续统一走 `pluginapi.Authorizer` 与 `httpx.RequirePermission`
* 用户角色最小读面只返回稳定 `role_ids` 快照，角色详情继续由 `GET /api/roles` 持有
* 角色权限和用户角色写入都保持 replace 语义，并继续把 `permission_ids` / `role_ids` 作为稳定请求字段
* builtin 角色允许更新展示字段，但不允许通过写接口修改稳定名称
* 角色/权限写操作单独通过插件内 service/usecase 收敛，而不是在路由层散落规则
* 目标用户不存在、角色/权限 ID 无效，以及 TOCTOU 场景下已删除 ID 的错误映射，继续保持当前 focused tests 已覆盖的稳定契约
