package logger

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"

	"graft/server/internal/httpx"
)

const (
	// appLogCorrelationFieldCount is the current fixed correlation field fan-out.
	appLogCorrelationFieldCount = 4

	// FieldApp stores the runtime app name attached by the base zap logger.
	FieldApp = "app"
	// FieldEnv stores the runtime environment attached by the base zap logger.
	FieldEnv = "env"
	// FieldComponent stores the explicit logger component path.
	FieldComponent = "component"
	// FieldOperation stores the stable operation name for one action.
	FieldOperation = "operation"
	// FieldRequestID stores the canonical request correlation id.
	FieldRequestID = "request_id"
	// FieldTraceID stores the canonical trace correlation id.
	FieldTraceID = "trace_id"
	// FieldRoute stores the resolved route template.
	FieldRoute = "route"
	// FieldMethod stores the request method.
	FieldMethod = "method"
	// FieldClientIP stores the resolved client IP.
	FieldClientIP = "client_ip"
	// FieldUserAgent stores the caller user-agent string.
	FieldUserAgent = "user_agent"
	// FieldError stores the canonical error text field.
	FieldError = "error"
)

const redactedValue = "[REDACTED]"
const (
	appLogPersistQueueSize = 1024
	appLogPersistTimeout   = 2 * time.Second
)

// AppLogger defines the canonical application-log contract for runtime and modules.
type AppLogger interface {
	Debug(context.Context, string, ...Field)
	Info(context.Context, string, ...Field)
	Warn(context.Context, string, ...Field)
	Error(context.Context, string, ...Field)
	Named(string) AppLogger
	With(...Field) AppLogger
	Zap() *zap.Logger
}

// AppLoggerOption customizes AppLogger runtime behavior.
type AppLoggerOption func(*appLogger)

// Field is the logger-owned structured field contract for application logs.
type Field struct {
	Key   string
	Value any
}

type appLogger struct {
	base   *zap.Logger
	sink   appLogPersistSink
	now    func() time.Time
	fields []Field
}

type appLogPersistSink interface {
	CreateAppLog(context.Context, CreateAppLogInput) (AppLogRecord, error)
}

type asyncAppLogPersistSink struct {
	repo  AppLogRepository
	base  *zap.Logger
	queue chan appLogPersistRequest
}

type appLogPersistRequest struct {
	ctx    context.Context
	record CreateAppLogInput
}

type appLogRecordSetter func(*CreateAppLogInput, string)

var appLogTopLevelRecordSetters = map[string]appLogRecordSetter{
	FieldComponent: func(record *CreateAppLogInput, value string) { record.Component = value },
	FieldOperation: func(record *CreateAppLogInput, value string) { record.Operation = value },
	FieldRequestID: func(record *CreateAppLogInput, value string) { record.RequestID = value },
	FieldTraceID:   func(record *CreateAppLogInput, value string) { record.TraceID = value },
	FieldRoute:     func(record *CreateAppLogInput, value string) { record.Route = value },
	FieldMethod:    func(record *CreateAppLogInput, value string) { record.Method = value },
	FieldError:     func(record *CreateAppLogInput, value string) { record.Error = value },
}

