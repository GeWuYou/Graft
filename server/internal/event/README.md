# event

`event` 是 Runtime 拥有的通用异步事件基础设施。

## 边界

* 模块在 `Register` 阶段通过 `Registry` 声明 handler，并通过 `Publisher` 发布领域事件。
* 本地 dispatcher 只提供有界、best-effort 的内存投递；`Receipt` 只表示事件已接收，不表示 consumer 已完成。
* 事件 envelope 不包含审计、日志或通知领域字段。业务 DTO 保持在各自模块边界。
* `eventbus` 仍是同步进程内观察者总线；不要改变它的顺序和错误回传语义。

## 生命周期

Runtime 负责启动、停止接收和有界 drain。handler 必须幂等，因为本地重试会重新执行同一事件的所有 handler；未来 Outbox 会把重试状态收敛到每个 consumer delivery。
