package httpx

import (
	"github.com/gin-gonic/gin"

	"graft/server/internal/i18n"
)

const localizedErrorMessageKeyContextKey = "httpx.localized_error_message_key"

// ErrorResponse 描述对外稳定的错误响应基础结构。
//
// `error` 字段暂时保留为 `message` 的兼容别名，方便现有调试脚本和早期
// 调用方在迁移到 `message_key` 前仍能读到人类可读的错误信息。
type ErrorResponse struct {
	Error      string         `json:"error"`
	Message    string         `json:"message"`
	MessageKey string         `json:"message_key"`
	Locale     string         `json:"locale"`
	Details    map[string]any `json:"details,omitempty"`
}

// AbortLocalizedError 以统一结构中止当前请求并返回本地化错误响应。
func AbortLocalizedError(ctx *gin.Context, service *i18n.Service, status int, key string, details map[string]any) {
	WriteLocalizedError(ctx, service, status, key, details)
	ctx.Abort()
}

// WriteLocalizedError 以统一结构写入本地化错误响应。
func WriteLocalizedError(ctx *gin.Context, service *i18n.Service, status int, key string, details map[string]any) {
	locale := "zh-CN"
	message := key
	if service != nil {
		locale = service.ResolveRequestLocale(ctx.Request, "")
		message = service.Message(locale, key)
	}

	ctx.Set(localizedErrorMessageKeyContextKey, key)
	ctx.JSON(status, ErrorResponse{
		Error:      message,
		Message:    message,
		MessageKey: key,
		Locale:     locale,
		Details:    details,
	})
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
