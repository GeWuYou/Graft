package httpx

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/event"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/moduleapi"
)

const httpStatusBadRequest = 400

const websocketUpgradeSucceededContextKey = "httpx.websocket_upgrade_succeeded"

// UpgradeWebSocket 执行 WebSocket 协议升级，并把成功升级的连接从普通请求性能统计中排除。
// Gorilla WebSocket 会直接劫持底层连接，Gin 的响应状态可能仍保持为 200；包装器只在握手成功后写入
// 访问日志分类标记，失败握手仍按普通 HTTP 请求处理。
func UpgradeWebSocket(ctx *gin.Context, upgrader *websocket.Upgrader, responseHeader http.Header) (*websocket.Conn, error) {
	if ctx == nil || ctx.Request == nil || ctx.Writer == nil {
		return nil, errors.New("websocket upgrade context is unavailable")
	}
	if upgrader == nil {
		return nil, errors.New("websocket upgrader is unavailable")
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, responseHeader)
	if err != nil {
		return nil, err
	}
	markWebSocketUpgrade(ctx)
	return conn, nil
}

func newAccessLogMiddleware(logger *zap.Logger, target any, activeRequests *activeRequestTracker, options AccessLogOptions) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	options = normalizeAccessLogOptions(options)
	sink := accessLogPersistSinkFromTarget(target)

	return func(ctx *gin.Context) {
		if ctx.Request != nil && !websocket.IsWebSocketUpgrade(ctx.Request) {
			requestContext, done := activeRequests.begin(ctx.Request.Context())
			ctx.Request = ctx.Request.WithContext(requestContext)
			defer done()
		}
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

		persistAccessLog(ctx, logger, sink, record, options.PersistTimeout)
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
	if options.PersistTimeout <= 0 {
		options.PersistTimeout = time.Second
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

// requestID 和 traceID 标识当前请求；startedAt 指定请求开始时间。
func buildAccessLogRecord(ctx *gin.Context, requestID string, traceID string, startedAt time.Time) CreateAccessLogInput {
	record := CreateAccessLogInput{
		RequestID:      sanitizeAccessLogStableText(requestID),
		TraceID:        sanitizeAccessLogStableText(traceID),
		Method:         sanitizeAccessLogStableText(ctx.Request.Method),
		Path:           sanitizeAccessLogPath(currentRequestPath(ctx)),
		Route:          sanitizeAccessLogRoute(currentRequestRoute(ctx)),
		ConnectionType: currentAccessLogConnectionType(ctx),
		StatusCode:     ctx.Writer.Status(),
		DurationMS:     time.Since(startedAt).Milliseconds(),
		ClientIP:       sanitizeAccessLogStableText(ctx.ClientIP()),
		UserAgent:      sanitizeAccessLogFreeText(ctx.Request.UserAgent()),
		RequestSize:    currentRequestSize(ctx),
		ResponseSize:   currentResponseSize(ctx),
		StartedAt:      startedAt.UTC(),
		OccurredAt:     time.Now().UTC(),
	}

	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx.Request.Context()); ok && requestAuth.User != nil {
		record.UserID = cloneUint64Pointer(&requestAuth.User.ID)
		record.Username = sanitizeAccessLogStableText(requestAuth.User.Username)
	}

	return record
}

// currentAccessLogConnectionType 根据实际成功的协议升级结果区分 HTTP 请求和 WebSocket 连接。
func currentAccessLogConnectionType(ctx *gin.Context) AccessLogConnectionType {
	if ctx == nil || ctx.Request == nil || ctx.Writer == nil {
		return AccessLogConnectionTypeHTTP
	}
	if websocket.IsWebSocketUpgrade(ctx.Request) && websocketUpgradeSucceeded(ctx) {
		return AccessLogConnectionTypeWebSocket
	}
	return AccessLogConnectionTypeHTTP
}

func websocketUpgradeSucceeded(ctx *gin.Context) bool {
	if ctx == nil {
		return false
	}
	value, exists := ctx.Get(websocketUpgradeSucceededContextKey)
	succeeded, ok := value.(bool)
	return exists && ok && succeeded
}

func markWebSocketUpgrade(ctx *gin.Context) {
	if ctx != nil {
		ctx.Set(websocketUpgradeSucceededContextKey, true)
	}
}

// persistAccessLog 将访问日志记录持久化到仓储；写入受独立 deadline 约束，失败不会改变原始 HTTP 响应。
func persistAccessLog(ctx *gin.Context, logger *zap.Logger, sink AccessLogPersistSink, record CreateAccessLogInput, timeout time.Duration) {
	if sink == nil {
		return
	}

	startedAt := time.Now()
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx.Request.Context()), timeout)
	defer cancel()

	if err := sink.PersistAccessLog(persistCtx, record); err != nil {
		failureType := "error"
		if errors.Is(err, context.DeadlineExceeded) {
			failureType = "timeout"
		}
		logsafe.Error(logger, "persist access log failed",
			zap.String("requestId", record.RequestID),
			zap.String("method", record.Method),
			zap.String("path", record.Path),
			zap.Int("statusCode", record.StatusCode),
			zap.String("failureType", failureType),
			zap.Duration("persistDuration", time.Since(startedAt)),
			zap.Duration("persistTimeout", timeout),
			zap.Error(err),
		)
	}
}

// AccessLogPersistSink 是 HTTP runtime 提交规范访问日志事实的窄边界。
// 实现可以把事实交给 Runtime event publisher；middleware 不拥有队列、worker 或重试生命周期。
type AccessLogPersistSink interface {
	PersistAccessLog(context.Context, CreateAccessLogInput) error
}

func accessLogPersistSinkFromTarget(target any) AccessLogPersistSink {
	if sink, ok := target.(AccessLogPersistSink); ok {
		return sink
	}
	if repo, ok := target.(AccessLogRepository); ok && repo != nil {
		// COMPAT(owner=server/internal/app, cleanup=所有 Runtime 调用方改为注入 NewAccessLogEventPersistSink)
		// 兼容直接传 AccessLogRepository 的旧构造路径；NewServer/NewServerWithOptions 的既有调用方仍依赖同步写入。
		// Runtime 正式装配优先注入事件 sink。覆盖事件 sink 与旧 repository 直传的测试共同保护该过渡边界。
		return accessLogRepositoryPersistSink{repo: repo}
	}
	return nil
}

type accessLogRepositoryPersistSink struct {
	repo AccessLogRepository
}

func (s accessLogRepositoryPersistSink) PersistAccessLog(ctx context.Context, record CreateAccessLogInput) error {
	if s.repo == nil {
		return nil
	}
	_, err := s.repo.CreateAccessLog(ctx, record)
	return err
}

// NewAccessLogEventPersistSink 创建基于 Runtime event publisher 的访问日志 sink。
// sink 只负责 best-effort 入队；持久化、重试和 shutdown drain 由 Runtime event dispatcher 管理。
func NewAccessLogEventPersistSink(publisher event.Publisher, fallback ...AccessLogRepository) AccessLogPersistSink {
	if publisher == nil {
		return nil
	}
	var repo AccessLogRepository
	if len(fallback) > 0 {
		repo = fallback[0]
	}
	return accessLogEventPersistSink{publisher: publisher, fallback: repo}
}

func logAccess(logger *zap.Logger, status int, fields ...zap.Field) {
	if status >= 0 {
		logsafe.Info(logger, "http access", fields...)
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
