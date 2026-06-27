package audit

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpheader "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/drilldown"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
	"graft/server/modules/audit/storeent"
)

type auditReader interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
	Detail(ctx context.Context, id uint64) (DetailResult, error)
	Overview(ctx context.Context, preset auditstore.AuditTimePreset) (OverviewResult, error)
	Incident(ctx context.Context, eventID uint64) (IncidentResult, error)
	VisibilityPolicy(ctx context.Context) (VisibilityPolicyResult, error)
	UpdateVisibilityDefault(
		ctx context.Context,
		strategy auditstore.AuditVisibilityStrategy,
		userID *uint64,
		username string,
	) (auditstore.AuditVisibilityDefault, error)
	UpdateVisibilityOverride(ctx context.Context, input auditstore.UpsertAuditVisibilityOverrideInput) (auditstore.AuditVisibilityOverride, error)
	DeleteVisibilityOverride(ctx context.Context, source auditstore.AuditSource, actionKey string) error
}

type auditListResult = ListResult
type auditDetailResult = DetailResult
type auditOverviewResult = OverviewResult
type auditIncidentResult = IncidentResult

type auditGuard struct {
	read   gin.HandlerFunc
	manage gin.HandlerFunc
}

// handleListAuditLogs 创建审计日志列表查询的处理器。
// 它绑定列表查询参数，校验可见性范围访问权限，读取审计日志列表，并将结果转换为响应。
func handleListAuditLogs(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := auditRouteLogger(ctx)

	return func(ginCtx *gin.Context) {
		_, query, invalidField := bindGeneratedAuditListParams(ginCtx)
		if !ensureAuditListParamsBound(ginCtx, ctx, invalidField) {
			return
		}

		if !ensureAuditVisibilityScopeAccess(ginCtx, ctx, query.VisibilityScope) {
			return
		}

		result, err := reader.List(withAuditRequestLocale(ginCtx, ctx), query)
		if err != nil {
			if handleAuditListReadError(ginCtx, ctx, logger, moduleName, err) {
				return
			}
			return
		}

		payload, mapErr := toAuditLogListResponse(result)
		if mapErr != nil {
			logger.Error("map audit logs response failed",
				zap.String("module", moduleName),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

// ensureAuditListParamsBound 在审计日志列表参数绑定失败时中止请求。
//
// @param invalidField 绑定失败的字段名。
// @returns 当参数绑定成功时返回 true；当存在无效字段并已中止请求时返回 false。
func ensureAuditListParamsBound(ginCtx *gin.Context, ctx *module.Context, invalidField string) bool {
	if invalidField == "" {
		return true
	}
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": invalidField,
	})
	return false
}

// requiresAuditManageVisibilityScope 判断给定可见性范围是否需要 manage 权限。
// `true` 仅在范围为 `AuditVisibilityScopeAll` 或 `AuditVisibilityScopeHiddenOnly` 时返回。
func requiresAuditManageVisibilityScope(scope auditstore.AuditVisibilityScope) bool {
	switch scope {
	case auditstore.AuditVisibilityScopeAll, auditstore.AuditVisibilityScopeHiddenOnly:
		return true
	default:
		return false
	}
}

// ensureAuditVisibilityScopeAccess 在需要管理权限时校验审计可见性范围访问。
// 当可见性范围属于需要 manage 权限的策略时，执行相应的权限检查；否则直接允许访问。
// 返回 `true` 表示允许访问，`false` 表示访问被拒绝。
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

// handleAuditListReadError 处理审计日志列表读取失败。
// 当错误属于无效 scope 时，返回 400 并标记字段为 "scope"；否则记录错误并返回 500。
// 
// @returns 始终返回 `true`。
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

// isAuditInvalidScopeError 判断错误是否表示审计范围无效。
// `true` if err 包装了 `drilldown.ErrScopeNotFound`、`drilldown.ErrScopeDisabled`、`drilldown.ErrTargetMismatch` 或 `drilldown.ErrScopeConflict`，否则为 `false`。
func isAuditInvalidScopeError(err error) bool {
	return errors.Is(err, drilldown.ErrScopeNotFound) ||
		errors.Is(err, drilldown.ErrScopeDisabled) ||
		errors.Is(err, drilldown.ErrTargetMismatch) ||
		errors.Is(err, drilldown.ErrScopeConflict)
}

// ensureAuditManageForVisibilityScope 校验当前请求是否具有审计可见性管理权限。
// 当授权信息解析失败或权限校验未通过时，返回 false 并完成相应的请求中止处理。
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

// resolveAuditManageAuthorizationInputs 获取审计管理权限校验所需的请求鉴权信息和授权器。
// 当 gin 上下文、请求、模块上下文或服务容器缺失时，它会中止请求并返回失败；
// 当请求中缺少鉴权信息时，它会返回未授权错误并返回失败。
// @returns 请求鉴权上下文、授权器以及是否成功解析。
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

// resolveAuditAuthorizer 尝试从模块服务中解析授权器。
// 成功时返回解析到的授权器及 true；否则返回 nil 和 false。
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

// handleAuditManageAuthorizationError 处理审计管理授权失败。
// 当错误属于权限拒绝、未认证或访问令牌无效时，发布权限拒绝审计事件并返回 403；
// 其他错误则返回内部错误。
func handleAuditManageAuthorizationError(
	ginCtx *gin.Context,
	ctx *module.Context,
	requestAuth moduleapi.RequestAuthContext,
	err error,
) {
	if errors.Is(err, moduleapi.ErrPermissionDenied) || errors.Is(err, moduleapi.ErrUnauthenticated) || errors.Is(err, moduleapi.ErrInvalidAccessToken) {
		publishAuditManagePermissionDenied(ginCtx, ctx, requestAuth)
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusForbidden, messagecontract.AuthForbidden.String(), nil)
		return
	}
	abortAuditReadInternal(ginCtx, ctx)
}

// abortAuditReadInternal 以内部错误中止审计读取请求。
func abortAuditReadInternal(ginCtx *gin.Context, ctx *module.Context) {
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
}

// publishAuditManagePermissionDenied 记录审计管理权限被拒绝事件。
// 当请求缺少必要上下文时直接返回；否则构造一条权限拒绝审计事件并发布到事件总线。
// 如果提供了用户鉴权信息，事件中会包含操作者信息。
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

// handleReadAuditLog 返回用于读取审计日志详情的 Gin 处理器。
// 当日志记录处于隐藏可见性策略时，处理器会要求相应的管理权限。
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

// auditLogReadConfig 构造审计日志详情读取配置。
// 当日志记录的可见性为隐藏时，会要求 manage 权限，并将不存在的日志视为未找到。
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

// auditIncidentReadConfig 构造按事件 ID 读取审计事件详情的配置。
// 当种子事件或任一关联事件的可见性为 hidden 时，要求管理权限后才能访问。
// `auditstore.ErrIncidentNotFound` 会被视为未找到。
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

// newAuditReadConfig 根据给定的元信息和处理函数构建审计按 ID 读取配置。
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

// handleAuditReadByID 返回一个用于按 ID 读取审计记录的 Gin 处理器。
// 处理器会解析请求中的 ID、读取记录、校验访问权限、映射响应并写回成功结果。
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

// bindAuditReadID 解析用于按 ID 读取审计记录的参数。
// 解析失败或参数缺失时，中止请求并返回 400。
func bindAuditReadID[T any](
	ginCtx *gin.Context,
	ctx *module.Context,
	config auditReadByIDConfig[T],
) (uint64, bool) {
	id, ok, err := parseOptionalUint64Param(ginCtx, config.param)
	if err == nil && ok {
		return id, true
	}
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": config.invalidField,
	})
	return 0, false
}

