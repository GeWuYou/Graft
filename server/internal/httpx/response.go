package httpx

import (
	"github.com/gin-gonic/gin"

	"graft/server/internal/i18n"
)

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

	ctx.JSON(status, ErrorResponse{
		Error:      message,
		Message:    message,
		MessageKey: key,
		Locale:     locale,
		Details:    details,
	})
}
