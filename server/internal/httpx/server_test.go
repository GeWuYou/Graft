package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/moduleapi"
)

func TestRecoveryMiddlewareLogsPanicStackAndReturnsSafeEnvelope(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	server := NewServer(zap.New(core))
	server.Engine().GET("/panic", func(ctx *gin.Context) {
		requestCtx := moduleapi.WithRequestAuthContext(ctx.Request.Context(), moduleapi.RequestAuthContext{
			User: &moduleapi.CurrentUser{ID: 7},
		})
		ctx.Request = ctx.Request.WithContext(requestCtx)
		panic("runtime secret detail")
	})

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	body := recorder.Body.Bytes()
	var payload ErrorResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode recovery response: %v", err)
	}
	if payload.Code != "COMMON_INTERNAL_ERROR" || payload.MessageKey != "common.internal_error" {
		t.Fatalf("expected common internal error envelope, got %#v", payload)
	}
	if strings.Contains(string(body), "runtime secret detail") {
		t.Fatalf("expected panic detail to stay out of response: %s", body)
	}

	entries := observed.All()
	if len(entries) != 2 {
		t.Fatalf("expected panic ERROR and access INFO, got %d entries", len(entries))
	}
	if entries[0].Message != "panic recovered" || entries[0].Level != zapcore.ErrorLevel {
		t.Fatalf("expected panic error first, got %#v", entries[0])
	}
	fields := entries[0].ContextMap()
	if fields["request_id"] == "" || fields["trace_id"] == "" || fields["user_id"] != uint64(7) {
		t.Fatalf("expected correlation and numeric user id, got %#v", fields)
	}
	if _, ok := fields["stacktrace"]; !ok {
		t.Fatalf("expected captured panic stack frames, got %#v", fields)
	}
	if entries[1].Message != "http access" || entries[1].Level != zapcore.InfoLevel {
		t.Fatalf("expected factual INFO access log, got %#v", entries[1])
	}
}

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

	if _, err := server.Start("127.0.0.1:0"); err == nil {
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
