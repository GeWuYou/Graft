package logger

import (
	"context"
	"strings"
	"unicode"

	"go.uber.org/zap"

	"graft/server/internal/httpx"
)

const (
	// appLogCorrelationFieldCount is the current fixed correlation field fan-out.
	appLogCorrelationFieldCount = 6

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

// AppLogger defines the canonical application-log contract for runtime and plugins.
type AppLogger interface {
	Debug(context.Context, string, ...Field)
	Info(context.Context, string, ...Field)
	Warn(context.Context, string, ...Field)
	Error(context.Context, string, ...Field)
	Named(string) AppLogger
	With(...Field) AppLogger
	Zap() *zap.Logger
}

// Field is the logger-owned structured field contract for application logs.
type Field struct {
	Key   string
	Value any
}

type appLogger struct {
	base *zap.Logger
}

// NewAppLogger wraps the runtime zap logger with the canonical AppLogger contract.
func NewAppLogger(base *zap.Logger) AppLogger {
	if base == nil {
		base = zap.NewNop()
	}

	return appLogger{base: base}
}

func (l appLogger) Debug(ctx context.Context, message string, fields ...Field) {
	l.base.Debug(sanitizeMessage(message), l.zapFields(ctx, fields...)...)
}

func (l appLogger) Info(ctx context.Context, message string, fields ...Field) {
	l.base.Info(sanitizeMessage(message), l.zapFields(ctx, fields...)...)
}

func (l appLogger) Warn(ctx context.Context, message string, fields ...Field) {
	l.base.Warn(sanitizeMessage(message), l.zapFields(ctx, fields...)...)
}

func (l appLogger) Error(ctx context.Context, message string, fields ...Field) {
	l.base.Error(sanitizeMessage(message), l.zapFields(ctx, fields...)...)
}

func (l appLogger) Named(component string) AppLogger {
	component = sanitizeComponent(component)
	if component == "" {
		return l
	}

	return appLogger{
		base: l.base.Named(component).With(zap.String(FieldComponent, component)),
	}
}

func (l appLogger) With(fields ...Field) AppLogger {
	return appLogger{base: l.base.With(l.zapFields(context.Background(), fields...)...)}
}

func (l appLogger) Zap() *zap.Logger {
	return l.base
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
	fields = appendStringField(fields, FieldClientIP, correlation.ClientIP)
	fields = appendStringField(fields, FieldUserAgent, correlation.UserAgent)
	return fields
}

func appendStringField(fields []zap.Field, key string, value string) []zap.Field {
	if value = sanitizeString(value); value != "" {
		fields = append(fields, zap.String(key, value))
	}
	return fields
}

// StringField adds one canonical string application-log field.
func StringField(key string, value string) Field {
	return Field{Key: key, Value: value}
}

// IntField adds one canonical int application-log field.
func IntField(key string, value int) Field {
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
