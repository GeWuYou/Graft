package user

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	applog "graft/server/internal/logger"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	authruntime "graft/server/modules/auth"
	userstore "graft/server/modules/user/store"
)

type routeRuntime struct {
	localizer  *i18n.Service
	logger     *zap.Logger
	appLog     applog.AppLogger
	moduleName string
}

func (r userRouteRegistrar) runtime() routeRuntime {
	return routeRuntime{
		localizer:  r.ctx.I18n,
		logger:     r.ctx.Logger,
		appLog:     r.appLog,
		moduleName: r.moduleName,
	}
}

func (r routeRuntime) appLogger() applog.AppLogger {
	if r.appLog != nil {
		return r.appLog.Named("modules.user.route")
	}

	return applog.NewAppLogger(r.logger).Named("modules.user.route")
}

func resolveUserRouteAppLogger(ctx *module.Context) applog.AppLogger {
	if ctx == nil || ctx.Services == nil {
		return nil
	}

	resolved, err := ctx.Services.Resolve((*applog.AppLogger)(nil))
	if err != nil {
		return nil
	}

	appLogger, ok := resolved.(applog.AppLogger)
	if !ok || appLogger == nil {
		return nil
	}

	return appLogger
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
	cookies authruntime.CookieManager,
	matches func(*moduleapi.AccessTokenClaims) bool,
) {
	if matches == nil {
		return
	}

	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || requestAuth.Claims == nil || !matches(requestAuth.Claims) {
		return
	}

	cookies.ClearRefreshCookie(ginCtx)
}

func (r routeRuntime) writeAuthRouteError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	status, messageKey := mapAuthError(err)
	if status == http.StatusInternalServerError {
		logFields := append([]zap.Field{zap.String("module", r.moduleName), zap.Error(err)}, fields...)
		logsafe.Error(r.logger, message, logFields...)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, authErrorDetails(err))
}

func (r routeRuntime) writeUserLookupError(ginCtx *gin.Context, userID uint64, message string, err error) {
	status := http.StatusInternalServerError
	messageKey := messagecontract.CommonInternalError
	if errors.Is(err, userstore.ErrUserNotFound) || errors.Is(err, moduleapi.ErrUserNotFound) {
		status = http.StatusNotFound
		messageKey = messagecontract.UserNotFound
	} else {
		logsafe.Error(r.logger, message,
			zap.String("module", r.moduleName),
			zap.Uint64("userID", userID),
			zap.Error(err),
		)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, nil)
}

func (r routeRuntime) writeUserManagementError(ginCtx *gin.Context, userID uint64, message string, err error) {
	status, messageKey, data := mapUserManagementError(err, passwordFieldForUserManagementOperation(message))
	if shouldLogUserManagementError(status, err) {
		responseCode := errorCodeFromMessageKey(messageKey)
		logFields := []zap.Field{
			zap.String("module", r.moduleName),
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
		logsafe.Error(r.logger, message,
			logFields...,
		)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, data)
}

func (r routeRuntime) writeCreateUserError(ginCtx *gin.Context, message string, err error) {
	status, messageKey, data := mapUserManagementError(err, "password")
	if shouldLogUserManagementError(status, err) {
		responseCode := errorCodeFromMessageKey(messageKey)
		logFields := []zap.Field{
			zap.String("module", r.moduleName),
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
		logsafe.Error(r.logger, message, logFields...)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, data)
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

	writeLocalizedContractError(ginCtx, r.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
}

// zapFieldValue 将 zap 字段转换为日志所需的底层值，避免错误映射逻辑依赖具体字段实现。
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

// shouldLogUserManagementError 根据 HTTP 状态判断用户管理错误是否应记录；只对服务端内部错误返回 true。
func shouldLogUserManagementError(status int, _ error) bool {
	return status == http.StatusInternalServerError
}

// errorFieldFromDetails 从错误详情中提取非空字段名；只有详情包含有效字段值时才返回该字段和 true。
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

func passwordFieldForUserManagementOperation(message string) string {
	if message == "reset user password failed" {
		return "new_password"
	}

	return ""
}

func errorCodeFromMessageKey(key messagecontract.Key) string {
	return errorcode.FromMessageKey(key).String()
}

// mapUserManagementError 将用户管理错误映射为 HTTP 状态码、本地化消息键和可选的字段信息。
// passwordField 由调用路由提供，以保留 password 与 new_password 的请求契约差异。
func mapUserManagementError(err error, passwordField string) (int, messagecontract.Key, map[string]any) {
	switch {
	case errors.Is(err, userstore.ErrUserNotFound), errors.Is(err, moduleapi.ErrUserNotFound):
		return http.StatusNotFound, messagecontract.UserNotFound, nil
	case errors.Is(err, userstore.ErrUsernameConflict):
		return http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "username"}
	case errors.Is(err, errInvalidUserPayload):
		return http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "body"}
	case errors.Is(err, errInvalidUserStatus):
		return http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "status"}
	case errors.Is(err, errCannotDisableOwnUser), errors.Is(err, errCannotDeleteOwnUser):
		return http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "id"}
	case errors.Is(err, errProtectedDefaultAdminImmutable):
		return http.StatusForbidden, messagecontract.UserProtectedDefaultAdminImmutable, nil
	case errors.Is(err, moduleapi.ErrPasswordPolicyViolation):
		return http.StatusBadRequest, messagecontract.AuthPasswordPolicyViolation, map[string]any{"field": passwordField}
	case errors.Is(err, moduleapi.ErrPasswordReuseForbidden):
		return http.StatusBadRequest, messagecontract.AuthPasswordReuseForbidden, map[string]any{"field": passwordField}
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
