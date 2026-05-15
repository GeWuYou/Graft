// Package cronx 提供插件任务使用的调度注册表。
//
// 当前阶段由插件在 Register 阶段显式声明任务元数据与执行入口，再由独立
// scheduler 封装在 Boot 阶段接入底层调度实现。
package cronx
