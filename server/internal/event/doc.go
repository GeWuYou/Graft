// Package event 提供 server 运行时拥有的通用异步事件分发基础设施。
//
// 本包只定义领域无关的事件 envelope、发布、订阅和本地内存派发语义。
// 业务模块拥有事件类型与载荷 DTO；可靠投递将在后续以 Outbox Repository 实现，
// 不改变业务模块依赖的 Publisher 与 Handler 边界。
package event
