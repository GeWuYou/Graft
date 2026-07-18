package httpx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"graft/server/internal/apperror"
	"graft/server/internal/contract/errorcode"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/i18n"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/requestctx"
)

const localizedErrorMessageKeyContextKey = "httpx.localized_error_message_key"
const requestIDContextKey = "httpx.request_id"
const traceIDContextKey = "httpx.trace_id"

// RequestAuditContext 是 core 请求关联契约在 HTTP 边界保留的调用名称。
// 新增非 HTTP 代码应直接依赖 requestctx。
type RequestAuditContext = requestctx.AuditContext

// RequestIDHeader 是统一回写给客户端的稳定 request-id 响应头。
const RequestIDHeader = string(httpheader.RequestID)

const traceIDFallbackHeader = string(httpheader.TraceID)

// SuccessResponse 描述统一成功响应 envelope。
//
// 成功响应固定返回 success/code/message/traceId/data，方便前端在最小 MVP
// 阶段也能稳定依赖固定结构，而不是按接口逐个猜测顶层字段。
type SuccessResponse[T any] struct {
	Success    bool   `json:"success"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	TraceID    string `json:"traceId"`
	MessageKey string `json:"messageKey,omitempty"`
	Locale     string `json:"locale,omitempty"`
	Data       T      `json:"data"`
}

// ErrorResponse 描述统一错误响应 envelope。
//
// 错误响应固定返回 success/code/message/traceId，messageKey/locale/data 仅在
// 当前错误路径需要时补充，避免 message 与 error 双字段重复。
type ErrorResponse struct {
	Success    bool           `json:"success"`
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	TraceID    string         `json:"traceId"`
	MessageKey string         `json:"messageKey,omitempty"`
	Locale     string         `json:"locale,omitempty"`
	Data       any            `json:"data,omitempty"`
	Error      string         `json:"-"`
	Details    map[string]any `json:"-"`
}

// AbortLocalizedError 以统一结构中止当前请求并返回本地化错误响应。
func AbortLocalizedError(ctx *gin.Context, service *i18n.Service, status int, key string, data any) {
	WriteLocalizedError(ctx, service, status, key, data)
	ctx.Abort()
}

// WriteLocalizedError 以统一结构写入本地化错误响应。
func WriteLocalizedError(ctx *gin.Context, service *i18n.Service, status int, key string, data any) {
	WriteLocalizedErrorCode(ctx, service, status, errorcode.FromMessageKey(messagecontract.Key(key)).String(), key, data)
}

// WriteLocalizedErrorCode 以显式业务 code 与 message key 写入统一错误响应。
func WriteLocalizedErrorCode(ctx *gin.Context, service *i18n.Service, status int, code string, key string, data any) {
	locale := "zh-CN"
	message := key
	if service != nil {
		locale = service.ResolveRequestLocale(ctx.Request, "")
		message = service.Message(locale, key)
	}

	ctx.Set(localizedErrorMessageKeyContextKey, key)
	traceID := EnsureRequestID(ctx)
	ctx.JSON(status, ErrorResponse{
		Success:    false,
		Code:       code,
		Message:    message,
		TraceID:    traceID,
		MessageKey: key,
		Locale:     locale,
		Data:       data,
	})
}

// WriteSuccess 以统一 envelope 写入成功响应。
func WriteSuccess[T any](ctx *gin.Context, status int, data T) {
	traceID := EnsureRequestID(ctx)
	ctx.JSON(status, SuccessResponse[T]{
		Success: true,
		Code:    errorcode.OK.String(),
		Message: errorcode.OK.String(),
		TraceID: traceID,
		Data:    data,
	})
}

// RequestIDMiddleware 确保当前请求在进入业务链路前获得稳定 request-id。
func RequestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := EnsureRequestID(ctx)
		traceID := EnsureTraceID(ctx)
		if ctx.Request != nil {
			ctx.Request = ctx.Request.WithContext(WithRequestAuditContext(ctx.Request.Context(), RequestAuditContext{
				RequestID: requestID,
				TraceID:   traceID,
				Route:     currentRequestRoute(ctx),
				Method:    ctx.Request.Method,
				ClientIP:  ctx.ClientIP(),
				UserAgent: ctx.Request.UserAgent(),
			}))
		}
		ctx.Next()
	}
}

// WriteAppError 将 typed application error 映射到现有本地化响应 envelope。
// 未记录的 internal error 会在 HTTP 最后边界补一条 cause 日志，公开响应只使用安全 descriptor。
func WriteAppError(ctx *gin.Context, service *i18n.Service, runtimeLogger *zap.Logger, err error) {
	descriptor, ok := apperror.Describe(err)
	if !ok {
		descriptor = apperror.Descriptor{
			Kind:       apperror.KindInternal,
			Code:       errorcode.CommonInternalError,
			MessageKey: messagecontract.CommonInternalError,
		}
	}
	descriptor, status := normalizeAppErrorDescriptor(descriptor)
	if descriptor.Kind == apperror.KindInternal && err != nil && !apperror.IsReported(err) {
		logUnreportedInternalError(ctx, runtimeLogger, err)
	}
	WriteLocalizedErrorCode(ctx, service, status, descriptor.Code.String(), descriptor.MessageKey.String(), descriptor.PublicData)
}

// AbortAppError 写入 typed application error 响应并中止当前 Gin 调用链。
func AbortAppError(ctx *gin.Context, service *i18n.Service, runtimeLogger *zap.Logger, err error) {
	WriteAppError(ctx, service, runtimeLogger, err)
	ctx.Abort()
}

func normalizeAppErrorDescriptor(descriptor apperror.Descriptor) (apperror.Descriptor, int) {
	descriptor.Kind = normalizeAppErrorKind(descriptor.Kind)
	if descriptor.Code == "" {
		descriptor.Code = errorcode.FromMessageKey(descriptor.MessageKey)
	}
	if descriptor.MessageKey == "" || descriptor.Code == "" {
		descriptor = internalAppErrorDescriptor()
	}
	return descriptor, appErrorStatus(descriptor.Kind)
}

func normalizeAppErrorKind(kind apperror.Kind) apperror.Kind {
	switch kind {
	case apperror.KindInvalidArgument, apperror.KindUnauthenticated, apperror.KindForbidden, apperror.KindNotFound, apperror.KindConflict, apperror.KindInternal:
		return kind
	default:
		return apperror.KindInternal
	}
}

func internalAppErrorDescriptor() apperror.Descriptor {
	return apperror.Descriptor{
		Kind:       apperror.KindInternal,
		Code:       errorcode.CommonInternalError,
		MessageKey: messagecontract.CommonInternalError,
	}
}

func appErrorStatus(kind apperror.Kind) int {
	switch kind {
	case apperror.KindInvalidArgument:
		return http.StatusBadRequest
	case apperror.KindUnauthenticated:
		return http.StatusUnauthorized
	case apperror.KindForbidden:
		return http.StatusForbidden
	case apperror.KindNotFound:
		return http.StatusNotFound
	case apperror.KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func logUnreportedInternalError(ctx *gin.Context, runtimeLogger *zap.Logger, err error) {
	if runtimeLogger == nil {
		runtimeLogger = zap.NewNop()
	}
	logsafe.Error(runtimeLogger, "unreported internal error",
		zap.String("request_id", EnsureRequestID(ctx)),
		zap.String("trace_id", EnsureTraceID(ctx)),
		zap.String("method", currentRequestMethod(ctx)),
		zap.String("route", currentRequestRoute(ctx)),
		zap.String("path", currentRequestPath(ctx)),
		zap.Error(err),
	)
}

func currentRequestMethod(ctx *gin.Context) string {
	if ctx == nil || ctx.Request == nil {
		return ""
	}
	return ctx.Request.Method
}

// EnsureRequestID 读取或生成当前请求的稳定 request-id，并统一回写响应头。
func EnsureRequestID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}

	if current, ok := ctx.Get(requestIDContextKey); ok {
		if requestID, ok := current.(string); ok {
			requestID = sanitizeAccessLogStableText(requestID)
			if requestID == "" {
				goto resolveRequestID
			}
			ctx.Writer.Header().Set(RequestIDHeader, requestID)
			return requestID
		}
	}

resolveRequestID:
	requestID := sanitizeAccessLogStableText(ctx.GetHeader(RequestIDHeader))
	if requestID == "" {
		requestID = sanitizeAccessLogStableText(ctx.GetHeader(traceIDFallbackHeader))
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}

	ctx.Set(requestIDContextKey, requestID)
	ctx.Writer.Header().Set(RequestIDHeader, requestID)
	return requestID
}

// EnsureTraceID reads the incoming trace identifier when present and falls back
// to the canonical request id when no distinct trace id is available.
func EnsureTraceID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}

	if current, ok := ctx.Get(traceIDContextKey); ok {
		if traceID, ok := current.(string); ok {
			traceID = sanitizeAccessLogStableText(traceID)
			if traceID == "" {
				goto resolveTraceID
			}
			return traceID
		}
	}

resolveTraceID:
	traceID := sanitizeAccessLogStableText(ctx.GetHeader(traceIDFallbackHeader))
	if traceID == "" {
		traceID = EnsureRequestID(ctx)
	}

	ctx.Set(traceIDContextKey, traceID)
	return traceID
}

// WithRequestAuditContext attaches the canonical request audit snapshot to one
// request-scoped context.
func WithRequestAuditContext(ctx context.Context, auditCtx RequestAuditContext) context.Context {
	return requestctx.WithAuditContext(ctx, auditCtx)
}

// RequestAuditContextFromContext reads the canonical request audit snapshot from
// one context chain.
func RequestAuditContextFromContext(ctx context.Context) (RequestAuditContext, bool) {
	return requestctx.AuditContextFromContext(ctx)
}

// LastErrorMessageKey 返回当前请求最近一次统一错误响应写入的稳定 message key。
func LastErrorMessageKey(ctx *gin.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	value, ok := ctx.Get(localizedErrorMessageKeyContextKey)
	if !ok {
		return "", false
	}

	key, ok := value.(string)
	if !ok || key == "" {
		return "", false
	}

	return key, true
}

// UnmarshalJSON 为测试与调试辅助保留旧字段别名视图，但不改变对外 JSON 契约。
func (r *ErrorResponse) UnmarshalJSON(data []byte) error {
	type rawErrorResponse ErrorResponse

	var decoded rawErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*r = ErrorResponse(decoded)
	r.Error = r.Message

	if r.Data == nil {
		return nil
	}

	switch details := r.Data.(type) {
	case map[string]any:
		r.Details = details
	}

	return nil
}
