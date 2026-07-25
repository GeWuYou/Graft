package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"graft/server/internal/event"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/requestctx"
)

const (
	// appLogCorrelationFieldCount is the current fixed correlation field fan-out.
	appLogCorrelationFieldCount = 4
	// appLoggerCallerSkip 跳过 AppLogger facade 与 write helper 栈帧。
	appLoggerCallerSkip = 2

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

const defaultAppLogCategory LogCategory = CategoryApplication

const redactedValue = "[REDACTED]"

// AppLogPersistEventType 由 logger 拥有，表示一条已净化的应用日志记录需要异步持久化。
const AppLogPersistEventType event.Type = "logger.app-log.persist.v1"

const (
	appLogPersistEventVersion = 1
	appLogPersistEventSource  = "internal.logger"
)

// AppLogger defines the canonical application-log contract for runtime and modules.
type AppLogger interface {
	Debug(context.Context, string, ...Field)
	Info(context.Context, string, ...Field)
	Warn(context.Context, string, ...Field)
	Error(context.Context, string, ...Field)
	Category(LogCategory) AppLogger
	Named(string) AppLogger
	With(...Field) AppLogger
	AddCallerSkip(int) AppLogger
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
	categoryBase *zap.Logger
	base         *zap.Logger
	sink         appLogPersistSink
	now          func() time.Time
	fields       []Field
	category     LogCategory
}

type appLogPersistSink interface {
	CreateAppLog(context.Context, CreateAppLogInput) (AppLogRecord, error)
}

type appLogRepositorySink struct{ repo AppLogRepository }

type appLogEventPublisherSink struct{ publisher event.Publisher }

type appLogPersistEventPayload struct {
	Record CreateAppLogInput `json:"record"`
}

type appLogEventHandler struct{ repo AppLogRepository }

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

	categoryBase := base.WithOptions(zap.AddCallerSkip(appLoggerCallerSkip))
	logger := appLogger{
		categoryBase: categoryBase,
		base:         categoryBase.With(zap.String(categoryFieldKey, string(defaultAppLogCategory))),
		category:     defaultAppLogCategory,
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

// WithAppLogRepository 配置直接写入 repository 的测试 seam。
//
// Runtime 应使用 WithAppLogEventPublisher，把生产写入交给统一事件管线。
func WithAppLogRepository(repo AppLogRepository) AppLoggerOption {
	return func(l *appLogger) {
		if repo != nil {
			l.sink = appLogRepositorySink{repo: repo}
		}
	}
}

// WithAppLogEventPublisher 配置由通用事件管线异步持久化 App Log 的 sink。
func WithAppLogEventPublisher(publisher event.Publisher) AppLoggerOption {
	return func(l *appLogger) {
		if publisher != nil {
			l.sink = appLogEventPublisherSink{publisher: publisher}
		}
	}
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

	categoryBase := l.categoryBase.Named(component).With(zap.String(FieldComponent, component))
	return appLogger{
		categoryBase: categoryBase,
		base:         categoryBase.With(zap.String(categoryFieldKey, string(l.category))),
		sink:         l.sink,
		now:          l.now,
		fields:       appendAppLoggerField(l.fields, StringField(FieldComponent, component)),
		category:     l.category,
	}
}

// Category 为 AppLogger 绑定一个已注册类别，同时保持现有 component 标识不变。
func (l appLogger) Category(category LogCategory) AppLogger {
	if !isRegisteredCategory(category) {
		return appLogger{categoryBase: zap.NewNop(), base: zap.NewNop(), sink: l.sink, now: l.now, fields: l.fields, category: category}
	}
	return appLogger{
		categoryBase: l.categoryBase,
		base:         l.categoryBase.With(zap.String(categoryFieldKey, string(category))),
		sink:         l.sink,
		now:          l.now,
		fields:       l.fields,
		category:     category,
	}
}

func (l appLogger) With(fields ...Field) AppLogger {
	categoryBase := l.categoryBase.With(l.zapFields(context.Background(), fields...)...)
	return appLogger{
		categoryBase: categoryBase,
		base:         categoryBase.With(zap.String(categoryFieldKey, string(l.category))),
		sink:         l.sink,
		now:          l.now,
		fields:       appendAppLoggerFields(l.fields, fields...),
		category:     l.category,
	}
}

// AddCallerSkip 返回增加调用栈补偿后的 logger，供跨越 logger facade 的 owner 保留真实调用点。
func (l appLogger) AddCallerSkip(skip int) AppLogger {
	if skip <= 0 {
		return l
	}
	return appLogger{
		categoryBase: l.categoryBase.WithOptions(zap.AddCallerSkip(skip)),
		base:         l.base.WithOptions(zap.AddCallerSkip(skip)),
		sink:         l.sink,
		now:          l.now,
		fields:       l.fields,
		category:     l.category,
	}
}

func (l appLogger) Zap() *zap.Logger {
	return l.base
}

func (l appLogger) write(ctx context.Context, severity AppLogSeverity, message string, fields ...Field) {
	if !isRegisteredCategory(l.effectiveCategory()) {
		return
	}
	if !appLogCategoryEnabled(l.base.Core(), l.effectiveCategory(), zapLevelForAppLogSeverity(severity)) {
		return
	}
	sanitizedMessage := sanitizeMessage(message)
	zapFields := l.zapFields(ctx, fields...)
	switch severity {
	case AppLogSeverityDebug:
		logsafe.Debug(l.base, sanitizedMessage, zapFields...)
	case AppLogSeverityInfo:
		logsafe.Info(l.base, sanitizedMessage, zapFields...)
	case AppLogSeverityWarn:
		logsafe.Warn(l.base, sanitizedMessage, zapFields...)
	case AppLogSeverityError:
		logsafe.Error(l.base, sanitizedMessage, zapFields...)
	}

	l.persist(ctx, severity, sanitizedMessage, fields...)
}

func appLogCategoryEnabled(core zapcore.Core, category LogCategory, level zapcore.Level) bool {
	gate, ok := core.(interface {
		CategoryGateEnabled(string, zapcore.Level) bool
	})
	return !ok || gate.CategoryGateEnabled(string(category), level)
}

func zapLevelForAppLogSeverity(severity AppLogSeverity) zapcore.Level {
	switch severity {
	case AppLogSeverityDebug:
		return zap.DebugLevel
	case AppLogSeverityWarn:
		return zap.WarnLevel
	case AppLogSeverityError:
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
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

func (s appLogRepositorySink) CreateAppLog(ctx context.Context, record CreateAppLogInput) (AppLogRecord, error) {
	if s.repo == nil {
		return AppLogRecord{}, nil
	}
	return s.repo.CreateAppLog(ctx, record)
}

func (s appLogEventPublisherSink) CreateAppLog(_ context.Context, record CreateAppLogInput) (AppLogRecord, error) {
	if s.publisher == nil {
		return AppLogRecord{}, nil
	}
	payload, err := json.Marshal(appLogPersistEventPayload{Record: record})
	if err != nil {
		return AppLogRecord{}, fmt.Errorf("encode app log persistence event: %w", err)
	}
	eventID, err := event.NewID()
	if err != nil {
		return AppLogRecord{}, fmt.Errorf("create app log persistence event id: %w", err)
	}
	_, err = s.publisher.PublishAsync(event.Event{
		ID:             eventID,
		Type:           AppLogPersistEventType,
		Version:        appLogPersistEventVersion,
		Source:         appLogPersistEventSource,
		Payload:        payload,
		OccurredAt:     record.OccurredAt,
		CorrelationID:  record.RequestID,
		IdempotencyKey: record.RequestID,
	}, event.PublishOptions{Delivery: event.DeliveryBestEffort})
	if err != nil {
		return AppLogRecord{}, fmt.Errorf("publish app log persistence event: %w", err)
	}
	return AppLogRecord{}, nil
}

// NewAppLogEventHandler 创建把 logger-owned App Log 事件写入 repository 的消费者。
func NewAppLogEventHandler(repo AppLogRepository) event.Handler {
	return appLogEventHandler{repo: repo}
}

func (appLogEventHandler) ID() string { return "logger.app-log-persist" }

func (appLogEventHandler) Types() []event.Type { return []event.Type{AppLogPersistEventType} }

func (h appLogEventHandler) Handle(ctx context.Context, current event.Event) error {
	if h.repo == nil {
		return fmt.Errorf("app log repository is unavailable")
	}
	var payload appLogPersistEventPayload
	if err := json.Unmarshal(current.Payload, &payload); err != nil {
		return fmt.Errorf("decode app log persistence event: %w", err)
	}
	if _, err := h.repo.CreateAppLog(ctx, payload.Record); err != nil {
		return fmt.Errorf("persist app log: %w", err)
	}
	return nil
}

func (l appLogger) appLogRecord(ctx context.Context, severity AppLogSeverity, message string, fields ...Field) (CreateAppLogInput, error) {
	record := CreateAppLogInput{
		OccurredAt: l.now().UTC(),
		Severity:   severity,
		Category:   l.effectiveCategory(),
		Message:    message,
		Fields:     make(map[string]string),
	}
	if correlation, ok := requestctx.AuditContextFromContext(ctx); ok {
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

func (l appLogger) effectiveCategory() LogCategory {
	return l.category
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
	if correlation, ok := requestctx.AuditContextFromContext(ctx); ok {
		zapFields = appendCorrelationFields(zapFields, correlation)
	}

	for _, field := range fields {
		key := sanitizeFieldKey(field.Key)
		if key == "" {
			continue
		}
		zapFields = append(zapFields, zap.Any(key, sanitizeFieldValue(key, field.Value)))
	}

	return logsafe.SanitizeFields(zapFields)
}

func appendCorrelationFields(fields []zap.Field, correlation requestctx.AuditContext) []zap.Field {
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
	return logsafe.SanitizeText(value)
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
