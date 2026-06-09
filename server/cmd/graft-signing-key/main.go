// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package main 提供 signing key 辅助生成命令的进程入口。
package main

import (
	"fmt"
	"os"

	"graft/server/internal/keygen"
	"graft/server/internal/logger"

	"go.uber.org/zap"
)

const signingKeyEnvKey = "GRAFT_AUTH_SIGNING_KEY"

// main 输出一行可直接写入 `.env` 的 signing key 配置。
func main() {
	bootstrapLogger := logger.NewBootstrap()
	defer func() {
		_ = logger.Close(bootstrapLogger)
	}()

	line, err := keygen.GenerateEnvLine(signingKeyEnvKey)
	if err != nil {
		bootstrapLogger.Error("generate signing key env line failed",
			zap.String("component", "cmd.graft-signing-key"),
			zap.String("target", signingKeyEnvKey),
			zap.Error(err),
		)
		_ = logger.Close(bootstrapLogger)
		os.Exit(1)
	}

	fmt.Println(line)
}
