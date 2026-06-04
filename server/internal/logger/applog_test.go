package logger

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/httpx"
)

type appLoggerSinkRecorder struct {
	records []CreateAppLogInput
	err     error
}

func (r *appLoggerSinkRecorder) CreateAppLog(_ context.Context, input CreateAppLogInput) (AppLogRecord, error) {
	r.records = append(r.records, input)
	if r.err != nil {
		return AppLogRecord{}, r.err
	}
	return AppLogRecord{}, nil
}

func (r *appLoggerSinkRecorder) DeleteAppLogsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *appLoggerSinkRecorder) ListAppLogs(context.Context, AppLogListQuery) (AppLogListResult, error) {
	return AppLogListResult{}, nil
}

func (r *appLoggerSinkRecorder) GetAppLogByID(context.Context, uint64) (AppLogRecord, error) {
	return AppLogRecord{}, ErrAppLogNotFound
}

func TestAppLoggerIncludesRequestCorrelationFields(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	logger := NewAppLogger(zap.New(core)).Named("user.service")
	ctx := httpx.WithRequestAuditContext(context.Background(), httpx.RequestAuditContext{
		RequestID: "req-1",
		TraceID:   "trace-1",
		Route:     "/api/users/:id",
		Method:    "PATCH",
		ClientIP:  "127.0.0.1",
		UserAgent: "curl/8.0",
	})

	logger.Info(ctx, " update user\tfailed ", StringField("operation", " update_user "))

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	if got := fields[FieldRequestID]; got != "req-1" {
		t.Fatalf("expected request_id req-1, got %#v", got)
	}
	if got := fields[FieldTraceID]; got != "trace-1" {
		t.Fatalf("expected trace_id trace-1, got %#v", got)
	}
	if got := fields[FieldComponent]; got != "user.service" {
		t.Fatalf("expected component user.service, got %#v", got)
	}
	if got := fields[FieldOperation]; got != "update_user" {
		t.Fatalf("expected sanitized operation, got %#v", got)
	}
	if entries[0].Message != "update user failed" {
		t.Fatalf("expected sanitized message, got %q", entries[0].Message)
	}
}

func TestAppLoggerRedactsSensitiveFields(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	logger := NewAppLogger(zap.New(core))

	logger.Warn(context.Background(), "login rejected", StringField("access_token", "secret-token"), StringField("cookie", "session=1"))

	fields := observed.All()[0].ContextMap()
	if got := fields["access_token"]; got != redactedValue {
		t.Fatalf("expected redacted access_token, got %#v", got)
	}
	if got := fields["cookie"]; got != redactedValue {
		t.Fatalf("expected redacted cookie, got %#v", got)
	}
}

func TestAppLoggerWithSanitizesFieldKeys(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	logger := NewAppLogger(zap.New(core)).With(StringField("request id", "req-2"), DurationField("latency-ms", 5*time.Millisecond))

	logger.Debug(context.Background(), "debug")
	logger.Info(context.Background(), "info")

	fields := observed.All()[1].ContextMap()
	if got := fields["request_id"]; got != "req-2" {
		t.Fatalf("expected request_id field, got %#v", got)
	}
	if _, ok := fields["latency_ms"]; !ok {
		t.Fatal("expected sanitized latency_ms field")
	}
}

func TestAppLoggerPersistsCanonicalRecordWhenRepositoryConfigured(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	sink := &appLoggerSinkRecorder{}
	logger := NewAppLogger(zap.New(core), WithAppLogRepository(sink)).
		Named("modules.user.route").
		With(StringField("release", " 2026.06 "))
	ctx := httpx.WithRequestAuditContext(context.Background(), httpx.RequestAuditContext{
		RequestID: "req-1",
		TraceID:   "trace-1",
		Route:     "/api/users/:id",
		Method:    "PATCH",
	})

	logger.Error(ctx, " map user response failed ",
		StringField(FieldOperation, " map_user "),
		ErrorField(errors.New("boom")),
		StringField("module", "user"),
		StringField("access_token", "secret"),
		StringField("status_code", "500"),
	)

	if len(observed.All()) != 1 {
		t.Fatalf("expected zap output to remain enabled, got %d entries", len(observed.All()))
	}
	if len(sink.records) != 1 {
		t.Fatalf("expected one persisted app log, got %d", len(sink.records))
	}

	record := sink.records[0]
	if record.Severity != AppLogSeverityError {
		t.Fatalf("expected error severity, got %q", record.Severity)
	}
	if record.Component != "modules.user.route" {
		t.Fatalf("expected named component, got %q", record.Component)
	}
	if record.RequestID != "req-1" || record.TraceID != "trace-1" {
		t.Fatalf("expected request correlation, got %#v", record)
	}
	if record.Operation != "map_user" || record.Error != "boom" {
		t.Fatalf("expected canonical operation and error, got %#v", record)
	}
	if got := record.Fields["module"]; got != "user" {
		t.Fatalf("expected module field, got %#v", record.Fields)
	}
	if got := record.Fields["release"]; got != "2026.06" {
		t.Fatalf("expected inherited release field, got %#v", record.Fields)
	}
	if got := record.Fields["access_token"]; got != redactedValue {
		t.Fatalf("expected redacted access token, got %q", got)
	}
	if _, exists := record.Fields["status_code"]; exists {
		t.Fatalf("expected access-owned status_code to stay out of app-log fields, got %#v", record.Fields)
	}
}

func TestAppLoggerPreservesZapOutputWhenPersistenceFails(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	sink := &appLoggerSinkRecorder{err: errors.New("db down")}
	logger := NewAppLogger(zap.New(core), WithAppLogRepository(sink)).Named("core.app")

	logger.Warn(context.Background(), "startup degraded", StringField(FieldOperation, "boot"))

	entries := observed.All()
	if len(entries) != 2 {
		t.Fatalf("expected original warn plus persistence failure warn, got %d entries", len(entries))
	}
	if entries[0].Message != "startup degraded" {
		t.Fatalf("expected original zap output first, got %q", entries[0].Message)
	}
	if entries[1].Message != "app log persistence failed" {
		t.Fatalf("expected persistence failure log, got %q", entries[1].Message)
	}
}
