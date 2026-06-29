package scheduler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	scheduleropenapi "graft/server/internal/contract/openapi/scheduler"
	"graft/server/internal/httpx"
	"graft/server/internal/logger/logsafe"
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
	triggerUserID := schedulerManualTriggerUserID(ginCtx)
	run, err := runtime.RunOnceWithTrigger(ginCtx.Request.Context(), key, schedulercore.RunTrigger{
		Type:          schedulercore.TriggerTypeManual,
		TriggerUserID: triggerUserID,
	})
	if err != nil {
		r.writeRouteError(ginCtx, "run scheduled task once failed", err, zap.String("taskKey", key))
		return
	}
	r.ctx.Logger.Debug("scheduled task manual run completed",
		zap.String("module", r.moduleName),
		zap.Uint64("runID", run.ID),
		zap.String("status", string(run.Status)),
		zap.Uint64("triggerUserID", triggerUserID),
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
	case errors.Is(err, schedulercore.ErrTaskKeyConflict):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), map[string]any{
			"field": "task_key",
		})
	case errors.Is(err, schedulercore.ErrTaskTitleConflict):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), map[string]any{
			"field": "title",
		})
	case errors.Is(err, schedulercore.ErrTaskImmutable), errors.Is(err, schedulercore.ErrTaskValidation):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, schedulercontract.ScheduledTaskInvalidRequest.String(), nil)
	default:
		if r.ctx != nil && r.ctx.Logger != nil {
			logFields := append([]zap.Field{zap.String("module", r.moduleName), zap.Error(err)}, fields...)
			logsafe.Error(r.ctx.Logger, message, logFields...)
		}
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	}
}
