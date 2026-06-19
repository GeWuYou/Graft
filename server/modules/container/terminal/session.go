// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package terminal

import "context"

// Size is the stable terminal geometry contract shared by runtime adapters and websocket bridges.
type Size struct {
	Cols uint
	Rows uint
}

// Session defines the minimal terminal lifecycle required by high-risk realtime shells.
type Session interface {
	Start(ctx context.Context, size Size) error
	Write(ctx context.Context, data []byte) error
	Resize(ctx context.Context, size Size) error
	Output() <-chan []byte
	Errors() <-chan error
	Close(ctx context.Context) error
}

