package audit

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/drilldown"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
)

// ensureAuditListParamsBound 在审计日志列表参数绑定失败时中止请求。
func ensureAuditListParamsBound(ginCtx *gin.Context, ctx *module.Context, invalidField string) bool {
	if invalidField == "" {
		return true
	}
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": invalidField,
	})
	return false
}

func requiresAuditManageVisibilityScope(scope auditstore.AuditVisibilityScope) bool {
	switch scope {
	case auditstore.AuditVisibilityScopeAll, auditstore.AuditVisibilityScopeHiddenOnly:
		return true
	default:
		return false
	}
}

func ensureAuditVisibilityScopeAccess(
	ginCtx *gin.Context,
	ctx *module.Context,
	scope auditstore.AuditVisibilityScope,
) bool {
	if !requiresAuditManageVisibilityScope(scope) {
		return true
	}
	return ensureAuditManageForVisibilityScope(ginCtx, ctx)
}

func handleAuditListReadError(
	ginCtx *gin.Context,
	ctx *module.Context,
	logger *zap.Logger,
	moduleName string,
	err error,
) bool {
	if isAuditInvalidScopeError(err) {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": "scope",
		})
		return true
	}
	logger.Error("list audit logs failed",
		zap.String("module", moduleName),
		zap.Error(err),
	)
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	return true
}

func isAuditInvalidScopeError(err error) bool {
	return errors.Is(err, drilldown.ErrScopeNotFound) ||
		errors.Is(err, drilldown.ErrScopeDisabled) ||
		errors.Is(err, drilldown.ErrTargetMismatch) ||
		errors.Is(err, drilldown.ErrScopeConflict)
}

func ensureAuditManageForVisibilityScope(ginCtx *gin.Context, ctx *module.Context) bool {
	requestAuth, authorizer, ok := resolveAuditManageAuthorizationInputs(ginCtx, ctx)
	if !ok {
		return false
	}
	if err := authorizer.Authorize(ginCtx.Request.Context(), requestAuth, auditcontract.AuditManagePermission.String()); err != nil {
		handleAuditManageAuthorizationError(ginCtx, ctx, requestAuth, err)
		return false
	}
	return true
}

func resolveAuditManageAuthorizationInputs(
	ginCtx *gin.Context,
	ctx *module.Context,
) (moduleapi.RequestAuthContext, moduleapi.Authorizer, bool) {
	if ginCtx == nil || ginCtx.Request == nil || ctx == nil || ctx.Services == nil {
		abortAuditReadInternal(ginCtx, ctx)
		return moduleapi.RequestAuthContext{}, nil, false
	}

	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return moduleapi.RequestAuthContext{}, nil, false
	}

	authorizer, ok := resolveAuditAuthorizer(ctx)
	if !ok {
		abortAuditReadInternal(ginCtx, ctx)
		return moduleapi.RequestAuthContext{}, nil, false
	}
	return requestAuth, authorizer, true
}

func resolveAuditAuthorizer(ctx *module.Context) (moduleapi.Authorizer, bool) {
	if ctx == nil || ctx.Services == nil {
		return nil, false
	}
	resolvedAuthorizer, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return nil, false
	}
	authorizer, ok := resolvedAuthorizer.(moduleapi.Authorizer)
	return authorizer, ok && authorizer != nil
}

func handleAuditManageAuthorizationError(
	ginCtx *gin.Context,
	ctx *module.Context,
	requestAuth moduleapi.RequestAuthContext,
	err error,
) {
	if errors.Is(err, moduleapi.ErrPermissionDenied) {
		publishAuditManagePermissionDenied(ginCtx, ctx, requestAuth)
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusForbidden, messagecontract.AuthForbidden.String(), nil)
		return
	}
	if errors.Is(err, moduleapi.ErrInvalidAccessToken) {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenInvalid.String(), nil)
		return
	}
	if errors.Is(err, moduleapi.ErrUnauthenticated) {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return
	}
	abortAuditReadInternal(ginCtx, ctx)
}

