// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
	ctx     *module.Context
	service *service
}

// RegisterRoutes registers HTTP API endpoints for container operations with permission-based access control.
// It returns an error if the service is unavailable, or if resolving auth service or authorizer fails; nil otherwise.
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

	routes := routeRuntime{ctx: ctx, service: service}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleName)
	group := ctx.Router.Group(containercontract.ContainerAPIGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(
		containercontract.ContainerCollectionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, containercontract.ContainerViewPermission.String(), publisher),
		routes.handleList,
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
	if value := strings.TrimSpace(ginCtx.Query("state")); value != "" {
		if !isValidContainerState(value) {
			writeInvalidContainerQuery(ginCtx, ctx, "state")
			return containeropenapi.GetContainersParams{}, false
		}
		state := containeropenapi.GetContainersParamsState(value)
		params.State = &state
	}
	if value := strings.TrimSpace(ginCtx.Query("health")); value != "" {
		if !isValidContainerHealth(value) {
			writeInvalidContainerQuery(ginCtx, ctx, "health")
			return containeropenapi.GetContainersParams{}, false
		}
		health := containeropenapi.GetContainersParamsHealth(value)
		params.Health = &health
	}
	return params, true
}

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

func bindPostContainerBatchActionsParams(ginCtx *gin.Context) containeropenapi.PostContainerBatchActionsParams {
	locale, requestID := commonHeaders(ginCtx)
	return containeropenapi.PostContainerBatchActionsParams{XGraftLocale: locale, XRequestId: requestID}
}

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
