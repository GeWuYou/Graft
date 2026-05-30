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
* 把日志写入第三方平台
* 替插件隐藏调用时机

## 主要入口

* `doc.go`：包职责说明
* `logger.go`：logger 构造与关闭逻辑
* `applog.go`：AppLogger 契约、字段与脱敏规则

## 关键依赖

* 由 `server/internal/app` 在 runtime 装配阶段调用
* 依赖 `server/internal/config` 提供日志级别和环境信息
* 通过 `server/internal/httpx` 的请求上下文读取相关关联字段
* 供 core 与插件通过容器或 `plugin.Context` 共享使用

## 维护提示

如果后续需要增加日志采样、输出目的地或 trace 关联字段，应继续收敛在
这个模块中，而不是让插件直接持有不同配置的 Zap 实例。

## AppLogger 采用规则

优先使用 `AppLogger` 的场景：

* runtime / plugin 的普通应用运行日志
* handler / service / job 的失败摘要，但该事件不属于 access / audit / security authority
* 需要自动继承请求 `request_id` / `trace_id` / `route` / `method` 等关联字段的日志
* 需要使用统一字段脱敏与消息清洗规则的日志

允许继续使用 raw `*zap.Logger` 的场景：

* `server/internal/logger/**` 自身的 logger 构造、全局替换、关闭与底层字段拼装
* `server/internal/httpx/**` 的 `Access Log` 与 access-retention runtime
* `server/internal/httpx/**` 的 security-event bridge
* `server/internal/audit/**`、`server/plugins/audit/**` 的 audit-owned 写入与读模型错误
* `server/internal/eventbus/**`、`server/internal/scheduler/**` 这类基础设施级 runtime 实现
* Ent debug hook、CLI bootstrap fallback、测试代码、第三方/生成代码边界

反模式：

* 在 handler / service 内手工补 `requestId`、`traceId`、`route`、`method`
* 为普通应用日志直接散落调用 `zap.String(...)` / `zap.Error(...)` 而绕过 `AppLogger`
* 把 access / audit / security 语义为了“统一接口”强行迁入 `AppLogger`

示例：

* 推荐：`logger.NewAppLogger(base).Named("plugins.user.route").Error(ctx, "map user response failed", logger.StringField("plugin", "user"), logger.ErrorField(err))`
* 例外：`internal/httpx/accesslog.go` 继续直接使用 raw zap 维护 access-log authority
* 例外：`plugins/audit/**` 继续直接使用 raw zap 维护 audit-owned runtime diagnostics

当前 App Log foundation 约束：

* canonical owner：`server/internal/logger/**`
* severity：`debug` / `info` / `warn` / `error`
* component naming：使用 `module.component` 风格，按调用链显式 `Named`
* request correlation：从请求上下文读取 `request_id` / `trace_id`
* persistence strategy：沿用当前 Zap runtime sink，不在此主题内新增 durable storage
* async behavior：不引入额外异步队列，沿用 Zap 当前写入语义
* sanitization：按字段名脱敏 `password` / `secret` / `token` / `authorization` / `cookie`
* retention boundary：当前仅存在进程日志基线，不在本主题内建立 retention authority

## App Log storage authority foundation

本主题内的最小 authority 结论：

* storage mode：`process_output_only`
* retention owner：`none`
* default retention policy：`0`，表示仓库 runtime 当前没有 App Log retention authority
* future durable-store owner：仍然预留给 `server/internal/logger/**`，但只有在后续主题显式批准 schema、repository、operator contract 与 cleanup lifecycle 后才允许落地
* durable-store decision status：`defer-until-operator-workflow`，当前不得把 developer debugging 的 process output 直接升级为仓库内 durable dataset

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
* `AppLogRecord` 是 future durable storage 的 canonical persisted shape
* `AppLogRepository` 只作为 future durable-store boundary 占位，不在本主题内注册 runtime 实现
* retention / archive / purge 仍然不可由仓库 runtime 执行，直到 durable storage authority 被批准
