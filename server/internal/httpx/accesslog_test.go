// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
)

type stubAccessLogRepository struct {
	created []CreateAccessLogInput
	queries []AccessLogListQuery
	results []AccessLogListResult
	listErr error
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

func (r *stubAccessLogRepository) DeleteAccessLogsBeforeLimit(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func (r *stubAccessLogRepository) ListAccessLogs(_ context.Context, query AccessLogListQuery) (AccessLogListResult, error) {
	r.queries = append(r.queries, query)
	if r.listErr != nil {
		return AccessLogListResult{}, r.listErr
	}
	if len(r.results) > 0 {
		result := r.results[0]
		r.results = r.results[1:]
		return result, nil
	}
	return AccessLogListResult{}, nil
}

func (r *stubAccessLogRepository) GetAccessLogByID(context.Context, uint64) (AccessLog, error) {
	return AccessLog{}, ErrAccessLogNotFound
}

func TestLoadAccessLogRequestAttentionPayloadQueriesErrorsAndSlowRequests(t *testing.T) {
	occurredAt := time.Now().UTC()
	repo := &stubAccessLogRepository{
		results: []AccessLogListResult{
			{
				Total: 2,
				Items: []AccessLog{
					{
						ID:         1,
						Method:     http.MethodGet,
						Path:       "/api/users",
						RequestID:  "req-error",
						StatusCode: http.StatusInternalServerError,
						DurationMS: 120,
						OccurredAt: occurredAt,
					},
				},
			},
			{
				Total: 0,
				Items: []AccessLog{},
			},
			{
				Total: 1,
				Items: []AccessLog{
					{
						ID:         2,
						Method:     http.MethodPost,
						Path:       "/api/audit/logs",
						StatusCode: http.StatusOK,
						DurationMS: accessLogSlowRequestThresholdMS + 1,
						OccurredAt: occurredAt,
					},
				},
			},
		},
	}

	payload, err := LoadAccessLogRequestAttentionPayload(context.Background(), repo)
	if err != nil {
		t.Fatalf("load access-log request attention payload: %v", err)
	}
	if len(repo.queries) != 3 {
		t.Fatalf("expected 4xx, 5xx, and slow request queries, got %#v", repo.queries)
	}
	if len(repo.queries[0].StatusGroups) != 1 || repo.queries[0].StatusGroups[0] != AccessLogStatusGroup4xx {
		t.Fatalf("expected first query to request 4xx status group, got %#v", repo.queries[0])
	}
	if len(repo.queries[1].StatusGroups) != 1 || repo.queries[1].StatusGroups[0] != AccessLogStatusGroup5xx {
		t.Fatalf("expected second query to request 5xx status group, got %#v", repo.queries[1])
	}
	if repo.queries[2].DurationMinMS == nil || *repo.queries[2].DurationMinMS != accessLogSlowRequestThresholdMS {
		t.Fatalf("expected third query to request slow requests, got %#v", repo.queries[2])
	}
	items, ok := payload["items"].([]map[string]any)
	if !ok {
		t.Fatalf("expected alert-list items payload, got %#v", payload["items"])
	}
	if len(items) != 2 {
		t.Fatalf("expected two access-log attention items, got %d", len(items))
	}
	if items[0]["count"] != 2 {
		t.Fatalf("expected error request count to come from repository total, got %#v", items[0])
	}
	if items[0]["route_location"] != accessLogMenuListPath+"?status_group=4xx" {
		t.Fatalf("expected error request group to drill into matching status group, got %#v", items[0])
	}
	expectedSlowRoute := accessLogMenuListPath + "?duration_min_ms=1000"
	if items[1]["route_location"] != expectedSlowRoute {
		t.Fatalf("expected slow request to drill into access-log filters, got %#v", items[1])
	}
}

func TestLoadAccessLogRequestAttentionPayloadReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("list access logs failed")
	_, err := LoadAccessLogRequestAttentionPayload(context.Background(), &stubAccessLogRepository{listErr: expectedErr})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
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
	if repo.created[0].TraceID != requestID {
		t.Fatalf("expected persisted trace id %q, got %#v", requestID, repo.created[0])
	}
	if repo.created[0].ResponseSize != nil && *repo.created[0].ResponseSize < 0 {
		t.Fatalf("expected bounded response size when present, got %#v", repo.created[0].ResponseSize)
	}
}

func TestAccessLogMiddlewareSuppressesConsoleForSuccessButPersists(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	repo := &stubAccessLogRepository{}
	server := NewServerWithOptions(zap.New(core), ServerOptions{
		AccessLog: AccessLogOptions{
			ConsolePolicy: config.AccessLogConsoleErrorOnly,
			SlowThreshold: time.Second,
		},
	}, repo)

	server.Engine().GET("/healthz", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if entries := recorded.All(); len(entries) != 0 {
		t.Fatalf("expected no console access log entries, got %d", len(entries))
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
	}
	if repo.created[0].StatusCode != http.StatusNoContent {
		t.Fatalf("expected persisted success status, got %#v", repo.created[0])
	}
}

func TestAccessLogMiddlewareAutoUsesResolvedConsolePolicy(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	repo := &stubAccessLogRepository{}
	server := NewServerWithOptions(zap.New(core), ServerOptions{
		AccessLog: AccessLogOptions{
			ConsolePolicy: config.AccessLogConsoleAuto,
			SlowThreshold: time.Second,
		},
	}, repo)

	server.Engine().GET("/healthz", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if entries := recorded.All(); len(entries) != 0 {
		t.Fatalf("expected auto policy to suppress local success console access logs, got %d", len(entries))
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
	}
}

func TestAccessLogMiddlewareErrorOnlyLogsClientAndServerErrors(t *testing.T) {
	testCases := []struct {
		name      string
		status    int
		wantLevel zapcore.Level
	}{
		{name: "client error", status: http.StatusBadRequest, wantLevel: zapcore.WarnLevel},
		{name: "server error", status: http.StatusInternalServerError, wantLevel: zapcore.ErrorLevel},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			core, recorded := observer.New(zapcore.DebugLevel)
			repo := &stubAccessLogRepository{}
			server := NewServerWithOptions(zap.New(core), ServerOptions{
				AccessLog: AccessLogOptions{
					ConsolePolicy: config.AccessLogConsoleErrorOnly,
					SlowThreshold: time.Second,
				},
			}, repo)

			server.Engine().GET("/status", func(ctx *gin.Context) {
				ctx.Status(testCase.status)
			})

			recorder := httptest.NewRecorder()
			server.Engine().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))

			if len(repo.created) != 1 {
				t.Fatalf("expected one persisted access log, got %d", len(repo.created))
			}
			entries := recorded.All()
			if len(entries) != 1 {
				t.Fatalf("expected one console access log entry, got %d", len(entries))
			}
			if entries[0].Level != testCase.wantLevel {
				t.Fatalf("expected console level %s, got %s", testCase.wantLevel, entries[0].Level)
			}
		})
	}
}