// readAuditRecordByID 按 ID 读取审计记录，并在读取失败时返回相应的错误响应。
// 读取未命中时返回 404；其他读取错误记录日志并返回 500。
// 成功时返回读取到的记录和 true。
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

// ensureAuditReadRecordAccess 在记录需要管理权限时校验可见性访问。
// 当 config.requiresManage 未设置或对该记录返回 false 时，允许访问；否则执行管理权限校验。
// `true` 表示允许继续处理，`false` 表示已被拒绝或中止。
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

// mapAuditReadRecord 将审计记录映射为响应载荷，并在映射失败时中止请求。
//
// @param record 要转换的审计记录。
// @returns 转换后的响应载荷及是否转换成功。
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

// handleReadAuditOverview 返回审计概览读取处理器。
// 它会绑定概览查询参数，规范化时间预设，读取概览数据并转换为响应后返回成功结果。
func handleReadAuditOverview(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := zap.NewNop()
	if ctx != nil && ctx.Logger != nil {
		logger = ctx.Logger
	}

	return func(ginCtx *gin.Context) {
		params := bindGeneratedAuditOverviewParams(ginCtx)
		preset := normalizeAuditOverviewPreset(params.Preset)

		result, err := reader.Overview(withAuditRequestLocale(ginCtx, ctx), preset)
		if err != nil {
			logger.Error("read audit overview failed",
				zap.String("module", moduleName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		payload, mapErr := toAuditOverviewResponse(result)
		if mapErr != nil {
			logger.Error("map audit overview response failed",
				zap.String("module", moduleName),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

type auditReadGeneratedHandler struct{}

var auditAllowedListQueryKeys = map[string]struct{}{
	"page":                    {},
	"page_size":               {},
	"actor_user_id":           {},
	"keyword":                 {},
	"actor":                   {},
	"action":                  {},
	"preset":                  {},
	"scope":                   {},
	"business_category":       {},
	"action_prefix":           {},
	"action_prefixes":         {},
	"action_prefixes[]":       {},
	"action_keywords":         {},
	"action_keywords[]":       {},
	"source":                  {},
	"resource_type":           {},
	"resource_types":          {},
	"resource_types[]":        {},
	"resource_id":             {},
	"resource_name":           {},
	"request_path_prefixes":   {},
	"request_path_prefixes[]": {},
	"request_id":              {},
	"session_id":              {},
	"result":                  {},
	"results":                 {},
	"results[]":               {},
	"risk_level":              {},
	"risk_levels":             {},
	"risk_levels[]":           {},
	"visibility_scope":        {},
	"success":                 {},
	"created_from":            {},
	"created_to":              {},
	"sort":                    {},
	"sort[]":                  {},
}

func withAuditRequestLocale(ginCtx *gin.Context, ctx *module.Context) context.Context {
	requestCtx := context.Background()
	if ginCtx != nil && ginCtx.Request != nil {
		requestCtx = ginCtx.Request.Context()
	}
	if ctx == nil || ctx.I18n == nil || ginCtx == nil {
		return requestCtx
	}

	locale := ctx.I18n.ResolveRequestLocale(ginCtx.Request, "")
	return storeent.WithAuditLocale(requestCtx, locale)
}

func (h auditReadGeneratedHandler) GetAuditLogs(params auditopenapi.GetAuditLogsParams) {
	_ = h
	_ = params
}

func (h auditReadGeneratedHandler) GetAuditLogDetail(id int64, params auditopenapi.GetAuditLogDetailParams) {
	_ = h
	_ = id
	_ = params
}

func (h auditReadGeneratedHandler) GetAuditOverview(params auditopenapi.GetAuditOverviewParams) {
	_ = h
	_ = params
}

func (h auditReadGeneratedHandler) GetAuditIncident(params auditopenapi.GetAuditIncidentParams) {
	_ = h
	_ = params
}

// bindGeneratedAuditListParams 绑定并校验审计日志列表查询参数。
// 返回 OpenAPI 请求参数、内部查询条件，以及首个无效字段名；若校验通过则字段名为空。
func bindGeneratedAuditListParams(
	ginCtx *gin.Context,
) (auditopenapi.GetAuditLogsParams, ListQuery, string) {
	params := newAuditListParams(ginCtx)
	query := ListQuery{}

	if field := bindAuditPrimaryFilters(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	bindAuditStringFilters(ginCtx, &params, &query)
	bindAuditStringSliceFilters(ginCtx, &params, &query)
	if field := bindAuditSecondaryFilters(ginCtx, &params, &query); field != "" {
		return params, query, field
	}
	if field := rejectUnknownAuditListQueryKeys(ginCtx); field != "" {
		return params, query, field
	}

	return params, query, ""
}

// 发生校验失败时，返回首个无效字段名。
func bindAuditPrimaryFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if field := bindAuditPagination(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditActorUserID(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditPreset(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditScope(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditVisibilityScope(ginCtx, query); field != "" {
		return field
	}
	return ""
}

// bindAuditSecondaryFilters 依次绑定审计日志的次级筛选条件：枚举类筛选、success、创建时间范围和排序。
// 发生校验失败时返回对应的字段名；全部成功时返回空字符串。
func bindAuditSecondaryFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if field := bindAuditEnumFilters(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditSuccessFilter(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditCreatedRange(ginCtx, params, query); field != "" {
		return field
	}
	if field := bindAuditSort(ginCtx, params, query); field != "" {
		return field
	}
	return ""
}

// newAuditListParams 从请求头构造审计日志列表参数，填充语言环境和请求 ID。
func newAuditListParams(ginCtx *gin.Context) auditopenapi.GetAuditLogsParams {
	locale, requestID := bindGeneratedAuditReadHeaders(ginCtx)
	return auditopenapi.GetAuditLogsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindAuditPagination(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	page, ok, err := parseOptionalIntQuery(ginCtx, "page")
	if err != nil {
		return "page"
	}
	if ok {
		params.Page = &page
		query.Page = page
	}

	pageSize, ok, err := parseOptionalIntQuery(ginCtx, "page_size")
	if err != nil {
		return "page_size"
	}
	if ok {
		params.PageSize = &pageSize
		query.PageSize = pageSize
	}

	return ""
}

func bindAuditActorUserID(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	value, ok, err := parseOptionalUint64Query(ginCtx, "actor_user_id")
	if err != nil {
		return "actor_user_id"
	}
	if !ok {
		return ""
	}

	converted, convErr := mustConvertAuditGeneratedID(value, "audit actor user id query")
	if convErr != nil {
		return "actor_user_id"
	}
	params.ActorUserId = &converted
	query.ActorUserID = &value
	return ""
}

func bindAuditPreset(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.TrimSpace(ginCtx.Query("preset")); raw != "" {
		value := auditopenapi.GetAuditLogsParamsPreset(raw)
		if !value.Valid() {
			return "preset"
		}
		params.Preset = &value
		query.TimePreset = auditstore.AuditTimePreset(raw)
	}

	return ""
}

// bindAuditScope 解析并验证审计日志列表的 scope 查询参数。
// 成功时将值写入请求参数和内部查询；无效时返回 "scope"。
func bindAuditScope(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.TrimSpace(ginCtx.Query("scope")); raw != "" {
		value := auditopenapi.GetAuditLogsParamsScope(raw)
		if !value.Valid() {
			return "scope"
		}
		params.Scope = &value
		query.Scope = raw
	}
	return ""
}

// bindAuditVisibilityScope 解析并校验审计列表的可见性范围。
// 当请求中提供有效值时，会将其写入查询条件；无效时返回 "visibility_scope"。
func bindAuditVisibilityScope(ginCtx *gin.Context, query *ListQuery) string {
	raw := strings.TrimSpace(ginCtx.Query("visibility_scope"))
	if raw == "" {
		return ""
	}
	switch auditstore.AuditVisibilityScope(raw) {
	case auditstore.AuditVisibilityScopeDefault,
		auditstore.AuditVisibilityScopeAll,
		auditstore.AuditVisibilityScopeHiddenOnly:
		query.VisibilityScope = auditstore.AuditVisibilityScope(raw)
		return ""
	default:
		return "visibility_scope"
	}
}

// bindAuditStringFilters 绑定审计日志列表的字符串过滤条件，并同步写入 API 参数与内部查询。
func bindAuditStringFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) {
	bindAuditStringFilter(ginCtx, "keyword", &params.Keyword, &query.Keyword)
	bindAuditStringFilter(ginCtx, "actor", &params.Actor, &query.Actor)
	bindAuditStringFilter(ginCtx, "action", &params.Action, &query.Action)
	bindAuditStringFilter(ginCtx, "action_prefix", &params.ActionPrefix, &query.ActionPrefix)
	bindAuditStringFilter(ginCtx, "resource_type", &params.ResourceType, &query.ResourceType)
	bindAuditStringFilter(ginCtx, "resource_id", &params.ResourceId, &query.ResourceID)
	bindAuditStringFilter(ginCtx, "resource_name", &params.ResourceName, &query.ResourceName)
	bindAuditStringFilter(ginCtx, "session_id", &params.SessionId, &query.SessionID)
	bindAuditStringFilter(ginCtx, "request_id", &params.RequestId, &query.RequestID)
}

func bindAuditStringSliceFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) {
	if values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, "action_prefixes")); len(values) > 0 {
		params.ActionPrefixes = &values
		query.ActionPrefixes = values
	}
	if values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, "action_keywords")); len(values) > 0 {
		params.ActionKeywords = &values
		query.ActionKeywords = values
	}
	if values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, "resource_types")); len(values) > 0 {
		params.ResourceTypes = &values
		query.ResourceTypes = values
	}
	if values := normalizeAuditStringQuerySlice(queryArrayCompat(ginCtx, "request_path_prefixes")); len(values) > 0 {
		params.RequestPathPrefixes = &values
		query.RequestPathPrefixes = values
	}
}

func bindAuditEnumFilters(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if errField := bindAuditBusinessCategoryFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if errField := bindAuditSourceFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if errField := bindAuditResultFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if values, normalized, ok := bindAuditResultSliceFilter(queryArrayCompat(ginCtx, "results")); !ok {
		return "results"
	} else if len(values) > 0 {
		params.Results = &values
		query.Results = normalized
	}
	if errField := bindAuditRiskLevelFilter(ginCtx, params, query); errField != "" {
		return errField
	}
	if values, normalized, ok := bindAuditRiskLevelSliceFilter(queryArrayCompat(ginCtx, "risk_levels")); !ok {
		return "risk_levels"
	} else if len(values) > 0 {
		params.RiskLevels = &values
		query.RiskLevels = normalized
	}
	return ""
}

func bindAuditBusinessCategoryFilter(
	ginCtx *gin.Context,
	params *auditopenapi.GetAuditLogsParams,
	query *ListQuery,
) string {
	raw := strings.TrimSpace(ginCtx.Query("business_category"))
	if raw == "" {
		return ""
	}

	switch auditstore.AuditBusinessCategory(raw) {
	case auditstore.AuditBusinessCategoryFailedOperations,
		auditstore.AuditBusinessCategoryHighRiskOperations,
		auditstore.AuditBusinessCategorySensitiveOperations,
		auditstore.AuditBusinessCategoryAuthFailures,
		auditstore.AuditBusinessCategoryPermissionDenials,
		auditstore.AuditBusinessCategoryRBACChanges,
		auditstore.AuditBusinessCategoryCriticalSecurity:
		value := auditopenapi.GetAuditLogsParamsBusinessCategory(raw)
		params.BusinessCategory = &value
		query.BusinessCategory = auditstore.AuditBusinessCategory(raw)
		return ""
	default:
		return "business_category"
	}
}

func bindAuditSourceFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("source"))); raw != "" {
		switch auditstore.AuditSource(raw) {
		case auditstore.AuditSourceRequest, auditstore.AuditSourceSecurityEvent, auditstore.AuditSourceDomainEvent:
			value := auditopenapi.GetAuditLogsParamsSource(raw)
			params.Source = &value
			query.Source = auditstore.AuditSource(raw)
		default:
			return "source"
		}
	}

	return ""
}

func bindAuditResultFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("result"))); raw != "" {
		switch auditstore.AuditResult(raw) {
		case auditstore.AuditResultSuccess, auditstore.AuditResultFailed, auditstore.AuditResultDenied, auditstore.AuditResultError:
			value := auditopenapi.GetAuditLogsParamsResult(raw)
			params.Result = &value
			query.Result = auditstore.AuditResult(raw)
		default:
			return "result"
		}
	}

	return ""
}

func bindAuditRiskLevelFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	if raw := strings.ToUpper(strings.TrimSpace(ginCtx.Query("risk_level"))); raw != "" {
		switch auditstore.AuditRiskLevel(raw) {
		case auditstore.AuditRiskLevelLow, auditstore.AuditRiskLevelMedium, auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
			value := auditopenapi.GetAuditLogsParamsRiskLevel(raw)
			params.RiskLevel = &value
			query.RiskLevel = auditstore.AuditRiskLevel(raw)
		default:
			return "risk_level"
		}
	}

	return ""
}

func bindAuditResultSliceFilter(rawValues []string) ([]auditopenapi.GetAuditLogsParamsResults, []auditstore.AuditResult, bool) {
	normalized, ok := normalizeAuditEnumQuerySlice(rawValues, func(value string) bool {
		switch auditstore.AuditResult(value) {
		case auditstore.AuditResultSuccess, auditstore.AuditResultFailed, auditstore.AuditResultDenied, auditstore.AuditResultError:
			return true
		default:
			return false
		}
	})
	if !ok {
		return nil, nil, false
	}

	return collectAuditResultSlice(normalized), collectAuditStoreResultSlice(normalized), true
}

func bindAuditRiskLevelSliceFilter(rawValues []string) ([]auditopenapi.GetAuditLogsParamsRiskLevels, []auditstore.AuditRiskLevel, bool) {
	normalized, ok := normalizeAuditEnumQuerySlice(rawValues, func(value string) bool {
		switch auditstore.AuditRiskLevel(value) {
		case auditstore.AuditRiskLevelLow, auditstore.AuditRiskLevelMedium, auditstore.AuditRiskLevelHigh, auditstore.AuditRiskLevelCritical:
			return true
		default:
			return false
		}
	})
	if !ok {
		return nil, nil, false
	}

	return collectAuditRiskLevelSlice(normalized), collectAuditStoreRiskLevelSlice(normalized), true
}

