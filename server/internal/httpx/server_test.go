// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestRunRejectsConcurrentStart 验证生命周期保护会拒绝第二次启动。
//
// 这里直接占用运行槽位，而不是依赖真实监听端口，避免测试结果受到
// 沙箱网络能力或本地监听时序的影响。
func TestRunRejectsConcurrentStart(t *testing.T) {
	server := NewServer(nil)
	running := &http.Server{ReadHeaderTimeout: time.Second}
	if err := server.bindRunningServer(running); err != nil {
		t.Fatalf("bind running server: %v", err)
	}

	if err := server.Run(context.Background(), "127.0.0.1:0"); err == nil {
		t.Fatal("expected concurrent run to fail")
	} else if err.Error() != "http server already running" {
		t.Fatalf("expected already running error, got %v", err)
	}
}

// TestDetachRunningServerClearsPointer 验证生命周期清理只会移除一次运行指针。
//
// 这个断言覆盖 Shutdown 内部依赖的“摘除后不再可见”语义，确保重复清理
// 不会拿到旧指针并尝试再次关闭同一个服务实例。
func TestDetachRunningServerClearsPointer(t *testing.T) {
	server := NewServer(nil)
	running := &http.Server{ReadHeaderTimeout: time.Second}
	if err := server.bindRunningServer(running); err != nil {
		t.Fatalf("bind running server: %v", err)
	}

	first := server.detachRunningServer()
	if first != running {
		t.Fatalf("expected first detach to return bound server, got %v", first)
	}

	second := server.detachRunningServer()
	if second != nil {
		t.Fatalf("expected second detach to observe empty state, got %v", second)
	}
}
