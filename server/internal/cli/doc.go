// Package cli 负责维护 server 进程的显式 Cobra 命令树。
//
// 运行时启动、数据库迁移与最小 smoke 验证仍保持为可见的 CLI 子命令，
// 避免普通应用启动时隐式执行 schema 变更或隐藏验证编排。
package cli
