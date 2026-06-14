// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/httpheader"
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
	return nil
}

func (r routeRuntime) handleList(ginCtx *gin.Context) {
	_ = bindGetContainersParams(ginCtx)
	runtime, items, err := r.service.List(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toContainerListResponse(runtime, items))
}

func (r routeRuntime) handleDetail(ginCtx *gin.Context) {
	_ = bindGetContainerParams(ginCtx)
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	detail, err := r.service.Detail(ginCtx.Request.Context(), ref)
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

func (r routeRuntime) handleStart(ginCtx *gin.Context) {
	r.handleAction(ginCtx, r.service.Start)
}

func (r routeRuntime) handleStop(ginCtx *gin.Context) {
	r.handleAction(ginCtx, r.service.Stop)
}

func (r routeRuntime) handleRestart(ginCtx *gin.Context) {
	r.handleAction(ginCtx, r.service.Restart)
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

func bindGetContainersParams(ginCtx *gin.Context) containeropenapi.GetContainersParams {
	locale, requestID := commonHeaders(ginCtx)
	return containeropenapi.GetContainersParams{XGraftLocale: locale, XRequestId: requestID}
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
