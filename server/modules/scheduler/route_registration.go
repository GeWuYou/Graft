package scheduler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	scheduleropenapi "graft/server/internal/contract/openapi/scheduler"
	"graft/server/internal/cronx"
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
	group.POST(
		schedulercontract.ScheduledTaskCollectionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskCreatePermission.String(), publisher),
		routeRuntime.handleCreateTask,
	)
	group.GET(
		schedulercontract.ScheduledTaskRunDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		routeRuntime.handleGetRun,
	)
	group.GET(
		schedulercontract.ScheduledTaskDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		routeRuntime.handleGetTask,
	)
	group.PUT(
		schedulercontract.ScheduledTaskDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskUpdatePermission.String(), publisher),
		routeRuntime.handleUpdateTask,
	)
	group.DELETE(
		schedulercontract.ScheduledTaskDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskDeletePermission.String(), publisher),
		routeRuntime.handleDeleteTask,
	)
	group.POST(
		schedulercontract.ScheduledTaskEnableRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskEnablePermission.String(), publisher),
		routeRuntime.handleEnableTask,
	)
	group.POST(
		schedulercontract.ScheduledTaskDisableRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskEnablePermission.String(), publisher),
		routeRuntime.handleDisableTask,
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

func (r schedulerRouteRuntime) handleCreateTask(ginCtx *gin.Context) {
	var request scheduleropenapi.PostScheduledTaskJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		writeInvalidSchedulerField(ginCtx, r.ctx, "body")
		return
	}
	schedulerGeneratedHandler{}.PostScheduledTask(bindGeneratedTaskCreateHeaders(ginCtx), request)

	command, ok := createTaskMutation(request)
	if !ok {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), map[string]any{
			"field": "task_type",
		})
		return
	}

	task, err := r.runtime.CreateTask(ginCtx.Request.Context(), command)
	if err != nil {
		r.writeRouteError(ginCtx, "create scheduled task failed", err, zap.String("taskKey", command.TaskKey))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskItem(task))
}

func (r schedulerRouteRuntime) handleUpdateTask(ginCtx *gin.Context) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	var request scheduleropenapi.PutScheduledTaskJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		writeInvalidSchedulerField(ginCtx, r.ctx, "body")
		return
	}
	schedulerGeneratedHandler{}.PutScheduledTask(key, bindGeneratedTaskUpdateHeaders(ginCtx), request)

	command, err := updateTaskMutation(request)
	if err != nil {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), map[string]any{
			"field": err.Error(),
		})
		return
	}
	task, err := r.runtime.UpdateTask(ginCtx.Request.Context(), key, command)
	if err != nil {
		r.writeRouteError(ginCtx, "update scheduled task failed", err, zap.String("taskKey", key))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskItem(task))
}

func (r schedulerRouteRuntime) handleDeleteTask(ginCtx *gin.Context) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.DeleteScheduledTask(key, bindGeneratedTaskDeleteHeaders(ginCtx))

	if err := r.runtime.DeleteTask(ginCtx.Request.Context(), key); err != nil {
		r.writeRouteError(ginCtx, "delete scheduled task failed", err, zap.String("taskKey", key))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, map[string]any{})
}

func (r schedulerRouteRuntime) handleEnableTask(ginCtx *gin.Context) {
	r.handleSetTaskEnabled(ginCtx, true)
}

func (r schedulerRouteRuntime) handleDisableTask(ginCtx *gin.Context) {
	r.handleSetTaskEnabled(ginCtx, false)
}

func (r schedulerRouteRuntime) handleSetTaskEnabled(ginCtx *gin.Context, enabled bool) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	if enabled {
		schedulerGeneratedHandler{}.PostScheduledTaskEnable(key, bindGeneratedTaskEnableHeaders(ginCtx))
	} else {
		schedulerGeneratedHandler{}.PostScheduledTaskDisable(key, bindGeneratedTaskDisableHeaders(ginCtx))
	}

	task, err := r.runtime.SetTaskEnabled(ginCtx.Request.Context(), key, enabled)
	if err != nil {
		r.writeRouteError(ginCtx, "set scheduled task enabled failed", err, zap.String("taskKey", key), zap.Bool("enabled", enabled))
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

func (r schedulerRouteRuntime) handleGetRun(ginCtx *gin.Context) {
	runID, ok := readScheduledTaskRunID(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.GetScheduledTaskRun(runID, bindGeneratedTaskRunDetailHeaders(ginCtx))

	run, err := r.runtime.GetRun(ginCtx.Request.Context(), runID)
	if err != nil {
		r.writeRouteError(ginCtx, "read scheduled task run failed", err, zap.Uint64("runID", runID))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskRunItem(run))
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
	case errors.Is(err, schedulercore.ErrTaskImmutable), errors.Is(err, schedulercore.ErrTaskValidation):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), nil)
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

func readScheduledTaskRunID(ginCtx *gin.Context, ctx *module.Context) (uint64, bool) {
	raw := strings.TrimSpace(ginCtx.Param("runID"))
	if raw == "" {
		raw = strings.TrimSpace(ginCtx.Param("runId"))
	}
	runID, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || runID == 0 {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": "runId",
		})
		return 0, false
	}
	return runID, true
}

