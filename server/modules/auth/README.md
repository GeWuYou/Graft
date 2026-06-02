# auth module

## 用途

`server/modules/auth` 是认证与会话生命周期模块的长期归属边界。

Phase 2 起，这个目录开始接管 token、refresh session、cookie 与 auth-owned
store/storeent/runtime helper；Phase 3 进一步接管 `/auth/*` 路由注册与 HTTP
运行时所有权。

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

## Phase 1 状态

当前目录当前提供：

* `doc.go`：插件边界说明
* `descriptor.go`：compile-time descriptor 骨架
* `module.go`：auth 路由生命周期入口
* `runtime.go`：token 与 refresh cookie runtime helper
* `route_*.go`：`/auth/*` 路由与受限会话 guard
* `store/`：auth-owned credential/session store contract
* `storeent/`：auth-owned Ent-backed persistence
* `contract/`：`/auth/*` 契约 owner 占位
* `migrations/`：后续 auth 自有迁移目录占位

后续迁移顺序固定为：

1. Phase 4+：前端 `modules/auth` 收口与兼容清理

兼容期内，`server/modules/user/store/auth.go` 与 `user` 侧 token helper 只保留薄桥接。