func normalizeAuditEnumQuerySlice(rawValues []string, isAllowed func(string) bool) ([]string, bool) {
	if len(rawValues) == 0 {
		return nil, true
	}

	normalized := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		value := strings.ToUpper(strings.TrimSpace(raw))
		if !isAllowed(value) {
			return nil, false
		}
		normalized = append(normalized, value)
	}

	return normalized, true
}

func collectAuditResultSlice(values []string) []auditopenapi.GetAuditLogsParamsResults {
	collected := make([]auditopenapi.GetAuditLogsParamsResults, 0, len(values))
	for _, value := range values {
		collected = append(collected, auditopenapi.GetAuditLogsParamsResults(value))
	}
	return collected
}

func collectAuditStoreResultSlice(values []string) []auditstore.AuditResult {
	collected := make([]auditstore.AuditResult, 0, len(values))
	for _, value := range values {
		collected = append(collected, auditstore.AuditResult(value))
	}
	return collected
}

func collectAuditRiskLevelSlice(values []string) []auditopenapi.GetAuditLogsParamsRiskLevels {
	collected := make([]auditopenapi.GetAuditLogsParamsRiskLevels, 0, len(values))
	for _, value := range values {
		collected = append(collected, auditopenapi.GetAuditLogsParamsRiskLevels(value))
	}
	return collected
}

