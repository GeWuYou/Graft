# audit module

## 用途

`server/modules/audit` 提供当前 MVP 阶段的最小审计模块，用于把请求级自动审计和业务主动审计收敛为稳定落盘行为。

## 职责边界

这个模块负责：

* 在 `Register` 阶段挂载 `/api` 请求级自动审计中间件
* 订阅 `eventbus` 上的主动审计事件并写入统一审计记录
* 通过模块自有 `store.AuditRepository` 持久化最小审计字段

这个模块不负责：

* 审计查询 DSL
* 审计归档、分析或回放
* 把业务模块的内部模型直接暴露成公共审计 API

## 主要入口

* `doc.go`：模块用途说明
* `module.go`：生命周期、HTTP 自动审计与 event bus 订阅接线

## 关键依赖

* 依赖 `module.Context` 提供的 `EventBus`、`Router`、`Logger`，并在 Builder 阶段显式解析共享 `*sql.DB`
* 写入逻辑复用 `server/internal/audit`、`server/modules/audit/store` 和模块自有 `storeent` SQL repository

## 维护提示

如果后续要增加审计查询或跨模块主动写入契约，应先确认 `ai-plan/roadmap/MVP实施计划.md` 的阶段范围，再决定是否扩展 `moduleapi` 或新增读侧仓储接口。
