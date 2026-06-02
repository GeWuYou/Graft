# scheduler module

## 用途

`server/modules/scheduler` 提供当前 MVP 阶段的最小调度模块，用来把各模块在 `cron registry` 中声明的任务装配到统一运行期调度器。

## 职责边界

这个模块负责：

* 在 `Register` 阶段读取 `cron registry` 快照并装配全部任务
* 在 `Boot` / `Shutdown` 阶段统一启动和停止调度器
* 复用 `server/internal/scheduler` 隔离底层 `robfig/cron/v3`

这个模块不负责：

* 分布式调度
* 持久化任务恢复
* 工作流编排
* 让业务模块直接依赖第三方 cron 实现

## 主要入口

* `doc.go`：模块用途说明
* `module.go`：生命周期与调度器装配

## 关键依赖

* 依赖 `module.Context` 提供的 `CronRegistry` 与 `Logger`
* 运行时控制委托给 `server/internal/scheduler`

## 维护提示

如果后续需要动态任务管理，应优先扩展 `server/internal/scheduler` 的稳定接口，再评估是否要向其它模块暴露运行期 `Scheduler` 能力。
