package logger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"graft/server/internal/apperror"
)

const reportErrorAdditionalFieldCount = 4

// ReportError 在业务语义 owner 处记录一次错误诊断，并返回带 reported 标记的原错误链。
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
	// 只保留稳定类型和不可逆指纹，避免把 cause 中可能携带的凭证或用户输入写入日志。
	logFields = append(logFields,
		StringField("error_type", fmt.Sprintf("%T", err)),
		StringField("error_fingerprint", fingerprintError(err)),
	)
	appLogger.AddCallerSkip(1).Error(ctx, message, logFields...)
	return apperror.MarkReported(err)
}

func fingerprintError(err error) string {
	if err == nil {
		return ""
	}
	metadata := fmt.Sprintf("%T", err)
	if descriptor, ok := apperror.Describe(err); ok {
		metadata = fmt.Sprintf("%s|%s|%s|%s", metadata, descriptor.Kind, descriptor.Code.String(), descriptor.MessageKey.String())
	}
	digest := sha256.Sum256([]byte(metadata))
	return hex.EncodeToString(digest[:])
}
