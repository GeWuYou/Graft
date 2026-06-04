# logger

## 用途

`logger` 负责为 `server` 运行时创建统一的 Zap 日志实例。

## 职责边界

这个模块负责：

* 根据 `config` 中的日志配置初始化结构化 logger
* 定义 `AppLogger` 统一契约与应用日志字段基线
* 复用请求上下文中的 `request_id` / `trace_id` 关联信息
* 对日志字段执行最小必要的脱敏与文本清洗
* 约束默认字段、日志级别和输出编码
* 在进程关闭时统一执行日志刷新

这个模块不负责：

* Access Log / Audit Log / Security Event 的领域归属
* App Log Explorer 或查询接口
* 新增 durable storage、归档或 retention runtime
* 在 App Log readiness 主题批准前抢先落地 App Log durable storage
* 把日志写入第三方平台
* 替模块隐藏调用时机

## 主要入口

* `doc.go`：包职责说明
* `logger.go`：logger 构造与关闭逻辑
* `applog.go`：AppLogger 契约、字段与脱敏规则

## 关键依赖

* 由 `server/internal/app` 在 runtime 装配阶段调用
* 依赖 `server/internal/config` 提供日志级别和环境信息
* 通过 `server/internal/httpx` 的请求上下文读取相关关联字段
* 供 core 与模块通过容器或 `module.Context` 共享使用

## 维护提示

如果后续需要增加日志采样、输出目的地或 trace 关联字段，应继续收敛在
这个模块中，而不是让模块直接持有不同配置的 Zap 实例。

## AppLogger 采用规则

优先使用 `AppLogger` 的场景：

* runtime / module 的普通应用运行日志
* handler / service / job 的失败摘要，但该事件不属于 access / audit / security authority
* 需要自动继承请求 `request_id` / `trace_id` / `route` / `method` 等关联字段的日志
* 需要使用统一字段脱敏与消息清洗规则的日志

生产路径约束：

* 所有需要持久化的 App Log 必须从 runtime / module `Context` 注入的 `logger.AppLogger` 获取
* 模块禁止把 `NewAppLogger(base)` 作为默认生产写法；它只允许用于测试或上下文/服务不可用时的降级 fallback
* runtime 只能补充高信号生命周期事件，不应把每个普通 `Info` 都写入 `app_logs`

当前高信号事件包括：

* `app runtime boot completed`
* `app runtime shutdown failed`
* `module boot failed` / `module boot completed`
* `scheduler job failed` / `scheduler job completed` with deletion count
* datasource / connector registration failed

允许继续使用 raw `*zap.Logger` 的场景：

* `server/internal/logger/**` 自身的 logger 构造、全局替换、关闭与底层字段拼装
* `server/internal/httpx/**` 的 `Access Log` 与 access-retention runtime
* `server/internal/httpx/**` 的 security-event bridge
* `server/internal/audit/**`、`server/modules/audit/**` 的 audit-owned 写入与读模型错误
* `server/internal/eventbus/**`、`server/internal/scheduler/**` 这类基础设施级 runtime 实现
* Ent debug hook、CLI bootstrap fallback、测试代码、第三方/生成代码边界

反模式：

* 在 handler / service 内手工补 `requestId`、`traceId`、`route`、`method`
* 为普通应用日志直接散落调用 `zap.String(...)` / `zap.Error(...)` 而绕过 `AppLogger`
* 把 access / audit / security 语义为了“统一接口”强行迁入 `AppLogger`

示例：

* 推荐：从 `module.Context.Services` 解析注入的 `logger.AppLogger`，再 `.Named("modules.user.route").Error(ctx, "map user response failed", logger.StringField("module", "user"), logger.ErrorField(err))`
* 例外：`internal/httpx/accesslog.go` 继续直接使用 raw zap 维护 access-log authority
* 例外：`modules/audit/**` 继续直接使用 raw zap 维护 audit-owned runtime diagnostics

当前 App Log foundation 约束：

* canonical owner：`server/internal/logger/**`
* severity：`debug` / `info` / `warn` / `error`
* component naming：使用 `module.component` 风格，按调用链显式 `Named`
* request correlation：从请求上下文读取 `request_id` / `trace_id`
* persistence strategy：始终保留当前 Zap runtime sink；当 runtime 配置了 logger-owned repository 时，通过 bounded async queue best-effort 写入 `app_logs`
* async behavior：调用方只负责完成 zap 输出和入队；repository sink 超时或失败只写出底层 zap 诊断，不阻断原调用路径
* sanitization：按字段名脱敏 `password` / `secret` / `token` / `authorization` / `cookie`
* retention boundary：App Log durable storage runtime approval 已完成；logger boundary owns repository cleanup lifecycle，archive/export 仍未批准

## App Log storage authority foundation

当前 authority 结论：

* storage mode：`repository_durable_store`
* storage owner：`server/internal/logger/**`
* durable table：PostgreSQL `app_logs`
* retention owner：`server_internal_logger`
* default retention policy：通过 `GRAFT_LOG_APP_LOG_RETENTION` 配置；默认 local/test/dev 为 3 天、staging 为 7 天、production 为 14 天
* runtime switch：`GRAFT_LOG_APP_LOG_PERSIST` 控制 repository sink 是否启用，默认启用
* archive/export：`not-ready`，当前没有第二 durable archive authority

当前 canonical persisted fields 定义为：

* `occurred_at`
* `severity`
* `component`
* `message`
* `operation`
* `request_id`
* `trace_id`
* `route`
* `method`
* `error`
* `fields`

当前 forbidden persisted fields：

* access-log owned：`path`、`status_code`、`request_size`、`response_size`、`client_ip`、`user_agent`
* audit/security owned：`actor_id`、`actor_type`、`resource_type`、`resource_id`、`action`、`decision`、`policy`、`permission`、`audit_id`、`security_event_id`、`session_id`
* identity / credential-like：`user_id`、`username`、`authorization`、`cookie`

repository / service boundary：

* `AppLogger` 仍是运行时应用日志入口
* `AppLogRecord` 是 durable storage 的 canonical persisted shape
* `AppLogRepository` 是 logger-owned durable-store boundary and read model foundation
* retention cleanup 由 `server/internal/logger/**` 注册 `logger.app-log-retention-cleanup` job 执行
* purge 只允许通过 logger-owned retention cleanup over `app_logs`
* `openapi/**` 与 `web/**` 是 Batch 2 downstream consumer 范围；不得在 `web` 侧发明字段或权限语义