// NewAppLogger wraps the runtime zap logger with the canonical AppLogger contract.
func NewAppLogger(base *zap.Logger, options ...AppLoggerOption) AppLogger {
	if base == nil {
		base = zap.NewNop()
	}

	logger := appLogger{
		base: base,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	for _, option := range options {
		if option != nil {
			option(&logger)
		}
	}

	return logger
}

// WithAppLogRepository configures a best-effort durable App Log sink.
func WithAppLogRepository(repo AppLogRepository) AppLoggerOption {
	return func(l *appLogger) {
		l.sink = newAsyncAppLogPersistSink(l.base, repo)
	}
}

func newAsyncAppLogPersistSink(base *zap.Logger, repo AppLogRepository) appLogPersistSink {
	if repo == nil {
		return nil
	}
	if base == nil {
		base = zap.NewNop()
	}

	sink := &asyncAppLogPersistSink{
		repo:  repo,
		base:  base,
		queue: make(chan appLogPersistRequest, appLogPersistQueueSize),
	}
	go sink.run()
	return sink
}

func (l appLogger) Debug(ctx context.Context, message string, fields ...Field) {
	l.write(ctx, AppLogSeverityDebug, message, fields...)
}

func (l appLogger) Info(ctx context.Context, message string, fields ...Field) {
	l.write(ctx, AppLogSeverityInfo, message, fields...)
}

func (l appLogger) Warn(ctx context.Context, message string, fields ...Field) {
	l.write(ctx, AppLogSeverityWarn, message, fields...)
}

func (l appLogger) Error(ctx context.Context, message string, fields ...Field) {
	l.write(ctx, AppLogSeverityError, message, fields...)
}

func (l appLogger) Named(component string) AppLogger {
	component = sanitizeComponent(component)
	if component == "" {
		return l
	}

	return appLogger{
		base:   l.base.Named(component).With(zap.String(FieldComponent, component)),
		sink:   l.sink,
		now:    l.now,
		fields: appendAppLoggerField(l.fields, StringField(FieldComponent, component)),
	}
}

func (l appLogger) With(fields ...Field) AppLogger {
	return appLogger{
		base:   l.base.With(l.zapFields(context.Background(), fields...)...),
		sink:   l.sink,
		now:    l.now,
		fields: appendAppLoggerFields(l.fields, fields...),
	}
}

func (l appLogger) Zap() *zap.Logger {
	return l.base
}

func (l appLogger) write(ctx context.Context, severity AppLogSeverity, message string, fields ...Field) {
	sanitizedMessage := sanitizeMessage(message)
	zapFields := l.zapFields(ctx, fields...)
	switch severity {
	case AppLogSeverityDebug:
		l.base.Debug(sanitizedMessage, zapFields...)
	case AppLogSeverityInfo:
		l.base.Info(sanitizedMessage, zapFields...)
	case AppLogSeverityWarn:
		l.base.Warn(sanitizedMessage, zapFields...)
	case AppLogSeverityError:
		l.base.Error(sanitizedMessage, zapFields...)
	}

	l.persist(ctx, severity, sanitizedMessage, fields...)
}

func (l appLogger) persist(ctx context.Context, severity AppLogSeverity, message string, fields ...Field) {
	if l.sink == nil || message == "" {
		return
	}

	record, err := l.appLogRecord(ctx, severity, message, fields...)
	if err != nil {
		l.base.Warn("app log persistence skipped", zap.Error(err))
		return
	}

	persistCtx := context.Background()
	if ctx != nil {
		persistCtx = context.WithoutCancel(ctx)
	}
	if _, err := l.sink.CreateAppLog(persistCtx, record); err != nil {
		l.base.Warn("app log persistence failed", zap.Error(err))
	}
}

func (s *asyncAppLogPersistSink) CreateAppLog(ctx context.Context, record CreateAppLogInput) (AppLogRecord, error) {
	if s == nil || s.repo == nil {
		return AppLogRecord{}, nil
	}

	select {
	case s.queue <- appLogPersistRequest{ctx: ctx, record: record}:
	default:
	}
	return AppLogRecord{}, nil
}

func (s *asyncAppLogPersistSink) run() {
	for request := range s.queue {
		ctx := request.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		persistCtx, cancel := context.WithTimeout(ctx, appLogPersistTimeout)
		if _, err := s.repo.CreateAppLog(persistCtx, request.record); err != nil {
			s.base.Warn("app log persistence failed", zap.Error(err))
		}
		cancel()
	}
}

func (l appLogger) appLogRecord(ctx context.Context, severity AppLogSeverity, message string, fields ...Field) (CreateAppLogInput, error) {
	record := CreateAppLogInput{
		OccurredAt: l.now().UTC(),
		Severity:   severity,
		Message:    message,
		Fields:     make(map[string]string),
	}
	if correlation, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		record.RequestID = correlation.RequestID
		record.TraceID = correlation.TraceID
		record.Route = correlation.Route
		record.Method = correlation.Method
	}

	for _, field := range appendAppLoggerFields(l.fields, fields...) {
		key := sanitizeFieldKey(field.Key)
		if key == "" {
			continue
		}
		value := stringifyAppLogFieldValue(key, sanitizeFieldValue(key, field.Value))
		applyAppLogRecordField(&record, key, value)
	}

	if record.Component == "" {
		record.Component = componentFromZapName(l.base.Name())
	}

	return record, nil
}

