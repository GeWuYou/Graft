package scheduler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	scheduleropenapi "graft/server/internal/contract/openapi/scheduler"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	schedulercore "graft/server/internal/scheduler"
	schedulercontract "graft/server/modules/scheduler/contract"
)

const (
	defaultScheduledTaskRunListLimit = 20
	maxScheduledTaskRunListLimit     = 100
)

type schedulerRouteRuntime struct {
	ctx        *module.Context
	moduleName string
	runtime    schedulercore.Runtime
}

func registerSchedulerRoutesWithSecurity(
	ctx *module.Context,
	moduleName string,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	runtime, err := resolveSchedulerRuntime(ctx)
	if err != nil {
		return err
	}
	if authService == nil {
		authService, err = resolveAuthService(ctx)
		if err != nil {
			return err
		}
	}
	if authorizer == nil {
		authorizer, err = resolveAuthorizer(ctx)
		if err != nil {
			return err
		}
	}

	routeRuntime := schedulerRouteRuntime{
		ctx:        ctx,
		moduleName: moduleName,
		runtime:    runtime,
	}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleName)
	group := ctx.Router.Group(schedulercontract.ScheduledTasksGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(
		schedulercontract.ScheduledTaskCollectionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		routeRuntime.handleListTasks,
	)
	group.GET(
		schedulercontract.ScheduledTaskDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		routeRuntime.handleGetTask,
	)
	group.GET(
		schedulercontract.ScheduledTaskRunsRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		routeRuntime.handleListRuns,
	)
	group.POST(
		schedulercontract.ScheduledTaskRunRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskRunPermission.String(), publisher),
		routeRuntime.handleRunOnce,
	)

	return nil
}

func resolveSchedulerRuntime(ctx *module.Context) (schedulercore.Runtime, error) {
	resolved, err := ctx.Services.Resolve((*schedulercore.Runtime)(nil))
	if err != nil {
		return nil, err
	}
	runtime, ok := resolved.(schedulercore.Runtime)
	if !ok || runtime == nil {
		return nil, errors.New("scheduler runtime service has unexpected type")
	}
	return runtime, nil
}

func (r schedulerRouteRuntime) handleListTasks(ginCtx *gin.Context) {
	schedulerGeneratedHandler{}.GetScheduledTasks(bindGeneratedTaskHeaders(ginCtx))

	tasks, err := r.runtime.ListTasks(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, "list scheduled tasks failed", err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskListResponse(tasks))
}

func (r schedulerRouteRuntime) handleGetTask(ginCtx *gin.Context) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.GetScheduledTask(key, bindGeneratedTaskDetailHeaders(ginCtx))

	task, err := r.runtime.GetTask(ginCtx.Request.Context(), key)
	if err != nil {
		r.writeRouteError(ginCtx, "read scheduled task failed", err, zap.String("taskKey", key))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskItem(task))
}

func (r schedulerRouteRuntime) handleListRuns(ginCtx *gin.Context) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	params, ok := bindGeneratedRunListParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.GetScheduledTaskRuns(key, params)

	limit, offset := normalizedRunListWindow(params)
	result, err := r.runtime.ListRuns(ginCtx.Request.Context(), schedulercore.RunListQuery{
		TaskKey: key,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		r.writeRouteError(ginCtx, "list scheduled task runs failed", err, zap.String("taskKey", key))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskRunListResponse(result, limit, offset))
}

func (r schedulerRouteRuntime) handleRunOnce(ginCtx *gin.Context) {
	key, ok := readScheduledTaskRunKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.PostScheduledTaskRun(key, bindGeneratedTaskRunHeaders(ginCtx))

	run, err := r.runtime.RunOnce(ginCtx.Request.Context(), key)
	if err != nil {
		r.writeRouteError(ginCtx, "run scheduled task once failed", err, zap.String("taskKey", key))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskRunItem(run))
}

