// Package logsafe sanitizes log messages and fields before they reach zap sinks.
package logsafe

import (
	"fmt"
	"strings"
	"unicode"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SanitizeText removes CRLF and other control characters from log text while
// preserving the visible content as a single line.
func SanitizeText(value string) string {
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

// WrapCore applies message and field sanitization to one zap core.
func WrapCore(core zapcore.Core) zapcore.Core {
	if core == nil {
		return zapcore.NewNopCore()
	}

	return sanitizingCore{Core: core}
}

// Debug emits one sanitized debug log entry.
func Debug(logger *zap.Logger, message string, fields ...zap.Field) {
	write(logger, debugWriter, message, fields...)
}

// Info emits one sanitized info log entry.
func Info(logger *zap.Logger, message string, fields ...zap.Field) {
	write(logger, infoWriter, message, fields...)
}

// Warn emits one sanitized warning log entry.
func Warn(logger *zap.Logger, message string, fields ...zap.Field) {
	write(logger, warnWriter, message, fields...)
}

// Error emits one sanitized error log entry.
func Error(logger *zap.Logger, message string, fields ...zap.Field) {
	write(logger, errorWriter, message, fields...)
}

type logWriter func(*zap.Logger, string, ...zap.Field)

var (
	debugWriter logWriter = func(logger *zap.Logger, message string, fields ...zap.Field) { logger.Debug(message, fields...) }
	infoWriter  logWriter = func(logger *zap.Logger, message string, fields ...zap.Field) { logger.Info(message, fields...) }
	warnWriter  logWriter = func(logger *zap.Logger, message string, fields ...zap.Field) { logger.Warn(message, fields...) }
	errorWriter logWriter = func(logger *zap.Logger, message string, fields ...zap.Field) { logger.Error(message, fields...) }
)

func write(logger *zap.Logger, writer logWriter, message string, fields ...zap.Field) {
	if logger == nil {
		logger = zap.NewNop()
	}

	writer(logger, SanitizeText(message), SanitizeFields(fields)...)
}

// SanitizeFields rewrites string-like zap fields into sanitized single-line values.
func SanitizeFields(fields []zap.Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	sanitized := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		sanitized = append(sanitized, SanitizeField(field))
	}
	return sanitized
}

// SanitizeField rewrites one zap field when its value can inject control characters.
func SanitizeField(field zap.Field) zap.Field {
	switch field.Type {
	case zapcore.StringType:
		return zap.String(field.Key, SanitizeText(field.String))
	case zapcore.ByteStringType:
		return zap.String(field.Key, SanitizeText(string(field.Interface.([]byte))))
	case zapcore.ErrorType:
		return sanitizeErrorField(field)
	case zapcore.StringerType:
		return sanitizeStringerField(field)
	case zapcore.ReflectType:
		return sanitizeReflectField(field)
	}

	return field
}

func sanitizeErrorField(field zap.Field) zap.Field {
	if err, ok := field.Interface.(error); ok && err != nil {
		return zap.String(field.Key, SanitizeText(err.Error()))
	}

	return field
}

func sanitizeStringerField(field zap.Field) zap.Field {
	if stringer, ok := field.Interface.(fmt.Stringer); ok && stringer != nil {
		return zap.String(field.Key, SanitizeText(stringer.String()))
	}

	return field
}

func sanitizeReflectField(field zap.Field) zap.Field {
	switch typed := field.Interface.(type) {
	case string:
		return zap.String(field.Key, SanitizeText(typed))
	case []byte:
		return zap.String(field.Key, SanitizeText(string(typed)))
	case error:
		if typed != nil {
			return zap.String(field.Key, SanitizeText(typed.Error()))
		}
	case fmt.Stringer:
		if typed != nil {
			return zap.String(field.Key, SanitizeText(typed.String()))
		}
	}

	return field
}

type sanitizingCore struct {
	zapcore.Core
}

func (c sanitizingCore) With(fields []zap.Field) zapcore.Core {
	return sanitizingCore{Core: c.Core.With(SanitizeFields(fields))}
}

func (c sanitizingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return checked
	}

	return checked.AddCore(entry, c)
}

func (c sanitizingCore) Write(entry zapcore.Entry, fields []zap.Field) error {
	entry.Message = SanitizeText(entry.Message)
	return c.Core.Write(entry, SanitizeFields(fields))
}
