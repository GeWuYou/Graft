// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package main 提供 JWT secret 辅助生成命令的进程入口。
package main

import (
	"fmt"
	"os"

	"graft/server/internal/keygen"
	"graft/server/internal/logger"

	"go.uber.org/zap"
)

//nolint:gosec // 这里是配置键名而不是凭据本身，固定字符串用于输出 `.env` 模板行。
const jwtSecretEnvKey = "GRAFT_AUTH_JWT_SECRET"

// main 输出一行可直接写入 `.env` 的 JWT secret 配置。
func main() {
	bootstrapLogger := logger.NewBootstrap()
	defer func() {
		_ = logger.Close(bootstrapLogger)
	}()

	line, err := keygen.GenerateEnvLine(jwtSecretEnvKey)
	if err != nil {
		bootstrapLogger.Error("generate jwt secret env line failed",
			zap.String("component", "cmd.graft-jwt-secret"),
			zap.String("target", jwtSecretEnvKey),
			zap.Error(err),
		)
		_ = logger.Close(bootstrapLogger)
		os.Exit(1)
	}

	fmt.Println(line)
}
