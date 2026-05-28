package httpx

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func newAccessLogMiddleware(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(ctx *gin.Context) {
		startedAt := time.Now()
		requestID := EnsureRequestID(ctx)

		ctx.Next()

		fields := []zap.Field{
			zap.String("requestId", requestID),
			zap.String("traceId", requestID),
			zap.String("method", strings.TrimSpace(ctx.Request.Method)),
			zap.String("path", currentRequestPath(ctx)),
			zap.String("route", currentRequestRoute(ctx)),
			zap.Int("status", ctx.Writer.Status()),
			zap.Duration("latency", time.Since(startedAt)),
			zap.String("clientIp", strings.TrimSpace(ctx.ClientIP())),
			zap.String("userAgent", strings.TrimSpace(ctx.Request.UserAgent())),
		}

		logAccess(logger, ctx.Writer.Status(), fields...)
	}
}

func logAccess(logger *zap.Logger, status int, fields ...zap.Field) {
	switch {
	case status >= 500:
		logger.Error("http access", fields...)
	case status >= 400:
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
