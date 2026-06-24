package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	containeropenapi "graft/server/internal/contract/openapi/container"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

type routeRuntime struct {
	ctx         *module.Context
	service     *service
	userService moduleapi.UserService
}

// RegisterRoutes registers HTTP API endpoints for container operations with permission-based access control.
// registerRoutes 注册容器管理路由，包括权限中间件和审计日志发布。若服务不可用或依赖项解析失败则返回错误，否则返回 nil。
func registerRoutes(ctx *module.Context, moduleName string, service *service) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	if service == nil {
		return errors.New("container service is unavailable")
	}
	authService, err := resolveAuthService(ctx)
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return fmt.Errorf("resolve authorizer: %w", err)
	}
	userService, err := resolveUserService(ctx)
	if err != nil {
		return fmt.Errorf("resolve user service: %w", err)
	}

	routes := routeRuntime{ctx: ctx, service: service, userService: userService}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleName)
	group := ctx.Router.Group(containercontract.ContainerAPIGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(
		containercontract.ContainerCollectionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerViewPermission.String(), publisher),
		routes.handleList,
	)
	group.GET(
		containercontract.ContainerDashboardSummaryRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerViewPermission.String(), publisher),
		routes.handleDashboardSummary,
	)
	group.GET(
		containercontract.ContainerDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerDetailPermission.String(), publisher),
		routes.handleDetail,
	)
	group.GET(
		containercontract.ContainerLogsRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerLogsPermission.String(), publisher),
		routes.handleLogs,
	)
	group.POST(
		containercontract.ContainerShellSessionsRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerShellPermission.String(), publisher),
		routes.handleShellSessionCreate,
	)
	group.GET(
		containercontract.ContainerShellWebSocketRoute,
		routes.handleShellWebSocket,
	)
	group.GET(
		containercontract.ContainerMountUsageRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerDetailPermission.String(), publisher),
		routes.handleMountUsageList,
	)
	group.POST(
		containercontract.ContainerMountUsageRefreshRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerDetailPermission.String(), publisher),
		routes.handleMountUsageRefresh,
	)
	group.POST(
		containercontract.ContainerStartRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerStartPermission.String(), publisher),
		routes.handleStart,
	)
	group.POST(
		containercontract.ContainerStopRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerStopPermission.String(), publisher),
		routes.handleStop,
	)
	group.POST(
		containercontract.ContainerRestartRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerRestartPermission.String(), publisher),
		routes.handleRestart,
	)
	group.POST(
		containercontract.ContainerRemoveRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerRemovePermission.String(), publisher),
		routes.handleRemove,
	)
	group.POST(
		containercontract.ContainerBatchActionsRoute,
		// 批量操作路由先通过认证中间件建立请求身份，再由 handler 按 action 分派精确权限。
		httpx.RequirePermission(ctx.I18n, authService, authorizer, "", publisher),
		routes.handleBatchAction,
	)
	return nil
}

func (r routeRuntime) handleList(ginCtx *gin.Context) {
	params, ok := bindGetContainersParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	result, err := r.service.List(ginCtx.Request.Context(), listQueryFromParams(params))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toContainerListResponse(result))
}

func (r routeRuntime) handleDashboardSummary(ginCtx *gin.Context) {
	result, err := r.service.DashboardSummary(ginCtx.Request.Context(), dashboardSummaryQuery{})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toContainerDashboardSummaryResponse(result))
}

func (r routeRuntime) handleDetail(ginCtx *gin.Context) {
	// Keep generated binding on routes with OpenAPI header parameters even when the handler does not read them.
	_ = bindGetContainerParams(ginCtx)
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	requestCtx := ginCtx.Request.Context()
	if r.authorizeEnvironmentPlainAccess(ginCtx) {
		requestCtx = withEnvironmentPlainAccess(requestCtx)
	}
	detail, err := r.service.Detail(requestCtx, ref)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toDetail(detail))
}