func applyAppLogRecordField(record *CreateAppLogInput, key string, value string) {
	if record == nil {
		return
	}

	if setter, ok := appLogTopLevelRecordSetters[key]; ok {
		setter(record, value)
		return
	}
	if isAppLogTopLevelField(key) || IsForbiddenAppLogPersistedField(key) {
		return
	}
	record.Fields[key] = value
}

func appendAppLoggerFields(existing []Field, fields ...Field) []Field {
	if len(existing) == 0 && len(fields) == 0 {
		return nil
	}
	combined := make([]Field, 0, len(existing)+len(fields))
	combined = append(combined, existing...)
	combined = append(combined, fields...)
	return combined
}

func appendAppLoggerField(existing []Field, field Field) []Field {
	return appendAppLoggerFields(existing, field)
}

func (l appLogger) zapFields(ctx context.Context, fields ...Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields)+appLogCorrelationFieldCount)
	if correlation, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		zapFields = appendCorrelationFields(zapFields, correlation)
	}

	for _, field := range fields {
		key := sanitizeFieldKey(field.Key)
		if key == "" {
			continue
		}
		zapFields = append(zapFields, zap.Any(key, sanitizeFieldValue(key, field.Value)))
	}

	return zapFields
}

func appendCorrelationFields(fields []zap.Field, correlation httpx.RequestAuditContext) []zap.Field {
	fields = appendStringField(fields, FieldRequestID, correlation.RequestID)
	fields = appendStringField(fields, FieldTraceID, correlation.TraceID)
	fields = appendStringField(fields, FieldRoute, correlation.Route)
	fields = appendStringField(fields, FieldMethod, correlation.Method)
	return fields
}

func appendStringField(fields []zap.Field, key string, value string) []zap.Field {
	if value = sanitizeString(value); value != "" {
		fields = append(fields, zap.String(key, value))
	}
	return fields
}

func stringifyAppLogFieldValue(_ string, value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return sanitizeString(text)
	}
	if err, ok := value.(error); ok {
		return sanitizeString(err.Error())
	}
	return sanitizeString(fmt.Sprint(value))
}

func componentFromZapName(name string) string {
	return sanitizeComponent(strings.TrimSpace(name))
}

// StringField adds one canonical string application-log field.
func StringField(key string, value string) Field {
	return Field{Key: key, Value: value}
}

// IntField adds one canonical int application-log field.
func IntField(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64Field adds one canonical int64 application-log field.
func Int64Field(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Uint64Field adds one canonical uint64 application-log field.
func Uint64Field(key string, value uint64) Field {
	return Field{Key: key, Value: value}
}

// BoolField adds one canonical bool application-log field.
func BoolField(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// DurationField adds one canonical duration field via zap-compatible value handling.
func DurationField(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// TimeField adds one canonical timestamp application-log field.
func TimeField(key string, value time.Time) Field {
	return Field{Key: key, Value: value.UTC().Format(time.RFC3339)}
}

// ErrorField stores the error text under the canonical app-log error field key.
func ErrorField(err error) Field {
	if err == nil {
		return Field{}
	}

	return Field{Key: FieldError, Value: err.Error()}
}

func sanitizeFieldValue(key string, value any) any {
	if isSensitiveKey(key) {
		return redactedValue
	}

	switch typed := value.(type) {
	case string:
		return sanitizeString(typed)
	case error:
		return sanitizeString(typed.Error())
	default:
		return value
	}
}

func sanitizeFieldKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(key))
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '.':
			builder.WriteRune(r)
		case r == '-', unicode.IsSpace(r):
			builder.WriteByte('_')
		}
	}

	return strings.Trim(builder.String(), "._")
}

func sanitizeComponent(component string) string {
	return sanitizeFieldKey(component)
}

func sanitizeMessage(message string) string {
	return sanitizeString(message)
}

func sanitizeString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if r == '\n' || r == '\r' || r == '\t' {
			builder.WriteByte(' ')
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		builder.WriteRune(r)
	}

	return strings.Join(strings.Fields(builder.String()), " ")
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, candidate := range []string{"password", "secret", "token", "authorization", "cookie", "set_cookie"} {
		if strings.Contains(key, candidate) {
			return true
		}
	}
	return false
}
