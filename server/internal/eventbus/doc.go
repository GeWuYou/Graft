// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package eventbus 提供运行时内最小事件总线能力。
//
// 当前 MVP 阶段的 event bus 只负责模块间的进程内解耦通信：显式订阅、
// 显式发布、顺序派发、panic recover 与错误日志记录。它不是消息中间件，
// 也不暴露 ack、retry、consumer-group 等分布式语义。
package eventbus
