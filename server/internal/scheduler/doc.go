// Package scheduler 提供当前 MVP 阶段的最小进程内调度器封装。
//
// 该包隔离底层 cron 实现，只暴露显式 RegisterJob / Start / Stop / RemoveJob
// 语义，供 scheduler 插件在生命周期中接管任务声明与运行控制。
package scheduler
