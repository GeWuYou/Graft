// Package cli 负责维护 server 进程的显式 Cobra 命令树。
//
// 运行时启动与数据库迁移仍保持为可见的 CLI 子命令，避免普通应用启动时
// 隐式执行 schema 变更。
package cli
