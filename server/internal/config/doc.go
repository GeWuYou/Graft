// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package config 负责加载 Graft 服务端以环境变量优先的运行时配置。
//
// 该包让 Docker 与本地开发共用同一路径：真实环境变量优先，可选的 .env 仅用于补充未提交到仓库的本地默认值。
package config
