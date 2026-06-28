# eventbus

## 用途

`eventbus` 提供 `server` 当前 MVP 阶段的最小进程内事件总线。

## 职责边界

这个模块负责：

* 提供显式 `Subscribe / Publish` 能力
* 在单个处理器失败或 panic 时记录错误并继续其余处理器
* 作为 `Runtime` 注入给模块的核心基础设施句柄

这个模块不负责：

* 消息持久化
* ack、retry、dead-letter、consumer-group 等 MQ 语义
* 分布式投递、跨进程传输或调度编排

## 主要入口

* `doc.go`：包职责说明
* `bus.go`：最小总线接口与内存实现
* `bus_test.go`：派发顺序、错误聚合与 panic recover 回归

## 关键依赖

* 上游由 `server/internal/app` 创建并注入
* 下游供 `server/modules/*` 在 `Register` / `Boot` 生命周期中发布和订阅事件

## 维护提示

如果未来需求开始依赖重试、持久化或跨进程语义，应先更新 `ai-plan/design/architecture/模块与依赖注入设计.md`
与 `ai-plan/roadmap/MVP实施计划.md`，再决定是否扩展当前接口或引入新的实现层。
