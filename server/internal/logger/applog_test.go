package logger

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/logger/logsafe"
)

type appLoggerSinkRecorder struct {
	mu      sync.Mutex
	records []CreateAppLogInput
	err     error
	block   chan struct{}
	seen    chan struct{}
}

type appLogEventPublisherRecorder struct {
	event event.Event
	err   error
}

func (r *appLogEventPublisherRecorder) Publish(context.Context, event.Event, event.PublishOptions) (event.Receipt, error) {
	return event.Receipt{}, errors.New("unexpected blocking publish")
}

func (r *appLogEventPublisherRecorder) PublishAsync(current event.Event, _ event.PublishOptions) (event.Receipt, error) {
	r.event = current
	if r.err != nil {
		return event.Receipt{}, r.err
	}
	return event.Receipt{EventID: current.ID, Delivery: event.DeliveryBestEffort}, nil
}

func (r *appLogEventPublisherRecorder) PublishBatch(context.Context, []event.Event, event.PublishOptions) event.BatchReceipt {
	return event.BatchReceipt{}
}

func newAppLoggerSinkRecorder() *appLoggerSinkRecorder {
	return &appLoggerSinkRecorder{seen: make(chan struct{}, 1)}
}

func newEnabledAppLogTestLogger() *zap.Logger {
	core, _ := observer.New(zapcore.DebugLevel)
	return zap.New(core)
}

func (r *appLoggerSinkRecorder) CreateAppLog(ctx context.Context, input CreateAppLogInput) (AppLogRecord, error) {
	if r.block != nil {
		select {
		case <-r.block:
		case <-ctx.Done():
			return AppLogRecord{}, ctx.Err()
		}
	}

	r.mu.Lock()
	r.records = append(r.records, input)
	r.mu.Unlock()
	r.notifySeen()
	if r.err != nil {
		return AppLogRecord{}, r.err
	}
	return AppLogRecord{}, nil
}

func (r *appLoggerSinkRecorder) notifySeen() {
	if r.seen == nil {
		return
	}
	select {
	case r.seen <- struct{}{}:
	default:
	}
}

func (r *appLoggerSinkRecorder) waitRecord(t *testing.T) CreateAppLogInput {
	t.Helper()

	if r.seen != nil {
		select {
		case <-r.seen:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for app log persistence")
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.records) == 0 {
		t.Fatal("expected persisted app log record")
	}
	return r.records[0]
}

func (r *appLoggerSinkRecorder) recordCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.records)
}

func (r *appLoggerSinkRecorder) DeleteAppLogByID(context.Context, uint64) (bool, error) {
	return false, nil
}

func (r *appLoggerSinkRecorder) DeleteAppLogsByIDs(context.Context, []uint64) (int64, error) {
	return 0, nil
}