func TestAccessLogMiddlewareErrorOnlyLogsSlowSuccess(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	repo := &stubAccessLogRepository{}
	server := NewServerWithOptions(zap.New(core), ServerOptions{
		AccessLog: AccessLogOptions{
			ConsolePolicy: config.AccessLogConsoleErrorOnly,
			SlowThreshold: time.Millisecond,
		},
	}, repo)

	server.Engine().GET("/slow", func(ctx *gin.Context) {
		time.Sleep(10 * time.Millisecond)
		ctx.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/slow", nil))

	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
	}
	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected one console access log entry, got %d", len(entries))
	}
	if entries[0].Level != zapcore.InfoLevel {
		t.Fatalf("expected slow success to log at info level, got %s", entries[0].Level)
	}
	if repo.created[0].DurationMS < 1 {
		t.Fatalf("expected persisted duration above threshold, got %#v", repo.created[0])
	}
}

func TestAccessLogMiddlewareNeverLogsToConsoleButPersists(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	repo := &stubAccessLogRepository{}
	server := NewServerWithOptions(zap.New(core), ServerOptions{
		AccessLog: AccessLogOptions{
			ConsolePolicy: config.AccessLogConsoleNever,
			SlowThreshold: time.Millisecond,
		},
	}, repo)

	server.Engine().GET("/slow", func(ctx *gin.Context) {
		time.Sleep(10 * time.Millisecond)
		ctx.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/slow", nil))

	if entries := recorded.All(); len(entries) != 0 {
		t.Fatalf("expected no console access log entries, got %d", len(entries))
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
	}
}

func TestAccessLogMiddlewareAlwaysLogsSuccess(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	repo := &stubAccessLogRepository{}
	server := NewServerWithOptions(zap.New(core), ServerOptions{
		AccessLog: AccessLogOptions{
			ConsolePolicy: config.AccessLogConsoleAlways,
			SlowThreshold: time.Second,
		},
	}, repo)

	server.Engine().GET("/healthz", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected one console access log entry, got %d", len(entries))
	}
	if entries[0].Level != zapcore.InfoLevel {
		t.Fatalf("expected success access log level %s, got %s", zapcore.InfoLevel, entries[0].Level)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
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

func TestNewServerPreservesIncomingTraceIDForRootRoutes(t *testing.T) {
	repo := &stubAccessLogRepository{}
	server := NewServer(zap.NewNop(), repo)

	server.Engine().GET("/healthz", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(RequestIDHeader, "req-root-healthz")
	request.Header.Set("X-Trace-Id", "trace-root-healthz")

	recorder := httptest.NewRecorder()
	server.Engine().ServeHTTP(recorder, request)

	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
	}
	if repo.created[0].TraceID != "trace-root-healthz" {
		t.Fatalf("expected incoming trace id to be preserved, got %#v", repo.created[0])
	}
}

func TestNewAccessLogMiddlewarePersistsAuthenticatedCanonicalFieldsAndRedactsSensitiveValues(t *testing.T) {
	repo, recorder := runAuthenticatedAccessLogMiddlewareRequest(t)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected one persisted access log, got %d", len(repo.created))
	}

	assertAccessLogRecordMatchesAuthenticatedLogin(t, repo.created[0])
}

func runAuthenticatedAccessLogMiddlewareRequest(t *testing.T) (*stubAccessLogRepository, *httptest.ResponseRecorder) {
	t.Helper()

	repo := &stubAccessLogRepository{}
	server := NewServer(zap.NewNop(), repo)

	server.Engine().Use(func(ctx *gin.Context) {
		ctx.Request = ctx.Request.WithContext(moduleapi.WithRequestAuthContext(ctx.Request.Context(), moduleapi.RequestAuthContext{
			User: &moduleapi.CurrentUser{ID: 7, Username: "alice"},
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

	return repo, recorder
}

func assertAccessLogRecordMatchesAuthenticatedLogin(t *testing.T, record CreateAccessLogInput) {
	t.Helper()

	assertCanonicalAccessLogRouteFields(t, record)
	assertAuthenticatedAccessLogUserFields(t, record)
	assertAccessLogRequestMetadata(t, record)
	assertAccessLogTimestamps(t, record)
}

func assertCanonicalAccessLogRouteFields(t *testing.T, record CreateAccessLogInput) {
	t.Helper()

	if record.Method != http.MethodPost || record.Path != "/login" || record.Route != "/login" {
		t.Fatalf("expected canonical route fields, got %#v", record)
	}
}

func assertAuthenticatedAccessLogUserFields(t *testing.T, record CreateAccessLogInput) {
	t.Helper()

	if record.UserID == nil || *record.UserID != 7 || record.Username != "alice" {
		t.Fatalf("expected authenticated user fields, got %#v", record)
	}
}

func assertAccessLogRequestMetadata(t *testing.T, record CreateAccessLogInput) {
	t.Helper()

	if record.RequestSize == nil || *record.RequestSize != 321 {
		t.Fatalf("expected request size 321, got %#v", record.RequestSize)
	}
	if record.UserAgent != "curl token=[REDACTED]" {
		t.Fatalf("expected redacted user agent, got %q", record.UserAgent)
	}
	if record.Path != "/login" {
		t.Fatalf("expected path to omit query string, got %q", record.Path)
	}
}

func assertAccessLogTimestamps(t *testing.T, record CreateAccessLogInput) {
	t.Helper()

	if record.StartedAt.IsZero() {
		t.Fatalf("expected started_at to be set, got %#v", record.StartedAt)
	}
	if record.OccurredAt.IsZero() {
		t.Fatalf("expected occurred_at to be set, got %#v", record.OccurredAt)
	}
	if record.StartedAt.After(record.OccurredAt) {
		t.Fatalf("expected started_at to be <= occurred_at, got started=%s occurred=%s", record.StartedAt, record.OccurredAt)
	}
}
