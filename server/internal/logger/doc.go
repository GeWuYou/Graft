// Package logger 负责构造 server 运行时使用的结构化日志能力。
//
// 该包把 Zap 的初始化与关闭语义集中在 core 内部，避免插件各自创建
// 日志实例而破坏统一的字段、级别和输出约定。
package logger
