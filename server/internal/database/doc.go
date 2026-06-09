// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package database 负责为核心运行时创建基于 PostgreSQL 的 Ent 客户端。
//
// 数据库资源由 core 统一持有，模块只依赖显式仓储契约，避免各自维护存储连接。
package database