func collectAuditStoreRiskLevelSlice(values []string) []auditstore.AuditRiskLevel {
	collected := make([]auditstore.AuditRiskLevel, 0, len(values))
	for _, value := range values {
		collected = append(collected, auditstore.AuditRiskLevel(value))
	}
	return collected
}

func normalizeAuditStringQuerySlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func queryArrayCompat(ginCtx *gin.Context, key string) []string {
	values := ginCtx.QueryArray(key)
	bracketValues := ginCtx.QueryArray(key + "[]")
	if len(bracketValues) == 0 {
		return values
	}

	combined := make([]string, 0, len(values)+len(bracketValues))
	combined = append(combined, values...)
	combined = append(combined, bracketValues...)
	return combined
}

func bindAuditStringFilter(ginCtx *gin.Context, key string, targetParam **string, targetQuery *string) {
	if raw := strings.TrimSpace(ginCtx.Query(key)); raw != "" {
		*targetParam = &raw
		*targetQuery = raw
	}
}

func bindAuditSuccessFilter(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	value, ok, err := parseOptionalBoolQuery(ginCtx, "success")
	if err != nil {
		return "success"
	}
	if ok {
		params.Success = &value
		query.Success = &value
	}
	return ""
}

func bindAuditCreatedRange(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	createdFrom, ok, err := parseOptionalTimeQuery(ginCtx, "created_from")
	if err != nil {
		return "created_from"
	}
	if ok {
		converted := createdFrom.UTC()
		params.CreatedFrom = &converted
		query.CreatedFrom = &createdFrom
	}

	createdTo, ok, err := parseOptionalTimeQuery(ginCtx, "created_to")
	if err != nil {
		return "created_to"
	}
	if ok {
		converted := createdTo.UTC()
		params.CreatedTo = &converted
		query.CreatedTo = &createdTo
	}

	return ""
}