func (r *appLoggerSinkRecorder) DeleteAppLogsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *appLoggerSinkRecorder) DeleteAppLogsBeforeLimit(context.Context, time.Time, int) (int64, error) {
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
	sink := newAppLoggerSinkRecorder()
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

	record := sink.waitRecord(t)
	if record.Severity != AppLogSeverityError {
		t.Fatalf("expected error severity, got %q", record.Severity)
	}
	if record.Category != defaultAppLogCategory {
		t.Fatalf("expected application default category %q, got %q", defaultAppLogCategory, record.Category)
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

func TestAppLoggerPublishesJSONEventForPersistence(t *testing.T) {
	publisher := &appLogEventPublisherRecorder{}
	appLogger := NewAppLogger(zap.NewNop(), WithAppLogEventPublisher(publisher)).Named("core.app")
	ctx := httpx.WithRequestAuditContext(context.Background(), httpx.RequestAuditContext{RequestID: "request-1"})

	appLogger.Error(ctx, "database unavailable", ErrorField(errors.New("connection refused")))

	if publisher.event.Type != AppLogPersistEventType {
		t.Fatalf("expected event type %q, got %q", AppLogPersistEventType, publisher.event.Type)
	}
	if publisher.event.Version != appLogPersistEventVersion || publisher.event.Source != appLogPersistEventSource {
		t.Fatalf("expected stable event envelope, got %#v", publisher.event)
	}
	var payload appLogPersistEventPayload
	if err := json.Unmarshal(publisher.event.Payload, &payload); err != nil {
		t.Fatalf("decode app log event payload: %v", err)
	}
	if payload.Record.Component != "core.app" || payload.Record.Error != "connection refused" {
		t.Fatalf("expected canonical record in event payload, got %#v", payload.Record)
	}
	if publisher.event.CorrelationID != "request-1" || publisher.event.IdempotencyKey != "request-1" {
		t.Fatalf("expected request correlation keys, got %#v", publisher.event)
	}
}

func TestAppLogEventHandlerPersistsDecodedRecord(t *testing.T) {
	repo := newAppLoggerSinkRecorder()
	payload, err := json.Marshal(appLogPersistEventPayload{Record: CreateAppLogInput{
		OccurredAt: time.Date(2026, time.July, 23, 10, 0, 0, 0, time.UTC),
		Severity:   AppLogSeverityWarn,
		Category:   CategoryApplication,
		Component:  "core.app",
		Message:    "degraded",
	}})
	if err != nil {
		t.Fatalf("encode app log event payload: %v", err)
	}

	err = NewAppLogEventHandler(repo).Handle(context.Background(), event.Event{Payload: payload})
	if err != nil {
		t.Fatalf("handle app log event: %v", err)
	}
	record := repo.waitRecord(t)
	if record.Severity != AppLogSeverityWarn || record.Message != "degraded" {
		t.Fatalf("expected decoded app log record, got %#v", record)
	}
}

func TestAppLoggerCategoryPersistsRegisteredCategory(t *testing.T) {
	sink := newAppLoggerSinkRecorder()
	core, _ := observer.New(zapcore.InfoLevel)
	logger := NewAppLogger(zap.New(core), WithAppLogRepository(sink)).Category(CategoryRuntimeMetrics)

	logger.Info(context.Background(), "sampled runtime metric")
	record := sink.waitRecord(t)
	if record.Category != CategoryRuntimeMetrics {
		t.Fatalf("expected selected category %q, got %q", CategoryRuntimeMetrics, record.Category)
	}
}

func TestAppLoggerDisabledCategorySkipsPersistence(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	disabledCore := wrapCategoryCore(CategoryRules{CategoryRuntimeStats: false})(core)
	sink := newAppLoggerSinkRecorder()
	logger := NewAppLogger(zap.New(disabledCore), WithAppLogRepository(sink)).Category(CategoryRuntimeStats)

	logger.Info(context.Background(), "disabled category event", StringField("expensive", "not serialized"))
	time.Sleep(25 * time.Millisecond)
	if sink.recordCount() != 0 {
		t.Fatalf("expected disabled category to skip durable persistence, got %d records", sink.recordCount())
	}
	if len(observed.All()) != 0 {
		t.Fatalf("expected disabled category to skip zap output, got %d entries", len(observed.All()))
	}
}

func TestAppLoggerUnregisteredCategorySkipsOutputAndPersistence(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	sink := newAppLoggerSinkRecorder()
	logger := NewAppLogger(zap.New(core), WithAppLogRepository(sink)).Category(LogCategory("runtime.unknown"))

	logger.Info(context.Background(), "unknown category event")
	time.Sleep(25 * time.Millisecond)
	if sink.recordCount() != 0 {
		t.Fatalf("expected unregistered category to skip durable persistence, got %d records", sink.recordCount())
	}
	if len(observed.All()) != 0 {
		t.Fatalf("expected unregistered category to skip zap output, got %d entries", len(observed.All()))
	}
}

func TestAppLoggerDefaultCategoryIsGatedBeforeWriting(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	disabledCore := logsafe.WrapCore(wrapCategoryCore(CategoryRules{CategoryApplication: false})(core))
	sink := newAppLoggerSinkRecorder()
	logger := NewAppLogger(zap.New(disabledCore), WithAppLogRepository(sink))

	logger.Info(context.Background(), "disabled application event", StringField("expensive", "not serialized"))
	time.Sleep(25 * time.Millisecond)
	if sink.recordCount() != 0 {
		t.Fatalf("expected disabled default category to skip durable persistence, got %d records", sink.recordCount())
	}
	if len(observed.All()) != 0 {
		t.Fatalf("expected disabled default category to skip zap output, got %d entries", len(observed.All()))
	}
}

func TestAppLoggerPreservesZapOutputWhenPersistenceFails(t *testing.T) {
	core, observed := observer.New(zapcore.WarnLevel)
	sink := newAppLoggerSinkRecorder()
	sink.err = errors.New("db down")
	logger := NewAppLogger(zap.New(core), WithAppLogRepository(sink)).Named("core.app")

	logger.Warn(context.Background(), "startup degraded", StringField(FieldOperation, "boot"))
	_ = sink.waitRecord(t)
	deadline := time.After(time.Second)
	for len(observed.All()) < 2 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for persistence failure log, got %d entries", len(observed.All()))
		default:
			time.Sleep(time.Millisecond)
		}
	}

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

func TestAppLoggerPersistenceDoesNotBlockCaller(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	sink := newAppLoggerSinkRecorder()
	sink.block = make(chan struct{})
	dispatcher := event.NewDispatcher(zap.NewNop(), event.Options{WorkerCount: 1})
	if err := dispatcher.Register(NewAppLogEventHandler(sink)); err != nil {
		t.Fatalf("register app log event handler: %v", err)
	}
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start event dispatcher: %v", err)
	}
	blockClosed := false
	defer func() {
		if !blockClosed {
			close(sink.block)
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := dispatcher.Shutdown(shutdownCtx); err != nil {
			t.Fatalf("shutdown event dispatcher: %v", err)
		}
	}()
	logger := NewAppLogger(zap.New(core), WithAppLogEventPublisher(dispatcher)).Named("core.app")

	started := time.Now()
	logger.Info(context.Background(), "request complete", StringField(FieldOperation, "request"))

	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("expected async persistence to return quickly, took %s", elapsed)
	}
	if len(observed.All()) != 1 {
		t.Fatalf("expected zap output before durable persistence completes, got %d entries", len(observed.All()))
	}
	if got := sink.recordCount(); got != 0 {
		t.Fatalf("expected durable sink to still be blocked, got %d records", got)
	}

	close(sink.block)
	blockClosed = true
	_ = sink.waitRecord(t)
}
