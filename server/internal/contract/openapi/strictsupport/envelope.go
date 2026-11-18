// Package strictsupport 桥接 oapi-codegen 生成的 gin strict server 与平台统一的
// httpx 响应 envelope 及鉴权运行时；生成契约负责请求边界，响应写入仍归 httpx 所有。
package strictsupport

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
)

// ErrMissingGinContext 表示生成的 gin strict server 未传入 gin 请求上下文；该路径
// 不可达，出现即视为运行时装配缺陷，由统一内部错误出口处理。
var ErrMissingGinContext = errors.New("strict handler context is missing gin request context")

// Dispatch 把生成的 strict handler 调用转发到既有 gin handler 主体。
// 它保证 gin 上下文存在后执行 handler；响应写入仍由 handler 内的 httpx 完成。
func Dispatch(ctx context.Context, handler func(*gin.Context)) error {
	ginCtx := GinContext(ctx)
	if ginCtx == nil {
		return ErrMissingGinContext
	}
	handler(ginCtx)
	return nil
}

// GinContext 从生成的 strict handler 调用上下文中取回 gin 请求上下文。
// 生成的 gin strict server 会把 *gin.Context 作为 handler context 传入；适配层通过
// 它保留 httpx envelope、request-id 与请求审计的现有所有权，不改变响应字节。
func GinContext(ctx context.Context) *gin.Context {
	if ctx == nil {
		return nil
	}
	ginCtx, _ := ctx.(*gin.Context)
	return ginCtx
}

// BindingErrorHandler 返回生成 wrapper 参数绑定失败时的统一 envelope 写入器；
// 生成默认响应体会绕过平台错误结构，这里强制回到 httpx 本地化错误链路。
func BindingErrorHandler(localizer *i18n.Service, messageKey string) func(*gin.Context, error, int) {
	return func(ginCtx *gin.Context, _ error, status int) {
		httpx.WriteLocalizedError(ginCtx, localizer, status, messageKey, nil)
	}
}

// RequestErrorHandler 返回 strict body 绑定失败时的统一 400 envelope 写入器；
// data 保留模块既有的错误结构字段（如 field 标注），保证响应与迁移前一致。
func RequestErrorHandler(localizer *i18n.Service, messageKey string, data any) func(*gin.Context, error) {
	return func(ginCtx *gin.Context, _ error) {
		httpx.WriteLocalizedError(ginCtx, localizer, http.StatusBadRequest, messageKey, data)
	}
}

// HandlerErrorHandler 返回 strict handler 返回错误时的统一 envelope 写入器；
// typed application error 仍由 httpx.AbortAppError 映射，内部错误日志归属不变。
func HandlerErrorHandler(localizer *i18n.Service, runtimeLogger *zap.Logger) func(*gin.Context, error) {
	return func(ginCtx *gin.Context, err error) {
		httpx.AbortAppError(ginCtx, localizer, runtimeLogger, err)
	}
}