func (r routeRuntime) handleLogs(ginCtx *gin.Context) {
	params := bindGetContainerLogsParams(ginCtx)
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	logs, err := r.service.Logs(ginCtx.Request.Context(), ref, LogQuery{
		Tail:       intValue(params.Tail),
		Since:      stringPtrValue(params.Since),
		Timestamps: boolPtrValue(params.Timestamps),
		Stdout:     boolPtrValue(params.Stdout),
		Stderr:     boolPtrValue(params.Stderr),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toLogs(logs))
}

func (r routeRuntime) handleShellSessionCreate(ginCtx *gin.Context) {
	_ = bindPostContainerShellSessionParams(ginCtx)
	var request containeropenapi.PostContainerShellSessionJSONRequestBody
	if !bindRequiredJSON(ginCtx, r, &request) {
		return
	}
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	session, err := r.service.IssueShellSession(ginCtx.Request.Context(), ref, ShellSessionRequest{
		Command: string(request.Command),
		Cols:    request.Cols,
		Rows:    request.Rows,
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toShellSession(session))
}

func (r routeRuntime) handleMountUsageList(ginCtx *gin.Context) {
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	items, err := r.service.MountUsageList(ginCtx.Request.Context(), ref)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toMountUsageList(items))
}

func (r routeRuntime) handleMountUsageRefresh(ginCtx *gin.Context) {
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	mountID := strings.TrimSpace(ginCtx.Param("mountId"))
	usage, err := r.service.RefreshMountUsage(ginCtx.Request.Context(), ref, mountID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toMountUsage(usage))
}

func (r routeRuntime) handleStart(ginCtx *gin.Context) {
	r.handleAction(ginCtx, r.service.Start)
}

func (r routeRuntime) handleStop(ginCtx *gin.Context) {
	r.handleAction(ginCtx, r.service.Stop)
}

func (r routeRuntime) handleRestart(ginCtx *gin.Context) {
	r.handleAction(ginCtx, r.service.Restart)
}

func (r routeRuntime) handleRemove(ginCtx *gin.Context) {
	_ = bindPostContainerRemoveParams(ginCtx)
	var request containeropenapi.PostContainerRemoveJSONRequestBody
	if !bindOptionalJSON(ginCtx, r, &request) {
		return
	}
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	result, err := r.service.Remove(ginCtx.Request.Context(), ref, RemoveOptions{Force: boolPtrValue(request.Force)})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toContainerAction(result))
}

func (r routeRuntime) handleBatchAction(ginCtx *gin.Context) {
	_ = bindPostContainerBatchActionsParams(ginCtx)
	var request containeropenapi.PostContainerBatchActionsJSONRequestBody
	if !bindRequiredJSON(ginCtx, r, &request) {
		return
	}
	if !request.Action.Valid() {
		r.writeRouteError(ginCtx, errInvalidBatchAction)
		return
	}
	if !r.authorizeBatchAction(ginCtx, string(request.Action)) {
		return
	}
	result, err := r.service.BatchAction(ginCtx.Request.Context(), BatchActionCommand{
		Action: string(request.Action),
		IDs:    request.Ids,
		Force:  boolPtrValue(request.Force),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toContainerBatchAction(result))
}

func (r routeRuntime) authorizeBatchAction(ginCtx *gin.Context, action string) bool {
	permission := permissionForAction(action)
	if permission == "" {
		r.writeRouteError(ginCtx, errInvalidBatchAction)
		return false
	}
	authorizer, err := resolveAuthorizer(r.ctx)
	if err != nil {
		httpx.WriteLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return false
	}
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok {
		httpx.WriteLocalizedError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return false
	}
	if err := authorizer.Authorize(ginCtx.Request.Context(), requestAuth, permission); err != nil {
		httpx.WriteLocalizedError(ginCtx, r.ctx.I18n, http.StatusForbidden, messagecontract.AuthForbidden.String(), map[string]any{
			"permission": permission,
		})
		return false
	}
	return true
}

func (r routeRuntime) authorizeEnvironmentPlainAccess(ginCtx *gin.Context) bool {
	authorizer, err := resolveAuthorizer(r.ctx)
	if err != nil {
		return false
	}
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok {
		return false
	}
	return authorizer.Authorize(
		ginCtx.Request.Context(),
		requestAuth,
		containercontract.ContainerEnvironmentPermission.String(),
	) == nil
}

func permissionForAction(action string) string {
	switch action {
	case containerActionStart:
		return containercontract.ContainerStartPermission.String()
	case containerActionStop:
		return containercontract.ContainerStopPermission.String()
	case containerActionRestart:
		return containercontract.ContainerRestartPermission.String()
	case containerActionRemove:
		return containercontract.ContainerRemovePermission.String()
	default:
		return ""
	}
}

func (r routeRuntime) handleAction(ginCtx *gin.Context, action func(context.Context, Ref) (ActionResult, error)) {
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	result, err := action(ginCtx.Request.Context(), ref)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toContainerAction(result))
}