func bindGeneratedTaskHeaders(ginCtx *gin.Context) scheduleropenapi.GetScheduledTasksParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.GetScheduledTasksParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskCreateHeaders(ginCtx *gin.Context) scheduleropenapi.PostScheduledTaskParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.PostScheduledTaskParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskDetailHeaders(ginCtx *gin.Context) scheduleropenapi.GetScheduledTaskParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.GetScheduledTaskParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskUpdateHeaders(ginCtx *gin.Context) scheduleropenapi.PutScheduledTaskParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.PutScheduledTaskParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskDeleteHeaders(ginCtx *gin.Context) scheduleropenapi.DeleteScheduledTaskParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.DeleteScheduledTaskParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskEnableHeaders(ginCtx *gin.Context) scheduleropenapi.PostScheduledTaskEnableParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.PostScheduledTaskEnableParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskDisableHeaders(ginCtx *gin.Context) scheduleropenapi.PostScheduledTaskDisableParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.PostScheduledTaskDisableParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskRunHeaders(ginCtx *gin.Context) scheduleropenapi.PostScheduledTaskRunParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.PostScheduledTaskRunParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskRunDetailHeaders(ginCtx *gin.Context) scheduleropenapi.GetScheduledTaskRunParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.GetScheduledTaskRunParams{XGraftLocale: locale, XRequestId: requestID}
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
	writeInvalidSchedulerField(ginCtx, ctx, field)
}

func writeInvalidSchedulerField(ginCtx *gin.Context, ctx *module.Context, field string) {
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
		"field": field,
	})
}

func createTaskMutation(request scheduleropenapi.PostScheduledTaskJSONRequestBody) (schedulercore.TaskMutation, bool) {
	if string(request.TaskType) != string(cronx.TaskTypeHTTP) {
		return schedulercore.TaskMutation{}, false
	}
	configJSON, ok := marshalHTTPConfig(request.Config)
	if !ok {
		return schedulercore.TaskMutation{}, false
	}
	return schedulercore.TaskMutation{
		TaskKey:        strings.TrimSpace(request.TaskKey),
		TaskType:       cronx.TaskTypeHTTP,
		Title:          strings.TrimSpace(request.Title),
		Description:    trimOptionalString(request.Description),
		CronExpression: strings.TrimSpace(request.CronExpression),
		Enabled:        request.Enabled,
		EnabledSet:     true,
		ConfigJSON:     configJSON,
	}, true
}

func updateTaskMutation(request scheduleropenapi.PutScheduledTaskJSONRequestBody) (schedulercore.TaskMutation, error) {
	mutation := schedulercore.TaskMutation{}
	if request.Title != nil {
		mutation.Title = strings.TrimSpace(*request.Title)
		if mutation.Title == "" {
			return schedulercore.TaskMutation{}, errors.New("title")
		}
	}
	if request.Description != nil {
		mutation.Description = strings.TrimSpace(*request.Description)
	}
	if request.CronExpression != nil {
		mutation.CronExpression = strings.TrimSpace(*request.CronExpression)
		if mutation.CronExpression == "" {
			return schedulercore.TaskMutation{}, errors.New("cron_expression")
		}
	}
	if request.Enabled != nil {
		mutation.Enabled = *request.Enabled
		mutation.EnabledSet = true
	}
	if request.Config != nil {
		configJSON, ok := marshalHTTPConfig(request.Config)
		if !ok {
			return schedulercore.TaskMutation{}, errors.New("config")
		}
		mutation.ConfigJSON = configJSON
	}
	return mutation, nil
}

func marshalHTTPConfig(config any) (string, bool) {
	if config == nil {
		return "{}", true
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func trimOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

type schedulerGeneratedHandler struct{}

func (schedulerGeneratedHandler) GetScheduledTasks(params scheduleropenapi.GetScheduledTasksParams) {
	_ = params
}

func (schedulerGeneratedHandler) GetScheduledTask(key string, params scheduleropenapi.GetScheduledTaskParams) {
	_ = key
	_ = params
}

func (schedulerGeneratedHandler) PostScheduledTask(
	params scheduleropenapi.PostScheduledTaskParams,
	body scheduleropenapi.PostScheduledTaskJSONRequestBody,
) {
	_ = params
	_ = body
}

func (schedulerGeneratedHandler) PutScheduledTask(
	key string,
	params scheduleropenapi.PutScheduledTaskParams,
	body scheduleropenapi.PutScheduledTaskJSONRequestBody,
) {
	_ = key
	_ = params
	_ = body
}

func (schedulerGeneratedHandler) DeleteScheduledTask(key string, params scheduleropenapi.DeleteScheduledTaskParams) {
	_ = key
	_ = params
}

func (schedulerGeneratedHandler) PostScheduledTaskEnable(key string, params scheduleropenapi.PostScheduledTaskEnableParams) {
	_ = key
	_ = params
}

func (schedulerGeneratedHandler) PostScheduledTaskDisable(key string, params scheduleropenapi.PostScheduledTaskDisableParams) {
	_ = key
	_ = params
}

func (schedulerGeneratedHandler) GetScheduledTaskRuns(key string, params scheduleropenapi.GetScheduledTaskRunsParams) {
	_ = key
	_ = params
}

func (schedulerGeneratedHandler) GetScheduledTaskRun(runID uint64, params scheduleropenapi.GetScheduledTaskRunParams) {
	_ = runID
	_ = params
}

func (schedulerGeneratedHandler) PostScheduledTaskRun(key string, params scheduleropenapi.PostScheduledTaskRunParams) {
	_ = key
	_ = params
}
