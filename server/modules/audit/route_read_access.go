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
// 当 invalidField 为空时返回 true；否则返回 false，并以 HTTP 400 返回本地化的“invalid argument”错误，且携带 field=invalidField。
func ensureAuditListParamsBound(ginCtx *gin.Context, ctx *module.Context, invalidField string) bool {
	if invalidField == "" {
		return true
	}
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": invalidField,
	})
	return false
}

// requiresAuditManageVisibilityScope 判断审计可见性范围是否需要 AuditManage 权限。
// 当范围为 AuditVisibilityScopeAll 或 AuditVisibilityScopeHiddenOnly 时返回 true，否则返回 false。
func requiresAuditManageVisibilityScope(scope auditstore.AuditVisibilityScope) bool {
	switch scope {
	case auditstore.AuditVisibilityScopeAll, auditstore.AuditVisibilityScopeHiddenOnly:
		return true
	default:
		return false
	}
}

// ensureAuditVisibilityScopeAccess 在需要时校验审计可见性范围的管理权限。
// @param scope 要校验的审计可见性范围。
// @returns 当当前范围允许访问或已通过管理权限校验时返回 `true`，否则返回 `false`。
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

// handleAuditListReadError 处理审计列表读取过程中的错误。
// 当错误属于无效 scope 时，返回 HTTP 400；否则记录错误并返回 HTTP 500。
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

// isAuditInvalidScopeError 判断错误是否表示无效的可见性作用域。
// 当错误包装了 `drilldown.ErrScopeNotFound`、`drilldown.ErrScopeDisabled`、`drilldown.ErrTargetMismatch` 或 `drilldown.ErrScopeConflict` 中任一值时，返回 `true`，否则返回 `false`。
func isAuditInvalidScopeError(err error) bool {
	return errors.Is(err, drilldown.ErrScopeNotFound) ||
		errors.Is(err, drilldown.ErrScopeDisabled) ||
		errors.Is(err, drilldown.ErrTargetMismatch) ||
		errors.Is(err, drilldown.ErrScopeConflict)
}

// ensureAuditManageForVisibilityScope 校验当前请求是否具备审计可见性范围所需的管理权限。
// 成功时返回 true；当上下文解析失败或权限校验失败时，会按对应错误类型中止请求并返回 false。
// @returns 通过权限校验时为 true，否则为 false。
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

// resolveAuditManageAuthorizationInputs 提取审计管理授权所需的请求认证信息和鉴权器。
//
// @returns 成功时返回请求认证上下文、鉴权器和 `true`；当请求上下文、服务、认证信息或鉴权器缺失时，返回零值和 `false`。
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

// 当上下文或服务不可用，或解析结果不是有效的 `moduleapi.Authorizer` 时返回失败。
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

// handleAuditManageAuthorizationError 根据授权错误类型中止审计管理访问请求。
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

// abortAuditReadInternal 以内部错误中止当前审计读取请求。
//
// 当请求上下文可用时，它会返回 HTTP 500 并使用本地化的“internal error”消息。
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

// publishAuditManagePermissionDenied 发布审计管理权限被拒绝的安全审计事件。
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

// handleReadAuditLog 返回一个用于按 ID 读取审计日志详情的 Gin 处理器。
func handleReadAuditLog(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	return handleAuditReadByID(ctx, moduleName, auditLogReadConfig(reader))
}

// handleReadAuditIncident 返回一个按 ID 读取审计告警事件的路由处理器。
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

// 当日志条目为隐藏可见性时，需要审计管理权限；读取结果不存在时按未找到处理。
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

// auditIncidentReadConfig 构造按 ID 读取审计告警事件的配置。
// 当主事件或任一关联事件处于隐藏可见性时，需要管理权限才能读取该事件。
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

// newAuditReadConfig 组装按 ID 读取审计记录所需的配置。
//
// 它将元数据与读取、映射、访问控制和未找到判断逻辑合并为一个 auditReadByIDConfig。
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

// handleAuditReadByID 构造按 ID 读取审计记录的 Gin 处理器。
// 该处理器会依次绑定请求参数、读取记录、执行访问控制、映射响应并返回成功结果。
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

// bindAuditReadID 解析审计读取请求中的可选 ID 参数。
// 当参数存在且为大于 0 的 uint64 时返回该 ID；否则返回 400 的“invalid argument”错误，并包含字段名。
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

// readAuditRecordByID 按 ID 读取审计记录，并在未找到或读取失败时返回相应错误响应。
// 找不到记录时返回 404；其他读取错误会记录日志并返回 500。
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

// ensureAuditReadRecordAccess 根据记录的可见性要求校验审计管理权限。
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

// mapAuditReadRecord 将记录转换为响应载荷；转换失败时记录错误并返回内部错误响应。
// 成功时返回转换后的载荷和 true；失败时返回 nil 和 false。
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
