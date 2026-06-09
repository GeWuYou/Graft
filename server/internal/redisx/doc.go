// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package redisx 负责创建 core 运行时使用的 Redis 客户端。
//
// Redis 会在模块 Boot 前初始化，便于后续缓存、会话、限流与调度等基础能力共享同一个显式资源句柄。
package redisx
