// Package event 提供 server 运行时拥有的通用异步事件分发基础设施。
//
// 本包定义领域无关的事件 envelope、发布、注册、内存派发和 PostgreSQL Outbox 投递。
// 业务模块拥有事件类型与载荷 DTO；可靠投递保持 Publisher 与 Handler 的稳定边界，
// 并以每个 consumer 的独立 delivery 支持重启和多实例恢复。
package event
