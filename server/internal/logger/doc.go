// Package logger 负责构造 server 运行时使用的结构化日志能力。
//
// 该包把 Zap 的初始化、AppLogger 契约、请求关联字段拼装、App Log storage
// authority foundation 与关闭语义集中在 core 内部，避免模块各自创建日志实例而
// 破坏统一的字段、级别、存储边界和输出约定。
package logger
