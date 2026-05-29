package auth

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/pluginapi"
)

const maxSessionListLimit = 100

type routeRuntime struct {
	localizer  *i18n.Service
	logger     *zap.Logger
	pluginName string
	authFlow   pluginapi.AuthFlowService
}

func (r authRouteRegistrar) runtime() routeRuntime {
	return routeRuntime{
		localizer:  r.ctx.I18n,
		logger:     r.ctx.Logger,
		pluginName: r.pluginName,
		authFlow:   r.authFlow,
	}
}

func (r routeRuntime) writeAuthRouteError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	mapped := r.authFlow.RouteError(err)
	if mapped.Status == http.StatusInternalServerError {
		logFields := append([]zap.Field{zap.String("plugin", r.pluginName), zap.Error(err)}, fields...)
		r.logger.Error(message, logFields...)
	}

	writeLocalizedContractError(ginCtx, r.localizer, mapped.Status, mapped.MessageKey, mapped.Data)
}

func (r routeRuntime) writeResponseMappingError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	logFields := append([]zap.Field{
		zap.String("plugin", r.pluginName),
		zap.String("requestId", httpx.EnsureRequestID(ginCtx)),
		zap.String("method", ginCtx.Request.Method),
		zap.String("route", ginCtx.FullPath()),
		zap.Error(err),
	}, fields...)
	r.logger.Error(message, logFields...)

	writeLocalizedContractError(ginCtx, r.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
}

func readSessionIDParam(ginCtx *gin.Context, localizer *i18n.Service) (string, bool) {
	sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
	if sessionID == "" {
		writeInvalidArgumentField(ginCtx, localizer, "sessionID")
		return "", false
	}

	return sessionID, true
}

func parseSessionListLimit(rawLimit string) (int, error) {
	rawLimit = strings.TrimSpace(rawLimit)
	if rawLimit == "" {
		return 0, nil
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return 0, err
	}
	if limit <= 0 || limit > maxSessionListLimit {
		return 0, strconv.ErrSyntax
	}

	return limit, nil
}

func clearRefreshCookieWhen(
	ginCtx *gin.Context,
	cookies CookieManager,
	matches func(*pluginapi.AccessTokenClaims) bool,
) {
	if matches == nil {
		return
	}

	requestAuth, ok := currentRequestAuth(ginCtx.Request.Context())
	if !ok || requestAuth.Claims == nil || !matches(requestAuth.Claims) {
		return
	}

	cookies.ClearRefreshCookie(ginCtx)
}

func writeLocalizedContractError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	status int,
	key string,
	data map[string]any,
) {
	httpx.WriteLocalizedError(ginCtx, localizer, status, key, data)
}

func writeInvalidArgumentField(ginCtx *gin.Context, localizer *i18n.Service, field string) {
	writeLocalizedContractError(ginCtx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": field,
	})
}

func abortLocalizedContractError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	status int,
	key string,
	data map[string]any,
) {
	writeLocalizedContractError(ginCtx, localizer, status, key, data)
	ginCtx.Abort()
}