func readRef(ginCtx *gin.Context, r routeRuntime) (Ref, bool) {
	ref, err := parseRef(ginCtx.Param("id"))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return Ref{}, false
	}
	return ref, true
}

func (r routeRuntime) writeRouteError(ginCtx *gin.Context, err error) {
	httpx.WriteLocalizedError(ginCtx, r.ctx.I18n, statusForError(err), messageKeyForError(err).String(), nil)
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

// ResolveAuthorizer 从服务容器中解析权限授权器，并验证其实现了 Authorizer 接口。解析失败或类型不匹配时返回错误。
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

// ResolveUserService resolves a UserService from the service container.
// It returns an error if resolution fails or the resolved type does not implement UserService.
func resolveUserService(ctx *module.Context) (moduleapi.UserService, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.UserService)(nil))
	if err != nil {
		return nil, err
	}
	service, ok := resolved.(moduleapi.UserService)
	if !ok {
		return nil, fmt.Errorf("resolved user service has unexpected type %T", resolved)
	}
	return service, nil
}

// bindGetContainersParams 绑定并校验容器列表请求的查询参数与请求头。
// 校验 limit、offset、keyword、state 和 health 查询参数的有效性。
// bindGetContainersParams 从请求中解析并校验容器列表查询参数，包括分页、关键词和筛选条件。校验失败时返回 false。
func bindGetContainersParams(ginCtx *gin.Context, ctx *module.Context) (containeropenapi.GetContainersParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	params := containeropenapi.GetContainersParams{XGraftLocale: locale, XRequestId: requestID}
	limit, ok := queryBoundedInt(ginCtx, ctx, "limit", 1, maxContainerListLimit)
	if !ok {
		return containeropenapi.GetContainersParams{}, false
	}
	params.Limit = limit
	offset, ok := queryBoundedInt(ginCtx, ctx, "offset", 0, 0)
	if !ok {
		return containeropenapi.GetContainersParams{}, false
	}
	params.Offset = offset
	if value := strings.TrimSpace(ginCtx.Query("keyword")); value != "" {
		if len(value) > containerListKeywordMaxLength {
			writeInvalidContainerQuery(ginCtx, ctx, "keyword")
			return containeropenapi.GetContainersParams{}, false
		}
		params.Keyword = &value
	}
	if !bindContainerListStateFilters(ginCtx, ctx, &params) {
		return containeropenapi.GetContainersParams{}, false
	}
	return params, true
}

