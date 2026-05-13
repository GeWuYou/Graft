# logger

## 用途

`logger` 负责为 `server` 运行时创建统一的 Zap 日志实例。

## 职责边界

这个模块负责：

* 根据 `config` 中的日志配置初始化结构化 logger
* 约束默认字段、日志级别和输出编码
* 在进程关闭时统一执行日志刷新

这个模块不负责：

* 业务日志格式约定
* 把日志写入第三方平台
* 替插件隐藏调用时机

## 主要入口

* `doc.go`：包职责说明
* `logger.go`：logger 构造与关闭逻辑

## 关键依赖

* 由 `server/internal/app` 在 runtime 装配阶段调用
* 依赖 `server/internal/config` 提供日志级别和环境信息
* 供 core 与插件通过容器或 `plugin.Context` 共享使用

## 维护提示

如果后续需要增加日志采样、输出目的地或 trace 关联字段，应继续收敛在
这个模块中，而不是让插件直接持有不同配置的 Zap 实例。
