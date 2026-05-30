package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/pluginapi"
)

type stubAccessLogRepository struct {
	created []CreateAccessLogInput
}

func (r *stubAccessLogRepository) CreateAccessLog(_ context.Context, input CreateAccessLogInput) (AccessLog, error) {
	r.created = append(r.created, input)
	return normalizeCreateAccessLogInput(input), nil
}

func (r *stubAccessLogRepository) CreateAccessLogs(_ context.Context, inputs []CreateAccessLogInput) ([]AccessLog, error) {
	items := make([]AccessLog, 0, len(inputs))
	for _, input := range inputs {
		r.created = append(r.created, input)
		items = append(items, normalizeCreateAccessLogInput(input))
	}
	return items, nil
}

func (r *stubAccessLogRepository) DeleteAccessLogsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

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
	repo := &stubAccessLogRepository{}
	server := NewServer(zap.New(core), repo)

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

	assertAccessLogEntry(t, entries[0], requestID)

	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
	}
	if repo.created[0].RequestID != requestID {
		t.Fatalf("expected persisted request id %q, got %#v", requestID, repo.created[0])
	}
	if repo.created[0].ResponseSize != nil && *repo.created[0].ResponseSize < 0 {
		t.Fatalf("expected bounded response size when present, got %#v", repo.created[0].ResponseSize)
	}
}

func assertAccessLogEntry(t *testing.T, entry observer.LoggedEntry, requestID string) {
	t.Helper()

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

func TestNewAccessLogMiddlewarePersistsAuthenticatedCanonicalFieldsAndRedactsSensitiveValues(t *testing.T) {
	repo := &stubAccessLogRepository{}
	server := NewServer(zap.NewNop(), repo)

	server.Engine().Use(func(ctx *gin.Context) {
		ctx.Request = ctx.Request.WithContext(pluginapi.WithRequestAuthContext(ctx.Request.Context(), pluginapi.RequestAuthContext{
			User: &pluginapi.CurrentUser{ID: 7, Username: "alice"},
		}))
		ctx.Next()
	})
	server.Engine().POST("/login", func(ctx *gin.Context) {
		ctx.String(http.StatusCreated, "ok")
	})

	request := httptest.NewRequest(http.MethodPost, "/login?password=guessme", nil)
	request.Header.Set("User-Agent", "curl token=secret-value")
	request.ContentLength = 321
	request.RemoteAddr = "198.51.100.12:2345"

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
	}

	record := repo.created[0]
	if record.Method != http.MethodPost || record.Path != "/login" || record.Route != "/login" {
		t.Fatalf("expected canonical route fields, got %#v", record)
	}
	if record.UserID == nil || *record.UserID != 7 || record.Username != "alice" {
		t.Fatalf("expected authenticated user fields, got %#v", record)
	}
	if record.RequestSize == nil || *record.RequestSize != 321 {
		t.Fatalf("expected request size 321, got %#v", record.RequestSize)
	}
	if record.UserAgent != "curl token=[REDACTED]" {
		t.Fatalf("expected redacted user agent, got %q", record.UserAgent)
	}
	if record.OccurredAt.IsZero() {
		t.Fatalf("expected occurred_at to be set, got %#v", record.OccurredAt)
	}
	if record.Path != "/login" {
		t.Fatalf("expected path to omit query string, got %q", record.Path)
	}
}
