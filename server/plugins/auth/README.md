# auth plugin

## 用途

`server/plugins/auth` 是认证与会话生命周期插件的长期归属边界。

Phase 1 只建立插件骨架与 capability owner，不迁移现有 `user` 插件里的
token、refresh session、cookie、`/auth/*` 路由或受限会话运行时逻辑。

## 职责边界

这个模块长期负责：

* login / refresh / logout / bootstrap 的认证闭环
* access token / refresh token / refresh cookie
* refresh session 的创建、轮换、吊销与当前会话治理
* 受限会话与 `must_change_password` 相关认证生命周期
* 对外暴露 `pluginapi.AuthService` 与 `pluginapi.AuthSessionService`

这个模块不负责：

* 用户资料与用户管理资源
* role / permission / resource 的授权模型
* 默认把认证持久化细节泄漏给其它插件

## Phase 1 状态

当前目录只提供：

* `doc.go`：插件边界说明
* `descriptor.go`：compile-time descriptor 骨架
* `plugin.go`：最小生命周期骨架
* `contract/`：`/auth/*` 契约 owner 占位
* `migrations/`：后续 auth 自有迁移目录占位

后续迁移顺序固定为：

1. Phase 2：迁移 token/session/cookie/refresh store
2. Phase 3：迁移 `/auth/*` 路由

在那之前，现有 runtime 仍由 `server/plugins/user` 承载。