func abortAuditReadInternal(ginCtx *gin.Context, ctx *module.Context) {
	if ginCtx == nil || ginCtx.Request == nil {
		return
	}
	if ctx == nil {
		httpx.AbortLocalizedError(ginCtx, nil, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return
	}
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
}

func publishAuditManagePermissionDenied(ginCtx *gin.Context, ctx *module.Context, requestAuth moduleapi.RequestAuthContext) {
	if ginCtx == nil || ginCtx.Request == nil || ctx == nil || ctx.EventBus == nil {
		return
	}

	metadata := map[string]any{
		"permission": auditcontract.AuditManagePermission.String(),
		"requestId":  httpx.EnsureRequestID(ginCtx),
		"traceId":    httpx.EnsureTraceID(ginCtx),
		"route":      ginCtx.FullPath(),
		"path":       ginCtx.FullPath(),
		"method":     strings.TrimSpace(ginCtx.Request.Method),
		"status":     http.StatusForbidden,
		"module":     moduleID,
		"component":  "audit.visibility",
		"eventType":  "auth.permission.denied",
		"riskLevel":  "CRITICAL",
		"targetType": "permission",
		"targetId":   auditcontract.AuditManagePermission.String(),
		"targetName": auditcontract.AuditManagePermission.String(),
	}

	event := moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindSecurity,
		Action:        "auth.permission.denied",
		ResourceType:  "permission",
		ResourceID:    auditcontract.AuditManagePermission.String(),
		ResourceName:  auditcontract.AuditManagePermission.String(),
		RequestMethod: strings.TrimSpace(ginCtx.Request.Method),
		RequestPath:   ginCtx.FullPath(),
		StatusCode:    http.StatusForbidden,
		RequestID:     httpx.EnsureRequestID(ginCtx),
		IP:            strings.TrimSpace(ginCtx.ClientIP()),
		UserAgent:     strings.TrimSpace(ginCtx.Request.UserAgent()),
		Success:       false,
		Message:       messagecontract.AuthForbidden.String(),
		Metadata:      metadata,
	}
	if requestAuth.User != nil {
		user := *requestAuth.User
		event.Operator = &user
		event.Metadata["actorId"] = strconv.FormatUint(user.ID, 10)
		event.Metadata["actorType"] = "user"
	}

	_ = ctx.EventBus.Publish(ginCtx.Request.Context(), eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  moduleID,
		Payload: event,
	})
}

func handleReadAuditLog(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	return handleAuditReadByID(ctx, moduleName, auditLogReadConfig(reader))
}

func handleReadAuditIncident(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	return handleAuditReadByID(ctx, moduleName, auditIncidentReadConfig(reader))
}

type auditReadByIDConfig[T any] struct {
	param          string
	invalidField   string
	read           func(context.Context, uint64) (T, error)
	mapper         func(T) (any, error)
	requiresManage func(T) bool
	isNotFound     func(error) bool
	notFoundField  string
	readLogMessage string
	mapLogMessage  string
}

type auditReadByIDMeta struct {
	param          string
	field          string
	readLogMessage string
	mapLogMessage  string
}

func auditLogReadConfig(reader auditReader) auditReadByIDConfig[auditDetailResult] {
	return newAuditReadConfig(
		auditReadByIDMeta{
			param:          auditcontract.AuditLogParam,
			field:          "id",
			readLogMessage: "read audit log detail failed",
			mapLogMessage:  "map audit log detail response failed",
		},
		func(requestCtx context.Context, id uint64) (auditDetailResult, error) {
			return reader.Detail(requestCtx, id)
		},
		func(result auditDetailResult) (any, error) {
			return toAuditLogDetailResponse(result)
		},
		func(result auditDetailResult) bool {
			return result.Visibility == auditstore.AuditVisibilityStrategyHidden
		},
		func(err error) bool {
			return errors.Is(err, auditstore.ErrAuditLogNotFound)
		},
	)
}

