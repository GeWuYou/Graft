package cli

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"graft/server/internal/app"
)

type runtimeRunner interface {
	Run(context.Context) error
}

var serveNewRuntime = func() (runtimeRunner, error) {
	return app.NewRuntime()
}
var serveNotifyContext = signal.NotifyContext

// newServeCommand 创建纯运行时启动命令。
//
// 该命令只负责装配并启动运行时，不隐式执行数据库迁移，避免普通启动路径
// 把 schema 变更和服务生命周期混在一起。
func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the Graft HTTP server",
		RunE:  runServe,
	}
}

// runServe 组装运行时并在收到终止信号前保持服务运行。
//
// 它把 CLI 上下文转换为可响应 SIGINT 和 SIGTERM 的运行时上下文，让
// `app.Runtime` 能沿同一条显式生命周期路径完成关闭。
func runServe(cmd *cobra.Command, _ []string) error {
	runtime, err := serveNewRuntime()
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}

	baseCtx := cmd.Context()
	if baseCtx == nil {
		// 测试或嵌入式调用可能没有预置命令上下文，这里回退到后台上下文，
		// 保持运行时入口对调用方显式且稳定。
		baseCtx = context.Background()
	}

	runCtx, stop := serveNotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runtime.Run(runCtx); err != nil {
		return fmt.Errorf("run runtime: %w", err)
	}

	return nil
}
