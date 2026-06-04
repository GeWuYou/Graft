package httpx

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
)

const (
	httpStatusBadRequest          = 400
	httpStatusInternalServerError = 500
	accessLogPersistTimeout       = 500 * time.Millisecond
)

func newAccessLogMiddleware(logger *zap.Logger, repo AccessLogRepository, options AccessLogOptions) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	options = normalizeAccessLogOptions(options)

	return func(ctx *gin.Context) {
		startedAt := time.Now()
		requestID := EnsureRequestID(ctx)
		traceID := EnsureTraceID(ctx)

		ctx.Next()

		record := buildAccessLogRecord(ctx, requestID, traceID, startedAt)
		fields := []zap.Field{
			zap.String("requestId", record.RequestID),
			zap.String("traceId", record.TraceID),
			zap.String("method", record.Method),
			zap.String("path", record.Path),
			zap.String("route", record.Route),
			zap.Int("status", record.StatusCode),
			zap.Duration("latency", time.Duration(record.DurationMS)*time.Millisecond),
			zap.String("clientIp", record.ClientIP),
			zap.String("userAgent", record.UserAgent),
		}

		if record.UserID != nil {
			fields = append(fields, zap.Uint64("userId", *record.UserID))
		}
		if record.Username != "" {
			fields = append(fields, zap.String("username", record.Username))
		}
		if record.RequestSize != nil {
			fields = append(fields, zap.Int64("requestSize", *record.RequestSize))
		}
		if record.ResponseSize != nil {
			fields = append(fields, zap.Int64("responseSize", *record.ResponseSize))
		}
		fields = append(fields, zap.Time("occurredAt", record.OccurredAt))

		persistAccessLog(ctx, logger, repo, record)
		if shouldLogAccessToConsole(record, options) {
			logAccess(logger, ctx.Writer.Status(), fields...)
		}
	}
}

func normalizeAccessLogOptions(options AccessLogOptions) AccessLogOptions {
	switch options.ConsolePolicy {
	case config.AccessLogConsoleAlways, config.AccessLogConsoleNever, config.AccessLogConsoleErrorOnly:
	case config.AccessLogConsoleAuto:
		options.ConsolePolicy = config.ResolveAccessLogConsolePolicy("", config.AccessLogConsoleAuto)
	default:
		options.ConsolePolicy = config.AccessLogConsoleAlways
	}
	if options.SlowThreshold <= 0 {
		options.SlowThreshold = time.Second
	}
	return options
}

func shouldLogAccessToConsole(record CreateAccessLogInput, options AccessLogOptions) bool {
	switch options.ConsolePolicy {
	case config.AccessLogConsoleAlways:
		return true
	case config.AccessLogConsoleNever:
		return false
	case config.AccessLogConsoleErrorOnly:
		return record.StatusCode >= httpStatusBadRequest || time.Duration(record.DurationMS)*time.Millisecond >= options.SlowThreshold
	default:
		return true
	}
}

func buildAccessLogRecord(ctx *gin.Context, requestID string, traceID string, startedAt time.Time) CreateAccessLogInput {
	record := CreateAccessLogInput{
		RequestID:    strings.TrimSpace(requestID),
		TraceID:      strings.TrimSpace(traceID),
		Method:       strings.TrimSpace(ctx.Request.Method),
		Path:         sanitizeAccessLogPath(currentRequestPath(ctx)),
		Route:        sanitizeAccessLogRoute(currentRequestRoute(ctx)),
		StatusCode:   ctx.Writer.Status(),
		DurationMS:   time.Since(startedAt).Milliseconds(),
		ClientIP:     strings.TrimSpace(ctx.ClientIP()),
		UserAgent:    sanitizeAccessLogFreeText(strings.TrimSpace(ctx.Request.UserAgent())),
		RequestSize:  currentRequestSize(ctx),
		ResponseSize: currentResponseSize(ctx),
		StartedAt:    startedAt.UTC(),
		OccurredAt:   time.Now().UTC(),
	}

	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx.Request.Context()); ok && requestAuth.User != nil {
		record.UserID = cloneUint64Pointer(&requestAuth.User.ID)
		record.Username = strings.TrimSpace(requestAuth.User.Username)
	}

	return record
}

func persistAccessLog(ctx *gin.Context, logger *zap.Logger, repo AccessLogRepository, record CreateAccessLogInput) {
	if repo == nil {
		return
	}

	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx.Request.Context()), accessLogPersistTimeout)
	defer cancel()

	if _, err := repo.CreateAccessLog(persistCtx, record); err != nil {
		logger.Error("persist access log failed",
			zap.String("requestId", record.RequestID),
			zap.String("method", record.Method),
			zap.String("path", record.Path),
			zap.Int("statusCode", record.StatusCode),
			zap.Error(err),
		)
	}
}

func logAccess(logger *zap.Logger, status int, fields ...zap.Field) {
	switch {
	case status >= httpStatusInternalServerError:
		logger.Error("http access", fields...)
	case status >= httpStatusBadRequest:
		logger.Warn("http access", fields...)
	case status >= 0:
		logger.Info("http access", fields...)
	}
}

func currentRequestPath(ctx *gin.Context) string {
	if ctx == nil || ctx.Request == nil || ctx.Request.URL == nil {
		return ""
	}

	return strings.TrimSpace(ctx.Request.URL.Path)
}

func currentRequestRoute(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}

	return strings.TrimSpace(ctx.FullPath())
}

func currentRequestSize(ctx *gin.Context) *int64 {
	if ctx == nil || ctx.Request == nil {
		return nil
	}

	if ctx.Request.ContentLength < 0 {
		return nil
	}

	size := ctx.Request.ContentLength
	return &size
}

func currentResponseSize(ctx *gin.Context) *int64 {
	if ctx == nil || ctx.Writer == nil {
		return nil
	}

	size := int64(ctx.Writer.Size())
	if size < 0 {
		return nil
	}

	return &size
}
