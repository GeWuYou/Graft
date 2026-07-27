package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger/logsafe"
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
	handlers := updateRouteHandlers{localizer: ctx.I18n, auth: auth, rollout: rollout, logger: ctx.Logger}
	group.GET(updatecontract.UpdateStatusRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), func(ginCtx *gin.Context) {
		httpx.WriteSuccess(ginCtx, http.StatusOK, statusForUpdateViewer(ginCtx.Request.Context(), authorizer, service.Status()))
	})
	group.POST(updatecontract.UpdateCheckRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateCheckPermission.String()), func(ginCtx *gin.Context) {
		httpx.WriteSuccess(ginCtx, http.StatusOK, statusForUpdateViewer(ginCtx.Request.Context(), authorizer, service.Check(ginCtx.Request.Context())))
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
	logger    *zap.Logger
}

// mayViewComposeCandidates 将宿主机路径限定在已通过更新管理权限校验的请求中。
// 读取状态失败时按拒绝处理，避免 read 权限意外获得升级配置细节。
func mayViewComposeCandidates(ctx context.Context, authorizer moduleapi.Authorizer) bool {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	return ok && authorizer != nil && authorizer.Authorize(ctx, requestAuth, updatecontract.UpdateManagePermission.String()) == nil
}

func statusForUpdateViewer(ctx context.Context, authorizer moduleapi.Authorizer, status Status) Status {
	if mayViewComposeCandidates(ctx, authorizer) {
		return status
	}
	return status.withoutComposeCandidates()
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
		CandidateKey  string `json:"compose_candidate_key,omitempty"`
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
	operation, err := h.rollout.Start(c.Request.Context(), actor.ID, request.TargetVersion, request.CandidateKey)
	if err != nil {
		h.writeStartFailure(c, actor.ID, request.TargetVersion, request.CandidateKey, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusAccepted, operation)
}

func (h updateRouteHandlers) writeStartFailure(c *gin.Context, actorID uint64, targetVersion, candidateKey string, err error) {
	code, stage, operationID := rolloutFailureDetails(err)
	status := rolloutFailureHTTPStatus(code)
	requestAudit, _ := httpx.RequestAuditContextFromContext(c.Request.Context())
	logsafe.Error(h.logger, "platform update rollout start failed",
		zap.String("event", "platform_update.rollout_start_"+stage+"_failed"),
		zap.String("request_id", httpx.EnsureRequestID(c)),
		zap.String("trace_id", httpx.EnsureTraceID(c)),
		zap.Uint64("actor_id", actorID),
		zap.String("target_version", targetVersion),
		zap.String("compose_candidate_key", candidateKey),
		zap.Int("http_status", status),
		zap.String("failure_code", code),
		zap.String("failure_stage", stage),
		zap.String("operation_id", operationID),
		zap.String("route", requestAudit.Route),
		zap.String("error", sanitizeRolloutError(err)),
	)
	httpx.WriteLocalizedErrorCode(c, h.localizer, status, code, rolloutFailureMessageKey(code), rolloutFailureResponseData(code))
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