func bindAuditSort(ginCtx *gin.Context, params *auditopenapi.GetAuditLogsParams, query *ListQuery) string {
	rawValues := queryArrayCompat(ginCtx, "sort")
	if len(rawValues) == 0 {
		return ""
	}

	sorts := make([]string, 0, len(rawValues))
	for _, raw := range rawValues {
		field, order, ok := ParseAuditSortExpressionForBinding(raw)
		if !ok {
			return "sort"
		}
		sorts = append(sorts, field+":"+order)
	}
	params.Sort = &sorts
	query.Sorts = sorts
	return ""
}

func rejectUnknownAuditListQueryKeys(ginCtx *gin.Context) string {
	for key := range ginCtx.Request.URL.Query() {
		if _, ok := auditAllowedListQueryKeys[key]; !ok {
			return key
		}
	}
	return ""
}

func parseOptionalIntQuery(ginCtx *gin.Context, key string) (int, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return 0, false, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func parseOptionalUint64Query(ginCtx *gin.Context, key string) (uint64, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return 0, false, nil
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func parseOptionalBoolQuery(ginCtx *gin.Context, key string) (bool, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return false, false, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false, err
	}
	return value, true, nil
}

func parseOptionalTimeQuery(ginCtx *gin.Context, key string) (time.Time, bool, error) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return time.Time{}, false, nil
	}

	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false, err
	}
	return value, true, nil
}