func (r schedulerRouteRuntime) writeRouteError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	switch {
	case errors.Is(err, schedulercore.ErrTaskNotFound):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, schedulercontract.ScheduledTaskNotFound.String(), nil)
	case errors.Is(err, schedulercore.ErrTaskAlreadyRunning):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusConflict, schedulercontract.ScheduledTaskAlreadyRunning.String(), nil)
	default:
		if r.ctx != nil && r.ctx.Logger != nil {
			logFields := append([]zap.Field{zap.String("module", r.moduleName), zap.Error(err)}, fields...)
			r.ctx.Logger.Error(message, logFields...)
		}
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	}
}

func readScheduledTaskKey(ginCtx *gin.Context, ctx *module.Context) (string, bool) {
	key := strings.TrimSpace(ginCtx.Param("taskKey"))
	if key == "" {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": "taskKey",
		})
		return "", false
	}
	return key, true
}

func readScheduledTaskRunKey(ginCtx *gin.Context, ctx *module.Context) (string, bool) {
	key := strings.TrimSpace(ginCtx.Param("taskKey"))
	if key == "" {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": "taskKey",
		})
		return "", false
	}
	return key, true
}

func bindGeneratedTaskHeaders(ginCtx *gin.Context) scheduleropenapi.GetScheduledTasksParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.GetScheduledTasksParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskDetailHeaders(ginCtx *gin.Context) scheduleropenapi.GetScheduledTaskParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.GetScheduledTaskParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskRunHeaders(ginCtx *gin.Context) scheduleropenapi.PostScheduledTaskRunParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.PostScheduledTaskRunParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRunListParams(ginCtx *gin.Context, ctx *module.Context) (scheduleropenapi.GetScheduledTaskRunsParams, bool) {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	params := scheduleropenapi.GetScheduledTaskRunsParams{XGraftLocale: locale, XRequestId: requestID}

	if raw := strings.TrimSpace(ginCtx.Query("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxScheduledTaskRunListLimit {
			writeInvalidSchedulerQuery(ginCtx, ctx, "limit")
			return scheduleropenapi.GetScheduledTaskRunsParams{}, false
		}
		params.Limit = &limit
	}
	if raw := strings.TrimSpace(ginCtx.Query("offset")); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			writeInvalidSchedulerQuery(ginCtx, ctx, "offset")
			return scheduleropenapi.GetScheduledTaskRunsParams{}, false
		}
		params.Offset = &offset
	}

	return params, true
}

func bindGeneratedSchedulerHeaders(ginCtx *gin.Context) (*string, *string) {
	var locale *string
	if raw := strings.TrimSpace(ginCtx.GetHeader(string(httpheader.Locale))); raw != "" {
		locale = &raw
	}

	var requestID *string
	if raw := strings.TrimSpace(ginCtx.GetHeader(httpx.RequestIDHeader)); raw != "" {
		requestID = &raw
	}

	return locale, requestID
}

func normalizedRunListWindow(params scheduleropenapi.GetScheduledTaskRunsParams) (int, int) {
	limit := defaultScheduledTaskRunListLimit
	if params.Limit != nil {
		limit = *params.Limit
	}
	offset := 0
	if params.Offset != nil {
		offset = *params.Offset
	}
	return limit, offset
}

func writeInvalidSchedulerQuery(ginCtx *gin.Context, ctx *module.Context, field string) {
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": field,
	})
}

type schedulerGeneratedHandler struct{}

func (schedulerGeneratedHandler) GetScheduledTasks(params scheduleropenapi.GetScheduledTasksParams) {
	_ = params
}

func (schedulerGeneratedHandler) GetScheduledTask(key string, params scheduleropenapi.GetScheduledTaskParams) {
	_ = key
	_ = params
}

func (schedulerGeneratedHandler) GetScheduledTaskRuns(key string, params scheduleropenapi.GetScheduledTaskRunsParams) {
	_ = key
	_ = params
}

func (schedulerGeneratedHandler) PostScheduledTaskRun(key string, params scheduleropenapi.PostScheduledTaskRunParams) {
	_ = key
	_ = params
}
