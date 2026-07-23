# event

`event` 是 Runtime 拥有的通用异步事件基础设施。

## 边界

* 模块在 `Register` 阶段通过 `Registry` 声明 handler，并通过 `Publisher` 发布领域事件。
* `DeliveryBestEffort` 只提供有界的内存投递；`DeliveryDurable` 以 PostgreSQL Outbox 保存事件和每个 consumer 的独立投递状态。`Receipt` 只表示平台已接收，不表示 consumer 已完成。
* 事件 envelope 不包含审计、日志或通知领域字段。业务 DTO 保持在各自模块边界。
* `eventbus` 仍是同步进程内观察者总线；不要改变它的顺序和错误回传语义。

## 生命周期

Runtime 负责启动、停止接收和有界 drain。handler 必须幂等：best-effort 重试会重新执行同一事件的所有 handler；durable Outbox 按 consumer 独立 claim、租约恢复与重试，进程崩溃后可能再次执行尚未确认完成的 handler。

## Durable Outbox

`event_outbox` 保存不可变 envelope，`event_deliveries` 以 `(event_id, consumer_id)` 保存 `pending`、`processing` 或 `delivered` 状态。worker 使用 PostgreSQL `FOR UPDATE SKIP LOCKED` claim pending 或过期租约，多个实例可安全并发恢复。

当前 `Publisher` API 保持不变。`DeliveryDurable` 原子写入 Outbox event 和当前注册 consumer 的 delivery；业务事实与 Outbox 必须共用事务时，业务 owner 应使用 Runtime 注入的 `TransactionalPublisher.PublishTx`，不能在业务提交后调用 Publisher 补写。
