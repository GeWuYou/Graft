// Package main 提供 `graft` 服务器 CLI 的进程入口。
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"graft/server/internal/cli"
)

// main 执行 Graft 的显式 CLI 入口。
//
// 根命令返回错误时直接以非零状态退出，保证脚本与运维入口可以可靠感知失败。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.NewRootCommand().ExecuteContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		log.Fatalf("execute graft command: %v", err)
	}
}
