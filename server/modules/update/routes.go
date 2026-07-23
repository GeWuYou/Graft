package update

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	updatecontract "graft/server/modules/update/contract"
)

// registerRoutes 安装只读发现与显式检查路由；检查不触发升级副作用。
func registerRoutes(ctx *module.Context, service *Service, rollout *RolloutService) error {
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
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	handlers := updateRouteHandlers{localizer: ctx.I18n, auth: auth, rollout: rollout}
	group.GET(updatecontract.UpdateStatusRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), func(ginCtx *gin.Context) { httpx.WriteSuccess(ginCtx, http.StatusOK, service.Status()) })
	group.POST(updatecontract.UpdateCheckRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateCheckPermission.String()), func(ginCtx *gin.Context) {
		httpx.WriteSuccess(ginCtx, http.StatusOK, service.Check(ginCtx.Request.Context()))
	})
	if rollout == nil {
		return errors.New("platform-update rollout service is unavailable")
	}
	group.GET(updatecontract.UpdateOperationCollectionRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), handlers.list)
	group.GET(updatecontract.UpdateOperationRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), handlers.get)
	group.POST(updatecontract.UpdateOperationCollectionRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateManagePermission.String(), publisher), handlers.start)
	return nil
}

type updateRouteHandlers struct {
	localizer *i18n.Service
	auth      moduleapi.AuthService
	rollout   *RolloutService
}

func (h updateRouteHandlers) list(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return
		}
		limit = parsed
	}
	items, err := h.rollout.operations.List(c.Request.Context(), limit)
	if err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, items)
}

func (h updateRouteHandlers) get(c *gin.Context) {
	item, err := h.rollout.operations.Get(c.Request.Context(), c.Param("operationID"))
	if errors.Is(err, errUpdateOperationNotFound) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusNotFound, messagecontract.CommonNotFound.String(), nil)
		return
	}
	if err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, item)
}

func (h updateRouteHandlers) start(c *gin.Context) {
	var request struct {
		TargetVersion string `json:"target_version"`
		Confirmation  string `json:"confirmation"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	actor, err := h.auth.CurrentUser(c.Request.Context())
	if err != nil || actor == nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusUnauthorized, "auth.unauthenticated", nil)
		return
	}
	operation, err := h.rollout.Start(c.Request.Context(), actor.ID, request.TargetVersion, request.Confirmation)
	if err != nil {
		switch {
		case errors.Is(err, errRolloutInvalidArgument):
			httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		case errors.Is(err, errRolloutPrecondition):
			httpx.WriteLocalizedError(c, h.localizer, http.StatusPreconditionFailed, messagecontract.CommonInvalidArgument.String(), nil)
		default:
			httpx.WriteLocalizedError(c, h.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		}
		return
	}
	httpx.WriteSuccess(c, http.StatusAccepted, operation)
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