func bindGeneratedAuditReadHeaders(ginCtx *gin.Context) (locale *string, requestID *string) {
	if raw := strings.TrimSpace(ginCtx.GetHeader(httpx.RequestIDHeader)); raw != "" {
		requestID = &raw
	}
	if raw := strings.TrimSpace(ginCtx.GetHeader(string(httpheader.Locale))); raw != "" {
		locale = &raw
	}

	return locale, requestID
}

func bindGeneratedAuditOverviewParams(ginCtx *gin.Context) auditopenapi.GetAuditOverviewParams {
	locale, requestID := bindGeneratedAuditReadHeaders(ginCtx)
	params := auditopenapi.GetAuditOverviewParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}

	if raw := strings.TrimSpace(ginCtx.Query("preset")); raw != "" {
		value := auditopenapi.GetAuditOverviewParamsPreset(raw)
		if value.Valid() {
			params.Preset = &value
		}
	}

	return params
}

func normalizeAuditOverviewPreset(value *auditopenapi.GetAuditOverviewParamsPreset) auditstore.AuditTimePreset {
	if value == nil {
		return auditstore.AuditTimePresetLast24Hours
	}
	switch strings.TrimSpace(string(*value)) {
	case string(auditstore.AuditTimePresetLast7Days):
		return auditstore.AuditTimePresetLast7Days
	case string(auditstore.AuditTimePresetLast30Days):
		return auditstore.AuditTimePresetLast30Days
	default:
		return auditstore.AuditTimePresetLast24Hours
	}
}
