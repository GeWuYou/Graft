package auth

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	applog "graft/server/internal/logger"
	"graft/server/internal/moduleapi"
)

const maxSessionListLimit = 100

type routeRuntime struct {
	localizer  *i18n.Service
	logger     *zap.Logger
	moduleName string
	authFlow   moduleapi.AuthFlowService
}

func (r authRouteRegistrar) runtime() routeRuntime {
	return routeRuntime{
		localizer:  r.ctx.I18n,
		logger:     r.ctx.Logger,
		moduleName: r.moduleName,
		authFlow:   r.authFlow,
	}
}

func (r routeRuntime) appLogger() applog.AppLogger {
	return applog.NewAppLogger(r.logger).Named("modules.auth.route")
}

func (r routeRuntime) writeAuthRouteError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	mapped := r.authFlow.RouteError(err)
	if mapped.Status == http.StatusInternalServerError {
		logFields := append([]zap.Field{zap.String("module", r.moduleName), zap.Error(err)}, fields...)
		r.logger.Error(message, logFields...)
	}

	writeLocalizedContractError(ginCtx, r.localizer, mapped.Status, mapped.MessageKey, mapped.Data)
}

func (r routeRuntime) writeResponseMappingError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	appFields := []applog.Field{
		applog.StringField("module", r.moduleName),
		applog.ErrorField(err),
	}
	for _, field := range fields {
		appFields = append(appFields, applog.Field{
			Key:   field.Key,
			Value: zapFieldValue(field),
		})
	}
	r.appLogger().Error(ginCtx.Request.Context(), message, appFields...)

	writeLocalizedContractError(ginCtx, r.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
}

func zapFieldValue(field zap.Field) any {
	switch field.Type {
	case zapcore.StringType:
		return field.String
	case zapcore.Uint64Type:
		return field.Integer
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return field.Integer
	case zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type, zapcore.UintptrType:
		return field.Integer
	case zapcore.BoolType:
		return field.Integer == 1
	default:
		return field.Interface
	}
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
	matches func(*moduleapi.AccessTokenClaims) bool,
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
