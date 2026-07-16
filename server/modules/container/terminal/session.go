// Package terminal 定义容器运行时与 WebSocket 桥接之间稳定的终端会话契约。
package terminal

import "context"

// Size 是运行时适配器与 WebSocket 桥接共用的终端几何尺寸契约。
type Size struct {
	Cols uint
	Rows uint
}

// Session 定义高风险实时 Shell 所需的最小生命周期；Close 必须能终止未启动和已启动会话。
type Session interface {
	Start(ctx context.Context, size Size) error
	Write(ctx context.Context, data []byte) error
	Resize(ctx context.Context, size Size) error
	Output() <-chan []byte
	Errors() <-chan error
	Close(ctx context.Context) error
}
