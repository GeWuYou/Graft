package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogAccessSeverityByStatus(t *testing.T) {
	testCases := []struct {
		name   string
		status int
		level  zapcore.Level
	}{
		{name: "success uses info", status: http.StatusOK, level: zapcore.InfoLevel},
		{name: "client error uses warn", status: http.StatusBadRequest, level: zapcore.WarnLevel},
		{name: "server error uses error", status: http.StatusInternalServerError, level: zapcore.ErrorLevel},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			core, recorded := observer.New(zapcore.DebugLevel)
			logger := zap.New(core)

			logAccess(logger, tc.status, zap.Int("status", tc.status))

			entries := recorded.All()
			if len(entries) != 1 {
				t.Fatalf("expected one access log entry, got %d", len(entries))
			}
			if entries[0].Level != tc.level {
				t.Fatalf("expected level %s, got %s", tc.level, entries[0].Level)
			}
		})
	}
}

func TestNewServerAppliesGlobalRequestIDAndAccessLog(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	server := NewServer(zap.New(core))

	server.Engine().GET("/healthz", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("User-Agent", "httpx-test")
	request.RemoteAddr = "203.0.113.9:1234"

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	requestID := recorder.Header().Get(RequestIDHeader)
	if requestID == "" {
		t.Fatal("expected request id header to be populated by global middleware")
	}

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected one access log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "http access" {
		t.Fatalf("expected access log message, got %q", entry.Message)
	}
	if entry.Level != zapcore.InfoLevel {
		t.Fatalf("expected info access log level, got %s", entry.Level)
	}

	fields := entry.ContextMap()
	if fields["requestId"] != requestID || fields["traceId"] != requestID {
		t.Fatalf("expected request and trace ids to match header, got %#v", fields)
	}
	if fields["method"] != http.MethodGet || fields["path"] != "/healthz" || fields["route"] != "/healthz" {
		t.Fatalf("expected stable request identity fields, got %#v", fields)
	}
	if fields["status"] != int64(http.StatusNoContent) {
		t.Fatalf("expected status field, got %#v", fields["status"])
	}
	if fields["clientIp"] != "203.0.113.9" || fields["userAgent"] != "httpx-test" {
		t.Fatalf("expected client metadata fields, got %#v", fields)
	}
	if _, ok := fields["latency"]; !ok {
		t.Fatalf("expected latency field, got %#v", fields)
	}
}

func TestNewServerReusesIncomingRequestIDForRootRoutes(t *testing.T) {
	server := NewServer(zap.NewNop())

	server.Engine().GET("/healthz", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(RequestIDHeader, "req-root-healthz")

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, request)

	if recorder.Header().Get(RequestIDHeader) != "req-root-healthz" {
		t.Fatalf("expected incoming request id to be preserved, got %q", recorder.Header().Get(RequestIDHeader))
	}
}
