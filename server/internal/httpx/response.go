package httpx

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"graft/server/internal/i18n"
)

const localizedErrorMessageKeyContextKey = "httpx.localized_error_message_key"
const requestIDContextKey = "httpx.request_id"

// RequestIDHeader 是统一回写给客户端的稳定 request-id 响应头。
const RequestIDHeader = "X-Request-Id"

const traceIDFallbackHeader = "X-Trace-Id"

// #nosec G101 -- 这里保存的是稳定业务 code 映射，不是账号或密钥。
var messageKeyCodes = map[string]string{
	"common.invalid_argument":        "COMMON_INVALID_ARGUMENT",
	"common.internal_error":          "COMMON_INTERNAL_ERROR",
	"auth.invalid_credentials":       "AUTH_INVALID_CREDENTIALS",
	"auth.token_missing":             "AUTH_TOKEN_MISSING",
	"auth.token_expired":             "AUTH_TOKEN_EXPIRED",
	"auth.token_invalid":             "AUTH_TOKEN_INVALID",
	"auth.forbidden":                 "AUTH_FORBIDDEN",
	"auth.password_policy_violation": "AUTH_PASSWORD_POLICY_VIOLATION",
	"auth.password_reuse_forbidden":  "AUTH_PASSWORD_REUSE_FORBIDDEN",
	"auth.current_password_invalid":  "AUTH_CURRENT_PASSWORD_INVALID",
}

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
	WriteLocalizedErrorCode(ctx, service, status, codeFromMessageKey(key), key, data)
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
		Code:    "OK",
		Message: "OK",
		TraceID: traceID,
		Data:    data,
	})
}

// RequestIDMiddleware 确保当前请求在进入业务链路前获得稳定 request-id。
func RequestIDMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		EnsureRequestID(ctx)
		ctx.Next()
	}
}

// EnsureRequestID 读取或生成当前请求的稳定 request-id，并统一回写响应头。
func EnsureRequestID(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}

	if current, ok := ctx.Get(requestIDContextKey); ok {
		if requestID, ok := current.(string); ok && strings.TrimSpace(requestID) != "" {
			ctx.Writer.Header().Set(RequestIDHeader, requestID)
			return requestID
		}
	}

	requestID := strings.TrimSpace(ctx.GetHeader(RequestIDHeader))
	if requestID == "" {
		requestID = strings.TrimSpace(ctx.GetHeader(traceIDFallbackHeader))
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}

	ctx.Set(requestIDContextKey, requestID)
	ctx.Writer.Header().Set(RequestIDHeader, requestID)
	return requestID
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

func codeFromMessageKey(key string) string {
	if code, ok := messageKeyCodes[key]; ok {
		return code
	}

	replacer := strings.NewReplacer(".", "_", "-", "_")
	return strings.ToUpper(replacer.Replace(strings.TrimSpace(key)))
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
