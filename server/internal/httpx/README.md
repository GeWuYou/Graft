# httpx

## 用途

`httpx` 管理 `server` 的 HTTP 服务外壳，包括路由根、MVP 阶段授权守卫与服务关闭语义。

## 职责边界

这个模块负责：

* 提供运行时使用的 Gin 服务包装
* 管理 `Run` 与 `Shutdown` 的生命周期衔接
* 提供基于稳定请求鉴权上下文的显式权限守卫
* 在业务执行前建立 request/trace correlation context
* 将 typed AppError 映射到既有本地化错误 envelope，并只为遗漏的 internal cause 提供一次 fallback 日志
* 统一恢复未处理 panic，记录真实 stack 与安全的请求上下文
* 记录只包含请求事实、console severity 固定为 `INFO` 的 Access Log
* 拥有 `Access Log` 的保留期语义与定时清理执行边界

这个模块不负责：

* 具体业务路由逻辑
* 最终版认证与 RBAC 模块实现
* 用前端路由元数据替代后端访问控制
* 审计保留期、归档、导出或 retention UI

## 主要入口

* `doc.go`：包职责说明
* `server.go`：HTTP 服务生命周期控制
* `authz.go`：当前 MVP 阶段的请求身份与权限守卫
* `response.go`：request correlation、统一响应与 typed AppError 映射
* `accesslog.go`：Access Log console policy 与 durable request facts
* `*_test.go`：并发启动、关闭与权限约束验证

## 关键依赖

* 上游由 `server/internal/app` 装配并驱动
* 下游供业务模块注册路由并叠加权限约束

## 维护提示

这里当前通过 `moduleapi.AuthService` 与 `moduleapi.Authorizer` 解析 bearer token、补充请求主体并执行后端权限校验。
Request correlation 已由全局 middleware 建立，鉴权不得重建它。Access Log 不解释 4xx/5xx cause；业务 owner 通过
`logger.ReportError` 记录原因，HTTP 只映射安全 descriptor，Recovery 只处理未捕获 panic。
