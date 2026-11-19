package systemconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/openapi/strictsupport"
	systemconfigopenapi "graft/server/internal/contract/openapi/systemconfig"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	systemconfigcontract "graft/server/modules/system-config/contract"
	systemconfigstore "graft/server/modules/system-config/store"
)

type routeRuntime struct {
	ctx     *module.Context
	service *Service
}

// systemConfigStrictServer 将系统配置服务适配到生成的 strict server 契约上。
// 路由与参数边界由生成 wrapper 编译期约束；响应仍通过 httpx envelope 写入，
// 保证迁移前后字节一致。
type systemConfigStrictServer struct {
	routes routeRuntime
}

// errStrictMissingGinContext 表示生成的 gin strict server 未传入 gin 请求上下文；
// 该路径不可达，出现即视为运行时装配缺陷。
var errStrictMissingGinContext = errors.New("strict handler context is missing gin request context")

// 编译期断言：适配层必须实现生成契约的全部 strict handler 方法；
// OpenAPI 新增或重命名操作时会在编译期暴露缺失。
var _ systemconfigopenapi.StrictServerInterface = systemConfigStrictServer{}

func registerSystemConfigRoutes(ctx *module.Context, moduleName string, service *Service) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	if service == nil {
		return errors.New("system config service is unavailable")
	}

	authService, err := resolveAuthService(ctx)
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return fmt.Errorf("resolve authorizer: %w", err)
	}

	routes := routeRuntime{ctx: ctx, service: service}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleName)
	// 生成的 wrapper 负责路径/请求头参数绑定与 strict body 绑定；绑定失败的响应
	// 仍走统一 envelope，错误消息 key 沿用模块既有契约。
	wrapper := systemconfigopenapi.ServerInterfaceWrapper{
		Handler: systemconfigopenapi.NewStrictHandlerWithOptions(
			systemConfigStrictServer{routes: routes},
			nil,
			systemconfigopenapi.StrictGinServerOptions{
				RequestErrorHandlerFunc: strictsupport.RequestErrorHandler(
					ctx.I18n,
					systemconfigcontract.SystemConfigInvalidRequest.String(),
					gin.H{"field": "body"},
				),
				HandlerErrorFunc: strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
			},
		),
		ErrorHandler: strictsupport.BindingErrorHandler(
			ctx.I18n,
			systemconfigcontract.SystemConfigInvalidRequest.String(),
		),
	}

	group := ctx.Router.Group(systemconfigcontract.SystemConfigGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(
		systemconfigcontract.SystemConfigCollectionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, systemconfigcontract.SystemConfigReadPermission.String(), publisher),
		strictsupport.Route(wrapper.GetSystemConfigs, nil),
	)
	group.GET(
		systemconfigcontract.SystemConfigDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, systemconfigcontract.SystemConfigReadPermission.String(), publisher),
		strictsupport.Route(wrapper.GetSystemConfig, nil),
	)
	group.PUT(
		systemconfigcontract.SystemConfigDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, systemconfigcontract.SystemConfigWritePermission.String(), publisher),
		strictsupport.Route(wrapper.PutSystemConfig, nil),
	)
	group.POST(
		systemconfigcontract.SystemConfigResetRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, systemconfigcontract.SystemConfigWritePermission.String(), publisher),
		strictsupport.Route(wrapper.PostSystemConfigReset, nil),
	)
	return nil
}

// GetSystemConfigs 返回配置集合；成功与错误响应都保持既有 envelope 字节。
func (a systemConfigStrictServer) GetSystemConfigs(
	ctx context.Context,
	_ systemconfigopenapi.GetSystemConfigsRequestObject,
) (systemconfigopenapi.GetSystemConfigsResponseObject, error) {
	ginCtx := strictsupport.GinContext(ctx)
	if ginCtx == nil {
		return nil, errStrictMissingGinContext
	}
	items, err := a.routes.service.List(ginCtx.Request.Context())
	if err != nil {
		a.routes.writeRouteError(ginCtx, err)
		return nil, nil
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toListResponse(items))
	return nil, nil
}

