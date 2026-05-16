# rbac plugin

## 用途

`server/plugins/rbac` 提供 MVP 阶段最小可用的后端授权与管理能力，用来把用户与权限仓储基线接到真实请求鉴权链路，并为 `web` 暴露真实的角色/权限读取入口。

## 职责边界

这个模块负责：

* 暴露 `pluginapi.Authorizer`
* 基于稳定仓储接口判断请求主体是否拥有所需权限
* 注册 RBAC 只读权限元数据与菜单元数据
* 提供 `GET /api/roles` 与 `GET /api/permissions` 只读接口
* 在当前第二波实施方向中收敛最小写接口：角色创建、角色更新、角色权限分配、用户角色分配

这个模块不负责：

* 超出最小范围的 RBAC 写接口，例如删除角色、禁用用户、批量治理策略或 `super_admin` bypass
* 认证登录、token 签发与刷新
* 把具体存储实现泄漏给其它插件

## 主要入口

* `doc.go`：插件用途说明
* `plugin.go`：插件生命周期与授权服务注册
* `plugin_routes.go`：角色/权限只读路由
* `plugin_write_routes.go`：最小写接口路由
* `read_service.go`：插件内只读管理服务收口
* `write_service.go`：插件内最小写接口服务收口

## 维护提示

后续如果接入完整 RBAC 管理能力，应继续保持：

* handler 不直接扩散 repository 或 ORM 细节
* 权限判断继续统一走 `pluginapi.Authorizer` 与 `httpx.RequirePermission`
* 角色/权限写操作单独通过插件内 service/usecase 收敛，而不是在路由层散落规则
* 当前最小写接口只有在主代理补跑相应 backend validation 后，才能从“实施中”升级为跟踪文档中的完成态
