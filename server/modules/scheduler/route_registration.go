// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	schedulercore "graft/server/internal/scheduler"
	schedulercontract "graft/server/modules/scheduler/contract"
)

const (
	defaultScheduledTaskListLimit    = 20
	maxScheduledTaskListLimit        = 100
	defaultScheduledTaskRunListLimit = 20
	maxScheduledTaskRunListLimit     = 100
)

type schedulerRouteRuntime struct {
	ctx        *module.Context
	moduleName string
	runtime    func() (schedulercore.Runtime, error)
}

func registerSchedulerRoutesWithRuntime(
	ctx *module.Context,
	moduleName string,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
	runtime func() (schedulercore.Runtime, error),
) error {
	var err error
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
		schedulercontract.ScheduledTaskJobDefinitionsRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		routeRuntime.handleListJobDefinitions,
	)
	group.GET(
		schedulercontract.ScheduledTaskJobDefinitionDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		routeRuntime.handleGetJobDefinition,
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
	group.POST(
		schedulercontract.ScheduledTaskActionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskRunPermission.String(), publisher),
		routeRuntime.handleRunAction,
	)

	return nil
}

func (r schedulerRouteRuntime) handleListTasks(ginCtx *gin.Context) {
	params, ok := bindGeneratedTaskListParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.GetScheduledTasks(params)

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	limit, offset := normalizedTaskListWindow(params)
	tasks, err := runtime.ListTasks(ginCtx.Request.Context(), schedulercore.TaskListQuery{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		r.writeRouteError(ginCtx, "list scheduled tasks failed", err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskListResponse(tasks, limit, offset))
}

func (r schedulerRouteRuntime) handleListJobDefinitions(ginCtx *gin.Context) {
	schedulerGeneratedHandler{}.GetScheduledTaskJobDefinitions(bindGeneratedTaskJobDefinitionsHeaders(ginCtx))

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	definitions, err := runtime.ListJobDefinitions(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, "list scheduled task job definitions failed", err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskJobDefinitionListResponse(definitions))
}

func (r schedulerRouteRuntime) handleGetJobDefinition(ginCtx *gin.Context) {
	jobKey, ok := readScheduledTaskJobKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.GetScheduledTaskJobDefinition(jobKey, bindGeneratedTaskJobDefinitionDetailHeaders(ginCtx))

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	definition, err := runtime.GetJobDefinition(ginCtx.Request.Context(), jobKey)
	if err != nil {
		r.writeRouteError(ginCtx, "read scheduled task job definition failed", err, zap.String("jobKey", jobKey))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskJobDefinitionItem(definition))
}

func (r schedulerRouteRuntime) handleGetTask(ginCtx *gin.Context) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.GetScheduledTask(key, bindGeneratedTaskDetailHeaders(ginCtx))

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	task, err := runtime.GetTask(ginCtx.Request.Context(), key)
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
			"field": "job_key",
		})
		return
	}

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	task, err := runtime.CreateTask(ginCtx.Request.Context(), command)
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
	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	task, err := runtime.UpdateTask(ginCtx.Request.Context(), key, command)
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

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	if err := runtime.DeleteTask(ginCtx.Request.Context(), key); err != nil {
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

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	task, err := runtime.SetTaskEnabled(ginCtx.Request.Context(), key, enabled)
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
	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	result, err := runtime.ListRuns(ginCtx.Request.Context(), schedulercore.RunListQuery{
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

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	run, err := runtime.GetRun(ginCtx.Request.Context(), runID)
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

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	run, err := runtime.RunOnceWithTrigger(ginCtx.Request.Context(), key, schedulercore.RunTrigger{
		Type:          schedulercore.TriggerTypeManual,
		TriggerUserID: schedulerManualTriggerUserID(ginCtx),
	})
	if err != nil {
		r.writeRouteError(ginCtx, "run scheduled task once failed", err, zap.String("taskKey", key))
		return
	}
	r.ctx.Logger.Debug("scheduled task manual run completed",
		zap.String("module", r.moduleName),
		zap.String("taskKey", key),
		zap.Uint64("runID", run.ID),
		zap.String("status", string(run.Status)),
		zap.Uint64("triggerUserID", schedulerManualTriggerUserID(ginCtx)),
	)

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskRunItem(run))
}

func schedulerManualTriggerUserID(ginCtx *gin.Context) uint64 {
	if ginCtx == nil || ginCtx.Request == nil {
		return 0
	}
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || requestAuth.User == nil {
		return 0
	}
	return requestAuth.User.ID
}

func (r schedulerRouteRuntime) handleRunAction(ginCtx *gin.Context) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	actionKey, ok := readScheduledTaskActionKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	schedulerGeneratedHandler{}.PostScheduledTaskAction(
		key,
		actionKey,
		bindGeneratedTaskActionHeaders(ginCtx),
		scheduleropenapi.PostScheduledTaskActionJSONRequestBody{},
	)
	requestConfig, ok := bindScheduledTaskActionConfig(ginCtx, r.ctx)
	if !ok {
		return
	}

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	result, err := runtime.RunAction(ginCtx.Request.Context(), key, actionKey, requestConfig)
	if err != nil {
		r.writeRouteError(ginCtx, "run scheduled task action failed", err, zap.String("taskKey", key), zap.String("actionKey", actionKey))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskActionResult(result))
}