// GetSystemConfig 读取单个配置；路径参数来自生成绑定的强类型请求对象。
func (a systemConfigStrictServer) GetSystemConfig(
	ctx context.Context,
	request systemconfigopenapi.GetSystemConfigRequestObject,
) (systemconfigopenapi.GetSystemConfigResponseObject, error) {
	ginCtx := strictsupport.GinContext(ctx)
	if ginCtx == nil {
		return nil, errStrictMissingGinContext
	}
	item, err := a.routes.service.Get(ginCtx.Request.Context(), cleanKey(request.Key))
	if err != nil {
		a.routes.writeRouteError(ginCtx, err)
		return nil, nil
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toItem(item))
	return nil, nil
}

// PutSystemConfig 更新单个配置覆盖；请求体来自生成绑定的强类型请求对象。
func (a systemConfigStrictServer) PutSystemConfig(
	ctx context.Context,
	request systemconfigopenapi.PutSystemConfigRequestObject,
) (systemconfigopenapi.PutSystemConfigResponseObject, error) {
	ginCtx := strictsupport.GinContext(ctx)
	if ginCtx == nil {
		return nil, errStrictMissingGinContext
	}
	if request.Body == nil {
		httpx.WriteLocalizedError(
			ginCtx,
			a.routes.ctx.I18n,
			http.StatusBadRequest,
			systemconfigcontract.SystemConfigInvalidRequest.String(),
			gin.H{"field": "body"},
		)
		return nil, nil
	}
	value, err := json.Marshal(request.Body.Value)
	if err != nil {
		httpx.WriteLocalizedError(
			ginCtx,
			a.routes.ctx.I18n,
			http.StatusBadRequest,
			systemconfigcontract.SystemConfigInvalidRequest.String(),
			gin.H{"field": "value"},
		)
		return nil, nil
	}
	item, err := a.routes.service.Update(ginCtx.Request.Context(), cleanKey(request.Key), value, currentUserID(ginCtx))
	if err != nil {
		a.routes.writeRouteError(ginCtx, err)
		return nil, nil
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toItem(item))
	return nil, nil
}

// PostSystemConfigReset 恢复单个配置默认值；路径参数来自生成绑定。
func (a systemConfigStrictServer) PostSystemConfigReset(
	ctx context.Context,
	request systemconfigopenapi.PostSystemConfigResetRequestObject,
) (systemconfigopenapi.PostSystemConfigResetResponseObject, error) {
	ginCtx := strictsupport.GinContext(ctx)
	if ginCtx == nil {
		return nil, errStrictMissingGinContext
	}
	item, err := a.routes.service.Reset(ginCtx.Request.Context(), cleanKey(request.Key))
	if err != nil {
		a.routes.writeRouteError(ginCtx, err)
		return nil, nil
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toItem(item))
	return nil, nil
}

func currentUserID(ginCtx *gin.Context) *uint64 {
	if ginCtx == nil || ginCtx.Request == nil {
		return nil
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || auth.User == nil {
		return nil
	}
	userID := auth.User.ID
	return &userID
}

func (r routeRuntime) writeRouteError(ginCtx *gin.Context, err error) {
	switch {
	case errors.Is(err, errDefinitionNotFound):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, systemconfigcontract.SystemConfigNotFound.String(), nil)
	case errors.Is(err, errInvalidConfigValue):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, systemconfigcontract.SystemConfigInvalidRequest.String(), nil)
	case errors.Is(err, systemconfigstore.ErrVersionConflict):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusConflict, systemconfigcontract.SystemConfigVersionConflict.String(), nil)
	default:
		httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, err)
	}
}

func resolveAuthService(ctx *module.Context) (moduleapi.AuthService, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return nil, err
	}
	service, ok := resolved.(moduleapi.AuthService)
	if !ok {
		return nil, fmt.Errorf("resolved auth service has unexpected type %T", resolved)
	}
	return service, nil
}

func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return nil, err
	}
	authorizer, ok := resolved.(moduleapi.Authorizer)
	if !ok {
		return nil, fmt.Errorf("resolved authorizer has unexpected type %T", resolved)
	}
	return authorizer, nil
}
