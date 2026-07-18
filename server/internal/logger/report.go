package logger

import (
	"context"

	"graft/server/internal/apperror"
)

const reportErrorAdditionalFieldCount = 3

// ReportError 在业务语义 owner 处记录一次 cause，并返回带 reported 标记的原错误链。
// nil logger 不会伪造 reported 状态，保证后续 HTTP fallback 仍能补足原因日志。
func ReportError(ctx context.Context, appLogger AppLogger, message string, err error, fields ...Field) error {
	if err == nil || apperror.IsReported(err) {
		return err
	}
	if appLogger == nil {
		return err
	}

	logFields := make([]Field, 0, len(fields)+reportErrorAdditionalFieldCount)
	logFields = append(logFields, fields...)
	if descriptor, ok := apperror.Describe(err); ok {
		logFields = append(logFields,
			StringField("error_kind", string(descriptor.Kind)),
			StringField("error_code", descriptor.Code.String()),
		)
	}
	logFields = append(logFields, ErrorField(err))
	appLogger.Error(ctx, message, logFields...)
	return apperror.MarkReported(err)
}