func (r schedulerRouteRuntime) resolveRuntime(ginCtx *gin.Context) (schedulercore.Runtime, bool) {
	if r.runtime == nil {
		r.writeRouteError(ginCtx, "resolve scheduler runtime failed", errors.New("scheduler runtime resolver is unavailable"))
		return nil, false
	}
	runtime, err := r.runtime()
	if err != nil {
		r.writeRouteError(ginCtx, "resolve scheduler runtime failed", err)
		return nil, false
	}
	return runtime, true
}

func (r schedulerRouteRuntime) writeRouteError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	var configErr schedulercore.ConfigValidationError
	switch {
	case errors.Is(err, schedulercore.ErrTaskNotFound), errors.Is(err, schedulercore.ErrJobDefinitionNotFound), errors.Is(err, schedulercore.ErrJobActionNotFound):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, schedulercontract.ScheduledTaskNotFound.String(), nil)
	case errors.Is(err, schedulercore.ErrTaskAlreadyRunning):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusConflict, schedulercontract.ScheduledTaskAlreadyRunning.String(), nil)
	case errors.As(err, &configErr):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), configErr.Details())
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

func readScheduledTaskJobKey(ginCtx *gin.Context, ctx *module.Context) (string, bool) {
	key := strings.TrimSpace(ginCtx.Param("jobKey"))
	if key == "" {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": "jobKey",
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

func readScheduledTaskActionKey(ginCtx *gin.Context, ctx *module.Context) (string, bool) {
	key := strings.TrimSpace(ginCtx.Param("actionKey"))
	if key == "" {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": "actionKey",
		})
		return "", false
	}
	return key, true
}

func bindScheduledTaskActionConfig(ginCtx *gin.Context, ctx *module.Context) (string, bool) {
	if ginCtx.Request.Body == nil || ginCtx.Request.ContentLength == 0 {
		return "{}", true
	}
	var request scheduleropenapi.PostScheduledTaskActionJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		writeInvalidSchedulerField(ginCtx, ctx, "body")
		return "", false
	}
	rawConfig, err := marshalScheduledTaskActionConfig(request.ConfigJson)
	if err != nil {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), map[string]any{
			"field": "config_json",
		})
		return "", false
	}
	configJSON, err := normalizeScheduledTaskActionConfig(rawConfig)
	if err != nil {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), map[string]any{
			"field": "config_json",
		})
		return "", false
	}
	return configJSON, true
}