// bindContainerListStateFilters validates and binds state, health, and orchestrator filters for container list queries, and validates source scope consistency. It returns true if all validations succeed, false otherwise.
func bindContainerListStateFilters(
	ginCtx *gin.Context,
	ctx *module.Context,
	params *containeropenapi.GetContainersParams,
) bool {
	state, ok := optionalEnumQueryValue(ginCtx, ctx, "state", isValidContainerState)
	if !ok {
		return false
	}
	if state != "" {
		value := containeropenapi.GetContainersParamsState(state)
		params.State = &value
	}

	health, ok := optionalEnumQueryValue(ginCtx, ctx, "health", isValidContainerHealth)
	if !ok {
		return false
	}
	if health != "" {
		value := containeropenapi.GetContainersParamsHealth(health)
		params.Health = &value
	}

	orchestrator, ok := optionalEnumQueryValue(ginCtx, ctx, "orchestrator", isValidContainerOrchestrator)
	if !ok {
		return false
	}
	if orchestrator != "" {
		value := containeropenapi.GetContainersParamsOrchestrator(orchestrator)
		params.Orchestrator = &value
	}
	if !bindContainerListSourceScopeFilters(ginCtx, ctx, params, orchestrator) {
		return false
	}
	return true
}

// bindContainerListSourceScopeFilters validates and binds the source_scope_kind and source_scope query parameters.
// Both parameters must be provided together and source_scope_kind must be compatible with the given orchestrator.
// Returns true if validation succeeds, false otherwise.
func bindContainerListSourceScopeFilters(
	ginCtx *gin.Context,
	ctx *module.Context,
	params *containeropenapi.GetContainersParams,
	orchestrator string,
) bool {
	sourceScopeKind, ok := optionalEnumQueryValue(ginCtx, ctx, "source_scope_kind", isValidContainerSourceScopeKind)
	if !ok {
		return false
	}
	sourceScope := strings.TrimSpace(ginCtx.Query("source_scope"))
	if (sourceScopeKind == "") != (sourceScope == "") {
		writeInvalidContainerQuery(ginCtx, ctx, "source_scope")
		return false
	}
	if sourceScopeKind == "" {
		return true
	}
	if !sourceScopeKindCompatibleWithOrchestrator(orchestrator, sourceScopeKind) {
		writeInvalidContainerQuery(ginCtx, ctx, "source_scope_kind")
		return false
	}
	value := containeropenapi.GetContainersParamsSourceScopeKind(sourceScopeKind)
	params.SourceScopeKind = &value
	params.SourceScope = &sourceScope
	return true
}

// optionalEnumQueryValue 读取并验证一个可选的枚举类查询参数。
// 参数不存在或为空时返回空字符串和 true；参数存在但验证失败时返回空字符串和 false；
// 参数有效时返回参数值和 true。
func optionalEnumQueryValue(
	ginCtx *gin.Context,
	ctx *module.Context,
	key string,
	valid func(string) bool,
) (string, bool) {
	value := strings.TrimSpace(ginCtx.Query(key))
	if value == "" {
		return "", true
	}
	if !valid(value) {
		writeInvalidContainerQuery(ginCtx, ctx, key)
		return "", false
	}
	return value, true
}

