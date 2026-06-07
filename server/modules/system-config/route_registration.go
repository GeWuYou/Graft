package systemconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	systemconfigopenapi "graft/server/internal/contract/openapi/systemconfig"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	systemconfigcontract "graft/server/modules/system-config/contract"
)

type routeRuntime struct {
	ctx     *module.Context
	service *Service
}

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
	group := ctx.Router.Group(systemconfigcontract.SystemConfigGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(
		systemconfigcontract.SystemConfigCollectionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, systemconfigcontract.SystemConfigReadPermission.String(), publisher),
		routes.handleList,
	)
	group.GET(
		systemconfigcontract.SystemConfigDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, systemconfigcontract.SystemConfigReadPermission.String(), publisher),
		routes.handleGet,
	)
	group.PUT(
		systemconfigcontract.SystemConfigDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, systemconfigcontract.SystemConfigWritePermission.String(), publisher),
		routes.handleUpdate,
	)
	group.POST(
		systemconfigcontract.SystemConfigResetRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, systemconfigcontract.SystemConfigWritePermission.String(), publisher),
		routes.handleReset,
	)
	return nil
}

func (r routeRuntime) handleList(ginCtx *gin.Context) {
	systemConfigGeneratedHandler{}.GetSystemConfigs(bindListParams(ginCtx))
	items, err := r.service.List(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toListResponse(items))
}

func (r routeRuntime) handleGet(ginCtx *gin.Context) {
	key := cleanKey(ginCtx.Param("key"))
	systemConfigGeneratedHandler{}.GetSystemConfig(key, bindDetailParams(ginCtx))
	item, err := r.service.Get(ginCtx.Request.Context(), key)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toItem(item))
}

func (r routeRuntime) handleUpdate(ginCtx *gin.Context) {
	key := cleanKey(ginCtx.Param("key"))
	var request systemconfigopenapi.PutSystemConfigJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, systemconfigcontract.SystemConfigInvalidRequest.String(), map[string]any{
			"field": "body",
		})
		return
	}
	systemConfigGeneratedHandler{}.PutSystemConfig(key, bindUpdateParams(ginCtx), request)
	value, err := json.Marshal(request.Value)
	if err != nil {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, systemconfigcontract.SystemConfigInvalidRequest.String(), map[string]any{
			"field": "value",
		})
		return
	}
	item, err := r.service.Update(ginCtx.Request.Context(), key, value)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toItem(item))
}

func (r routeRuntime) handleReset(ginCtx *gin.Context) {
	key := cleanKey(ginCtx.Param("key"))
	systemConfigGeneratedHandler{}.PostSystemConfigReset(key, bindResetParams(ginCtx))
	item, err := r.service.Reset(ginCtx.Request.Context(), key)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toItem(item))
}

func (r routeRuntime) writeRouteError(ginCtx *gin.Context, err error) {
	switch {
	case errors.Is(err, errDefinitionNotFound):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, systemconfigcontract.SystemConfigNotFound.String(), nil)
	case errors.Is(err, errInvalidConfigValue):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, systemconfigcontract.SystemConfigInvalidRequest.String(), nil)
	default:
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
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

type systemConfigGeneratedHandler struct{}

func (systemConfigGeneratedHandler) GetSystemConfigs(systemconfigopenapi.GetSystemConfigsParams) {}

func (systemConfigGeneratedHandler) GetSystemConfig(string, systemconfigopenapi.GetSystemConfigParams) {
}

func (systemConfigGeneratedHandler) PutSystemConfig(string, systemconfigopenapi.PutSystemConfigParams, systemconfigopenapi.PutSystemConfigJSONRequestBody) {
}

func (systemConfigGeneratedHandler) PostSystemConfigReset(string, systemconfigopenapi.PostSystemConfigResetParams) {
}

func bindListParams(ginCtx *gin.Context) systemconfigopenapi.GetSystemConfigsParams {
	locale, requestID := commonHeaders(ginCtx)
	return systemconfigopenapi.GetSystemConfigsParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindDetailParams(ginCtx *gin.Context) systemconfigopenapi.GetSystemConfigParams {
	locale, requestID := commonHeaders(ginCtx)
	return systemconfigopenapi.GetSystemConfigParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindUpdateParams(ginCtx *gin.Context) systemconfigopenapi.PutSystemConfigParams {
	locale, requestID := commonHeaders(ginCtx)
	return systemconfigopenapi.PutSystemConfigParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindResetParams(ginCtx *gin.Context) systemconfigopenapi.PostSystemConfigResetParams {
	locale, requestID := commonHeaders(ginCtx)
	return systemconfigopenapi.PostSystemConfigResetParams{XGraftLocale: locale, XRequestId: requestID}
}

func commonHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := ginCtx.GetHeader(string(httpheader.Locale))
	requestID := httpx.EnsureRequestID(ginCtx)
	return &locale, &requestID
}