func auditIncidentReadConfig(reader auditReader) auditReadByIDConfig[auditIncidentResult] {
	return newAuditReadConfig(
		auditReadByIDMeta{
			param:          auditcontract.AuditIncidentParam,
			field:          "event_id",
			readLogMessage: "read audit incident failed",
			mapLogMessage:  "map audit incident response failed",
		},
		func(requestCtx context.Context, id uint64) (auditIncidentResult, error) {
			return reader.Incident(requestCtx, id)
		},
		func(result auditIncidentResult) (any, error) {
			return toAuditIncidentResponse(result)
		},
		func(result auditIncidentResult) bool {
			if result.SeedEvent.Visibility == auditstore.AuditVisibilityStrategyHidden {
				return true
			}
			for _, item := range result.RelatedEvents {
				if item.Visibility == auditstore.AuditVisibilityStrategyHidden {
					return true
				}
			}
			return false
		},
		func(err error) bool {
			return errors.Is(err, auditstore.ErrIncidentNotFound)
		},
	)
}

func newAuditReadConfig[T any](
	meta auditReadByIDMeta,
	read func(context.Context, uint64) (T, error),
	mapper func(T) (any, error),
	requiresManage func(T) bool,
	isNotFound func(error) bool,
) auditReadByIDConfig[T] {
	return auditReadByIDConfig[T]{
		param:          meta.param,
		invalidField:   meta.field,
		read:           read,
		mapper:         mapper,
		requiresManage: requiresManage,
		isNotFound:     isNotFound,
		notFoundField:  meta.field,
		readLogMessage: meta.readLogMessage,
		mapLogMessage:  meta.mapLogMessage,
	}
}

func handleAuditReadByID[T any](
	ctx *module.Context,
	moduleName string,
	config auditReadByIDConfig[T],
) gin.HandlerFunc {
	logger := auditRouteLogger(ctx)

	return func(ginCtx *gin.Context) {
		id, ok := bindAuditReadID(ginCtx, ctx, config)
		if !ok {
			return
		}
		recordLogger := logger.With(
			zap.String("module", moduleName),
			zap.Uint64("id", id),
		)

		record, ok := readAuditRecordByID(ginCtx, ctx, recordLogger, config, id)
		if !ok {
			return
		}
		if !ensureAuditReadRecordAccess(ginCtx, ctx, config, record) {
			return
		}

		payload, ok := mapAuditReadRecord(ginCtx, ctx, recordLogger, config, record)
		if !ok {
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

func bindAuditReadID[T any](
	ginCtx *gin.Context,
	ctx *module.Context,
	config auditReadByIDConfig[T],
) (uint64, bool) {
	id, ok, err := parseOptionalUint64Param(ginCtx, config.param)
	if err == nil && ok && id > 0 {
		return id, true
	}
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": config.invalidField,
	})
	return 0, false
}

func readAuditRecordByID[T any](
	ginCtx *gin.Context,
	ctx *module.Context,
	logger *zap.Logger,
	config auditReadByIDConfig[T],
	id uint64,
) (T, bool) {
	record, readErr := config.read(withAuditRequestLocale(ginCtx, ctx), id)
	if readErr == nil {
		return record, true
	}
	if config.isNotFound(readErr) {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusNotFound, "common.not_found", map[string]any{
			"field": config.notFoundField,
		})
		return record, false
	}
	logger.Error(config.readLogMessage,
		zap.Error(readErr),
	)
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	return record, false
}

func ensureAuditReadRecordAccess[T any](
	ginCtx *gin.Context,
	ctx *module.Context,
	config auditReadByIDConfig[T],
	record T,
) bool {
	if config.requiresManage == nil || !config.requiresManage(record) {
		return true
	}
	return ensureAuditManageForVisibilityScope(ginCtx, ctx)
}

func mapAuditReadRecord[T any](
	ginCtx *gin.Context,
	ctx *module.Context,
	logger *zap.Logger,
	config auditReadByIDConfig[T],
	record T,
) (any, bool) {
	payload, mapErr := config.mapper(record)
	if mapErr == nil {
		return payload, true
	}
	logger.Error(config.mapLogMessage,
		zap.Error(mapErr),
	)
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	return nil, false
}