func marshalScheduledTaskActionConfig(config *map[string]interface{}) (json.RawMessage, error) {
	if config == nil {
		return nil, nil
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func normalizeScheduledTaskActionConfig(raw json.RawMessage) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return "{}", nil
	}
	if !isSchedulerJSONObject(trimmed) {
		return "", errors.New("config_json must be a JSON object")
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func isSchedulerJSONObject(value string) bool {
	var decoded map[string]any
	return json.Unmarshal([]byte(strings.TrimSpace(value)), &decoded) == nil
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

func bindGeneratedTaskListParams(ginCtx *gin.Context, ctx *module.Context) (scheduleropenapi.GetScheduledTasksParams, bool) {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	params := scheduleropenapi.GetScheduledTasksParams{XGraftLocale: locale, XRequestId: requestID}

	limit, offset, ok := bindScheduledTaskWindowParams(ginCtx, ctx, maxScheduledTaskListLimit)
	if !ok {
		return scheduleropenapi.GetScheduledTasksParams{}, false
	}
	params.Limit = limit
	params.Offset = offset

	return params, true
}

func bindGeneratedTaskCreateHeaders(ginCtx *gin.Context) scheduleropenapi.PostScheduledTaskParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.PostScheduledTaskParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskJobDefinitionsHeaders(ginCtx *gin.Context) scheduleropenapi.GetScheduledTaskJobDefinitionsParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.GetScheduledTaskJobDefinitionsParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskJobDefinitionDetailHeaders(ginCtx *gin.Context) scheduleropenapi.GetScheduledTaskJobDefinitionParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.GetScheduledTaskJobDefinitionParams{XGraftLocale: locale, XRequestId: requestID}
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

func bindGeneratedTaskActionHeaders(ginCtx *gin.Context) scheduleropenapi.PostScheduledTaskActionParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.PostScheduledTaskActionParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedTaskRunDetailHeaders(ginCtx *gin.Context) scheduleropenapi.GetScheduledTaskRunParams {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	return scheduleropenapi.GetScheduledTaskRunParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRunListParams(ginCtx *gin.Context, ctx *module.Context) (scheduleropenapi.GetScheduledTaskRunsParams, bool) {
	locale, requestID := bindGeneratedSchedulerHeaders(ginCtx)
	params := scheduleropenapi.GetScheduledTaskRunsParams{XGraftLocale: locale, XRequestId: requestID}

	limit, offset, ok := bindScheduledTaskWindowParams(ginCtx, ctx, maxScheduledTaskRunListLimit)
	if !ok {
		return scheduleropenapi.GetScheduledTaskRunsParams{}, false
	}
	params.Limit = limit
	params.Offset = offset

	return params, true
}

func bindScheduledTaskWindowParams(ginCtx *gin.Context, ctx *module.Context, maxLimit int) (*int, *int, bool) {
	var parsedLimit *int
	if raw := strings.TrimSpace(ginCtx.Query("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxLimit {
			writeInvalidSchedulerQuery(ginCtx, ctx, "limit")
			return nil, nil, false
		}
		parsedLimit = &limit
	}

	var parsedOffset *int
	if raw := strings.TrimSpace(ginCtx.Query("offset")); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			writeInvalidSchedulerQuery(ginCtx, ctx, "offset")
			return nil, nil, false
		}
		parsedOffset = &offset
	}
	return parsedLimit, parsedOffset, true
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

func normalizedTaskListWindow(params scheduleropenapi.GetScheduledTasksParams) (int, int) {
	limit := defaultScheduledTaskListLimit
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
	if strings.TrimSpace(request.JobKey) == "" {
		return schedulercore.TaskMutation{}, false
	}
	return schedulercore.TaskMutation{
		TaskKey:        strings.TrimSpace(request.TaskKey),
		JobKey:         strings.TrimSpace(request.JobKey),
		Title:          strings.TrimSpace(request.Title),
		Description:    trimOptionalString(request.Description),
		CronExpression: strings.TrimSpace(request.CronExpression),
		Enabled:        request.Enabled,
		EnabledSet:     true,
		ConfigJSON:     trimOptionalString(request.ConfigJson),
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
	if request.ConfigJson != nil {
		mutation.ConfigJSON = strings.TrimSpace(*request.ConfigJson)
	}
	return mutation, nil
}

func trimOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

// schedulerGeneratedHandler 只在手写路由中引用生成类型，确保 OpenAPI 绑定在编译期持续校验。
type schedulerGeneratedHandler struct{}

func (schedulerGeneratedHandler) GetScheduledTasks(params scheduleropenapi.GetScheduledTasksParams) {
	_ = params
}

func (schedulerGeneratedHandler) GetScheduledTaskJobDefinitions(params scheduleropenapi.GetScheduledTaskJobDefinitionsParams) {
	_ = params
}

func (schedulerGeneratedHandler) GetScheduledTaskJobDefinition(
	jobKey string,
	params scheduleropenapi.GetScheduledTaskJobDefinitionParams,
) {
	_ = jobKey
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

func (schedulerGeneratedHandler) PostScheduledTaskAction(
	taskKey string,
	actionKey string,
	params scheduleropenapi.PostScheduledTaskActionParams,
	body scheduleropenapi.PostScheduledTaskActionJSONRequestBody,
) {
	_ = taskKey
	_ = actionKey
	_ = params
	_ = body
}
