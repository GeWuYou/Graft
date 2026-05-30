# httpx

## 用途

`httpx` 管理 `server` 的 HTTP 服务外壳，包括路由根、MVP 阶段授权守卫与服务关闭语义。

## 职责边界

这个模块负责：

* 提供运行时使用的 Gin 服务包装
* 管理 `Run` 与 `Shutdown` 的生命周期衔接
* 提供基于稳定请求鉴权上下文的显式权限守卫
* 拥有 `Access Log` 的保留期语义与定时清理执行边界

这个模块不负责：

* 具体业务路由逻辑
* 最终版认证与 RBAC 插件实现
* 用前端路由元数据替代后端访问控制
* 审计保留期、归档、导出或 retention UI

## 主要入口

* `doc.go`：包职责说明
* `server.go`：HTTP 服务生命周期控制
* `authz.go`：当前 MVP 阶段的请求身份与权限守卫
* `*_test.go`：并发启动、关闭与权限约束验证

## 关键依赖

* 上游由 `server/internal/app` 装配并驱动
* 下游供业务插件注册路由并叠加权限约束

## 维护提示

这里当前通过 `pluginapi.AuthService` 与 `pluginapi.Authorizer` 解析 bearer token、构造请求鉴权上下文并执行后端权限校验。后续继续扩展登录与 refresh 能力时，应保留“后端显式校验权限”的原则，而不是退回只依赖前端菜单或路由元数据。
