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

当前 App Log foundation 约束：

* canonical owner：`server/internal/logger/**`
* severity：`debug` / `info` / `warn` / `error`
* component naming：使用 `module.component` 风格，按调用链显式 `Named`
* request correlation：从请求上下文读取 `request_id` / `trace_id`
* persistence strategy：沿用当前 Zap runtime sink，不在此主题内新增 durable storage
* async behavior：不引入额外异步队列，沿用 Zap 当前写入语义
* sanitization：按字段名脱敏 `password` / `secret` / `token` / `authorization` / `cookie`
* retention boundary：当前仅存在进程日志基线，不在本主题内建立 retention authority
