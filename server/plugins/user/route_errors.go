package user

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/pluginapi"
	userstore "graft/server/plugins/user/store"
)

type routeRuntime struct {
	localizer  *i18n.Service
	logger     *zap.Logger
	pluginName string
}

func (r authRouteRegistrar) runtime() routeRuntime {
	return routeRuntime{
		localizer:  r.ctx.I18n,
		logger:     r.ctx.Logger,
		pluginName: r.pluginName,
	}
}

func (r userRouteRegistrar) runtime() routeRuntime {
	return routeRuntime{
		localizer:  r.ctx.I18n,
		logger:     r.ctx.Logger,
		pluginName: r.pluginName,
	}
}

func readUserIDParam(ginCtx *gin.Context, localizer *i18n.Service) (uint64, bool) {
	rawID, err := parseUserID(ginCtx.Param("id"))
	if err != nil {
		writeInvalidArgumentField(ginCtx, localizer, "id")
		return 0, false
	}

	return rawID, true
}

func readSessionIDParam(ginCtx *gin.Context, localizer *i18n.Service) (string, bool) {
	sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
	if sessionID == "" {
		writeInvalidArgumentField(ginCtx, localizer, "sessionID")
		return "", false
	}

	return sessionID, true
}

func clearRefreshCookieWhen(
	ginCtx *gin.Context,
	authSvc *authService,
	matches func(*pluginapi.AccessTokenClaims) bool,
) {
	if authSvc == nil || matches == nil {
		return
	}

	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || requestAuth.Claims == nil || !matches(requestAuth.Claims) {
		return
	}

	authSvc.cookies.clearRefreshCookie(ginCtx)
}

func (r routeRuntime) writeAuthRouteError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	status, messageKey := mapAuthError(err)
	if status == http.StatusInternalServerError {
		logFields := append([]zap.Field{zap.String("plugin", r.pluginName), zap.Error(err)}, fields...)
		r.logger.Error(message, logFields...)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, authErrorDetails(err))
}

func (r routeRuntime) writeUserLookupError(ginCtx *gin.Context, userID uint64, message string, err error) {
	status := http.StatusInternalServerError
	messageKey := messagecontract.CommonInternalError
	if errors.Is(err, userstore.ErrUserNotFound) || errors.Is(err, pluginapi.ErrUserNotFound) {
		status = http.StatusNotFound
		messageKey = messagecontract.UserNotFound
	} else {
		r.logger.Error(message,
			zap.String("plugin", r.pluginName),
			zap.Uint64("userID", userID),
			zap.Error(err),
		)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, nil)
}

func (r routeRuntime) writeUserManagementError(ginCtx *gin.Context, userID uint64, message string, err error) {
	status, messageKey, data := mapUserManagementError(err)
	if shouldLogUserManagementError(status, err) {
		responseCode := errorCodeFromMessageKey(messageKey)
		logFields := []zap.Field{
			zap.String("plugin", r.pluginName),
			zap.String("operation", userManagementOperationFromMessage(message)),
			zap.String("method", ginCtx.Request.Method),
			zap.String("route", ginCtx.FullPath()),
			zap.String("response_code", responseCode),
			zap.String("message_key", messageKey.String()),
			zap.Uint64("userID", userID),
			zap.Error(err),
		}
		if field, ok := errorFieldFromDetails(data); ok {
			logFields = append(logFields, zap.String("field", field))
		}
		r.logger.Error(message,
			logFields...,
		)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, data)
}

func (r routeRuntime) writeCreateUserError(ginCtx *gin.Context, message string, err error) {
	status, messageKey, data := mapUserManagementError(err)
	if field, ok := errorFieldFromDetails(data); ok && field == "new_password" {
		data = map[string]any{"field": "password"}
	}
	if shouldLogUserManagementError(status, err) {
		responseCode := errorCodeFromMessageKey(messageKey)
		logFields := []zap.Field{
			zap.String("plugin", r.pluginName),
			zap.String("operation", userManagementOperationFromMessage(message)),
			zap.String("method", ginCtx.Request.Method),
			zap.String("route", ginCtx.FullPath()),
			zap.String("response_code", responseCode),
			zap.String("message_key", messageKey.String()),
			zap.Error(err),
		}
		if field, ok := errorFieldFromDetails(data); ok {
			logFields = append(logFields, zap.String("field", field))
		}
		r.logger.Error(message, logFields...)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, data)
}

func shouldLogUserManagementError(status int, err error) bool {
	return status == http.StatusInternalServerError ||
		errors.Is(err, errPasswordPolicyViolation) ||
		errors.Is(err, errPasswordReuseForbidden)
}

func errorFieldFromDetails(data map[string]any) (string, bool) {
	if data == nil {
		return "", false
	}
	field, ok := data["field"].(string)
	if !ok || field == "" {
		return "", false
	}
	return field, true
}

func userManagementOperationFromMessage(message string) string {
	switch message {
	case "create user failed":
		return "create_user"
	case "update user failed":
		return "update_user"
	case "set user status failed":
		return "set_user_status"
	case "reset user password failed":
		return "reset_user_password"
	case "delete user failed":
		return "delete_user"
	default:
		return "user_management"
	}
}

func errorCodeFromMessageKey(key messagecontract.Key) string {
	return errorcode.FromMessageKey(key).String()
}

func mapUserManagementError(err error) (int, messagecontract.Key, map[string]any) {
	switch {
	case errors.Is(err, userstore.ErrUserNotFound), errors.Is(err, pluginapi.ErrUserNotFound):
		return http.StatusNotFound, messagecontract.UserNotFound, nil
	case errors.Is(err, userstore.ErrUsernameConflict):
		return http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "username"}
	case errors.Is(err, errInvalidUserPayload):
		return http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "body"}
	case errors.Is(err, errInvalidUserStatus):
		return http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "status"}
	case errors.Is(err, errCannotDisableOwnUser), errors.Is(err, errCannotDeleteOwnUser):
		return http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "id"}
	case errors.Is(err, errPasswordPolicyViolation), errors.Is(err, errPasswordReuseForbidden):
		status, key := mapAuthError(err)
		return status, key, map[string]any{"field": "new_password"}
	default:
		return http.StatusInternalServerError, messagecontract.CommonInternalError, nil
	}
}

func writeLocalizedContractError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	status int,
	key messagecontract.Key,
	data map[string]any,
) {
	httpx.WriteLocalizedError(ginCtx, localizer, status, key.String(), data)
}

func writeInvalidArgumentField(ginCtx *gin.Context, localizer *i18n.Service, field string) {
	writeLocalizedContractError(ginCtx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
		"field": field,
	})
}

func abortLocalizedContractError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	status int,
	key messagecontract.Key,
	data map[string]any,
) {
	writeLocalizedContractError(ginCtx, localizer, status, key, data)
	ginCtx.Abort()
}
