package update

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	updatecontract "graft/server/modules/update/contract"
)

// registerRoutes 安装只读发现与显式检查路由；检查不触发升级副作用。
func registerRoutes(ctx *module.Context, service *Service) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	auth, err := resolveAuth(ctx)
	if err != nil {
		return err
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return err
	}
	group := ctx.Router.Group(updatecontract.UpdateGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(updatecontract.UpdateStatusRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), func(ginCtx *gin.Context) { httpx.WriteSuccess(ginCtx, http.StatusOK, service.Status()) })
	group.POST(updatecontract.UpdateCheckRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateCheckPermission.String()), func(ginCtx *gin.Context) {
		httpx.WriteSuccess(ginCtx, http.StatusOK, service.Check(ginCtx.Request.Context()))
	})
	return nil
}

func resolveAuth(ctx *module.Context) (moduleapi.AuthService, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("module services are unavailable")
	}
	value, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve auth service: %w", err)
	}
	service, ok := value.(moduleapi.AuthService)
	if !ok || service == nil {
		return nil, fmt.Errorf("resolved auth service has unexpected type %T", value)
	}
	return service, nil
}
func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	value, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve authorizer: %w", err)
	}
	service, ok := value.(moduleapi.Authorizer)
	if !ok || service == nil {
		return nil, fmt.Errorf("resolved authorizer has unexpected type %T", value)
	}
	return service, nil
}
