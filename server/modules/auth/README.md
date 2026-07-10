# auth module

## 用途

`server/modules/auth` 是认证与会话生命周期模块的长期归属边界。

这个目录拥有 token、credential、refresh session、cookie、`/auth/*` 路由与
认证运行时。`auth` 是所有 `moduleapi` 认证 capability 的唯一注册者。
`user` 仅通过 `UserIdentityProvider` 提供用户资料事实，并通过
`UserBootstrapProvider` 提供资料、RBAC、菜单和 locale 快照。

## 职责边界

这个模块长期负责：

* login / refresh / logout / bootstrap 的认证闭环
* access token / refresh token / refresh cookie
* refresh session 的创建、轮换、吊销与当前会话治理
* 受限会话与 `must_change_password` 相关认证生命周期
* 对外暴露 `moduleapi.AuthService` 与 `moduleapi.AuthSessionService`

这个模块不负责：

* 用户资料与用户管理资源
* role / permission / resource 的授权模型
* 默认把认证持久化细节泄漏给其它模块

## 主要入口

当前目录提供：

* `doc.go`：模块边界说明
* `descriptor.go`：compile-time descriptor 骨架
* `module.go`：auth 路由生命周期入口
* `runtime.go`：token 与 refresh cookie runtime helper
* `route_*.go`：`/auth/*` 路由与受限会话 guard
* `store/`：auth-owned credential/session store contract
* `storeent/`：auth-owned Ent-backed persistence
* `contract/`：`/auth/*` 契约 owner 占位
* `migrations/`：auth-owned credential 和 refresh session schema 与数据前向迁移