// BindGetContainerParams extracts common header values and returns them as an OpenAPI GetContainerParams object.
func bindGetContainerParams(ginCtx *gin.Context) containeropenapi.GetContainerParams {
	locale, requestID := commonHeaders(ginCtx)
	return containeropenapi.GetContainerParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetContainerLogsParams(ginCtx *gin.Context) containeropenapi.GetContainerLogsParams {
	locale, requestID := commonHeaders(ginCtx)
	params := containeropenapi.GetContainerLogsParams{XGraftLocale: locale, XRequestId: requestID}
	if value, ok := queryInt(ginCtx, "tail"); ok {
		params.Tail = &value
	}
	if value := ginCtx.Query("since"); value != "" {
		params.Since = &value
	}
	if value, ok := queryBool(ginCtx, "timestamps"); ok {
		params.Timestamps = &value
	}
	if value, ok := queryBool(ginCtx, "stdout"); ok {
		params.Stdout = &value
	}
	if value, ok := queryBool(ginCtx, "stderr"); ok {
		params.Stderr = &value
	}
	return params
}

func bindPostContainerRemoveParams(ginCtx *gin.Context) containeropenapi.PostContainerRemoveParams {
	locale, requestID := commonHeaders(ginCtx)
	return containeropenapi.PostContainerRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}

// BindPostContainerBatchActionsParams extracts locale and request ID headers from the HTTP request.
func bindPostContainerBatchActionsParams(ginCtx *gin.Context) containeropenapi.PostContainerBatchActionsParams {
	locale, requestID := commonHeaders(ginCtx)
	return containeropenapi.PostContainerBatchActionsParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostContainerShellSessionParams extracts locale and request ID headers for shell session parameters.
func bindPostContainerShellSessionParams(ginCtx *gin.Context) containeropenapi.PostContainerShellSessionParams {
	locale, requestID := commonHeaders(ginCtx)
	return containeropenapi.PostContainerShellSessionParams{XGraftLocale: locale, XRequestId: requestID}
}

// BindGetContainerShellWebSocketParams extracts the WebSocket ticket from the query string and request ID from headers for shell endpoint access.
func bindGetContainerShellWebSocketParams(ginCtx *gin.Context) containeropenapi.GetContainerShellWebSocketParams {
	requestID := optionalHeader(ginCtx, httpx.RequestIDHeader)
	return containeropenapi.GetContainerShellWebSocketParams{
		Ticket:     strings.TrimSpace(ginCtx.Query("ticket")),
		XRequestId: requestID,
	}
}

// BindRequiredJSON binds JSON from the request body to target. It returns true on success and false on binding failure, in which case it writes a localized error response.
func bindRequiredJSON(ginCtx *gin.Context, r routeRuntime, target any) bool {
	if err := ginCtx.ShouldBindJSON(target); err != nil {
		r.writeRouteError(ginCtx, errInvalidListQuery)
		return false
	}
	return true
}

func bindOptionalJSON(ginCtx *gin.Context, r routeRuntime, target any) bool {
	if ginCtx.Request == nil || ginCtx.Request.Body == nil {
		return true
	}
	if err := ginCtx.ShouldBindJSON(target); err != nil && !errors.Is(err, io.EOF) {
		r.writeRouteError(ginCtx, errInvalidListQuery)
		return false
	}
	return true
}

func commonHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := optionalHeader(ginCtx, string(httpheader.Locale))
	requestID := optionalHeader(ginCtx, httpx.RequestIDHeader)
	return locale, requestID
}

func optionalHeader(ginCtx *gin.Context, name string) *string {
	value := ginCtx.GetHeader(name)
	if value == "" {
		return nil
	}
	return &value
}

func queryInt(ginCtx *gin.Context, key string) (int, bool) {
	value := ginCtx.Query(key)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func queryBoundedInt(ginCtx *gin.Context, ctx *module.Context, key string, min int, max int) (*int, bool) {
	raw := strings.TrimSpace(ginCtx.Query(key))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || (max > 0 && value > max) {
		writeInvalidContainerQuery(ginCtx, ctx, key)
		return nil, false
	}
	return &value, true
}

func writeInvalidContainerQuery(ginCtx *gin.Context, ctx *module.Context, field string) {
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": field,
	})
}

// listQueryFromParams converts OpenAPI container list parameters into an internal ListQuery.
func listQueryFromParams(params containeropenapi.GetContainersParams) ListQuery {
	query := ListQuery{
		Limit:   intValue(params.Limit),
		Offset:  intValue(params.Offset),
		Keyword: stringPtrValue(params.Keyword),
	}
	if params.State != nil {
		query.State = string(*params.State)
	}
	if params.Health != nil {
		query.Health = string(*params.Health)
	}
	if params.Orchestrator != nil {
		query.Orchestrator = string(*params.Orchestrator)
	}
	if params.SourceScopeKind != nil {
		query.SourceScopeKind = string(*params.SourceScopeKind)
	}
	if params.SourceScope != nil {
		query.SourceScope = *params.SourceScope
	}
	return query
}

func queryBool(ginCtx *gin.Context, key string) (bool, bool) {
	value := ginCtx.Query(key)
	if value == "" {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}
	return parsed, true
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolPtrValue(value *bool) bool {
	return value != nil && *value
}
