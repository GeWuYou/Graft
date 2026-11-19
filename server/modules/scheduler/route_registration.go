package scheduler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	scheduleropenapi "graft/server/internal/contract/openapi/scheduler"
	"graft/server/internal/contract/openapi/strictsupport"
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
	savedViewService, err := module.ResolveService[moduleapi.SavedViewService](ctx.Services, (*moduleapi.SavedViewService)(nil))
	if err != nil {
		return err
	}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleName)
	group := ctx.Router.Group(schedulercontract.ScheduledTasksGroup)
	group.Use(httpx.RequestIDMiddleware())
	// 生成 wrapper 承担路径与请求参数绑定；权限中间件与 handler envelope 写入保持迁移前语义。
	wrapper := scheduleropenapi.ServerInterfaceWrapper{
		Handler: scheduleropenapi.NewStrictHandlerWithOptions(
			schedulerStrictServer{
				runtime:    routeRuntime,
				savedViews: schedulerSavedViewHandlersOf(ctx, savedViewService),
			},
			nil,
			scheduleropenapi.StrictGinServerOptions{
				RequestErrorHandlerFunc: strictsupport.RequestErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String(), gin.H{"field": "body"}),
				HandlerErrorFunc:        strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
			},
		),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String()),
	}
	group.GET(
		schedulercontract.ScheduledTaskCollectionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		strictsupport.Route(wrapper.GetScheduledTasks, routeRuntime.handleListTasks),
	)
	if err := registerSchedulerSavedViews(group, ctx, authService, authorizer, publisher, wrapper); err != nil {
		return err
	}
	group.POST(
		schedulercontract.ScheduledTaskCollectionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskCreatePermission.String(), publisher),
		strictsupport.Route(wrapper.PostScheduledTask, nil),
	)
	group.GET(
		schedulercontract.ScheduledTaskJobDefinitionsRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		strictsupport.Route(wrapper.GetScheduledTaskJobDefinitions, nil),
	)
	group.GET(
		schedulercontract.ScheduledTaskJobDefinitionDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		strictsupport.Route(wrapper.GetScheduledTaskJobDefinition, nil),
	)
	group.GET(
		schedulercontract.ScheduledTaskRunDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		strictsupport.Route(wrapper.GetScheduledTaskRun, routeRuntime.handleGetRun),
	)
	group.GET(
		schedulercontract.ScheduledTaskDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		strictsupport.Route(wrapper.GetScheduledTask, nil),
	)
	group.PUT(
		schedulercontract.ScheduledTaskDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskUpdatePermission.String(), publisher),
		strictsupport.Route(wrapper.PutScheduledTask, nil),
	)
	group.DELETE(
		schedulercontract.ScheduledTaskDetailRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskDeletePermission.String(), publisher),
		strictsupport.Route(wrapper.DeleteScheduledTask, nil),
	)
	registerScheduledTaskActionRoutes(group, routeRuntime, authService, authorizer, publisher, wrapper)
	return nil
}

func registerScheduledTaskActionRoutes(group *gin.RouterGroup, runtime schedulerRouteRuntime, authService moduleapi.AuthService, authorizer moduleapi.Authorizer, publisher httpx.SecurityAuditPublisher, wrapper scheduleropenapi.ServerInterfaceWrapper) {
	ctx := runtime.ctx
	group.POST(
		schedulercontract.ScheduledTaskEnableRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskEnablePermission.String(), publisher),
		strictsupport.Route(wrapper.PostScheduledTaskEnable, nil),
	)
	group.POST(
		schedulercontract.ScheduledTaskDisableRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskEnablePermission.String(), publisher),
		strictsupport.Route(wrapper.PostScheduledTaskDisable, nil),
	)
	group.GET(
		schedulercontract.ScheduledTaskRunsRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		strictsupport.Route(wrapper.GetScheduledTaskRuns, runtime.handleListRuns),
	)
	group.POST(
		schedulercontract.ScheduledTaskRunRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskRunPermission.String(), publisher),
		strictsupport.Route(wrapper.PostScheduledTaskRun, nil),
	)
	group.POST(
		schedulercontract.ScheduledTaskActionRoute,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskRunPermission.String(), publisher),
		strictsupport.Route(wrapper.PostScheduledTaskAction, nil),
	)

}

func registerSchedulerSavedViews(
	group *gin.RouterGroup,
	ctx *module.Context,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
	publisher httpx.SecurityAuditPublisher,
	wrapper scheduleropenapi.ServerInterfaceWrapper,
) error {
	savedViews, err := module.ResolveService[moduleapi.SavedViewService](ctx.Services, (*moduleapi.SavedViewService)(nil))
	if err != nil {
		return err
	}
	registerSchedulerSavedViewRoutes(
		group,
		ctx,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, schedulercontract.ScheduledTaskReadPermission.String(), publisher),
		wrapper,
		schedulerSavedViewHandlersOf(ctx, savedViews),
	)
	return nil
}

// schedulerStrictServer 将计划任务路由主体适配到生成的 strict server 契约；
// 编译期断言保证 OpenAPI 计划任务操作变化时必须在适配层补齐。
type schedulerStrictServer struct {
	runtime    schedulerRouteRuntime
	savedViews schedulerSavedViewHandlers
}

// errSchedulerStrictMissingBody 表示必填请求体在生成绑定后缺失；该路径不可达。
var errSchedulerStrictMissingBody = errors.New("strict handler request body is missing")

var _ scheduleropenapi.StrictServerInterface = schedulerStrictServer{}

func (s schedulerStrictServer) GetScheduledTasks(ctx context.Context, _ scheduleropenapi.GetScheduledTasksRequestObject) (scheduleropenapi.GetScheduledTasksResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleListTasks)
}

func (s schedulerStrictServer) GetScheduledTask(ctx context.Context, _ scheduleropenapi.GetScheduledTaskRequestObject) (scheduleropenapi.GetScheduledTaskResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleGetTask)
}

func (s schedulerStrictServer) PostScheduledTask(ctx context.Context, request scheduleropenapi.PostScheduledTaskRequestObject) (scheduleropenapi.PostScheduledTaskResponseObject, error) {
	if request.Body == nil {
		return nil, errSchedulerStrictMissingBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.runtime.handleCreateTask(ginCtx, *request.Body) })
}

func (s schedulerStrictServer) PutScheduledTask(ctx context.Context, request scheduleropenapi.PutScheduledTaskRequestObject) (scheduleropenapi.PutScheduledTaskResponseObject, error) {
	if request.Body == nil {
		return nil, errSchedulerStrictMissingBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.runtime.handleUpdateTask(ginCtx, *request.Body) })
}

func (s schedulerStrictServer) DeleteScheduledTask(ctx context.Context, _ scheduleropenapi.DeleteScheduledTaskRequestObject) (scheduleropenapi.DeleteScheduledTaskResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleDeleteTask)
}

func (s schedulerStrictServer) PostScheduledTaskEnable(ctx context.Context, _ scheduleropenapi.PostScheduledTaskEnableRequestObject) (scheduleropenapi.PostScheduledTaskEnableResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleEnableTask)
}

func (s schedulerStrictServer) PostScheduledTaskDisable(ctx context.Context, _ scheduleropenapi.PostScheduledTaskDisableRequestObject) (scheduleropenapi.PostScheduledTaskDisableResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleDisableTask)
}

func (s schedulerStrictServer) GetScheduledTaskRuns(ctx context.Context, _ scheduleropenapi.GetScheduledTaskRunsRequestObject) (scheduleropenapi.GetScheduledTaskRunsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleListRuns)
}

func (s schedulerStrictServer) GetScheduledTaskRun(ctx context.Context, _ scheduleropenapi.GetScheduledTaskRunRequestObject) (scheduleropenapi.GetScheduledTaskRunResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleGetRun)
}

func (s schedulerStrictServer) PostScheduledTaskRun(ctx context.Context, _ scheduleropenapi.PostScheduledTaskRunRequestObject) (scheduleropenapi.PostScheduledTaskRunResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleRunOnce)
}

func (s schedulerStrictServer) GetScheduledTaskJobDefinitions(ctx context.Context, _ scheduleropenapi.GetScheduledTaskJobDefinitionsRequestObject) (scheduleropenapi.GetScheduledTaskJobDefinitionsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleListJobDefinitions)
}

func (s schedulerStrictServer) GetScheduledTaskJobDefinition(ctx context.Context, _ scheduleropenapi.GetScheduledTaskJobDefinitionRequestObject) (scheduleropenapi.GetScheduledTaskJobDefinitionResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.runtime.handleGetJobDefinition)
}

func (s schedulerStrictServer) PostScheduledTaskAction(ctx context.Context, request scheduleropenapi.PostScheduledTaskActionRequestObject) (scheduleropenapi.PostScheduledTaskActionResponseObject, error) {
	var body scheduleropenapi.PostScheduledTaskActionJSONRequestBody
	if request.Body != nil {
		body = *request.Body
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.runtime.handleRunAction(ginCtx, body) })
}

func (s schedulerStrictServer) GetScheduledTaskSavedViews(ctx context.Context, _ scheduleropenapi.GetScheduledTaskSavedViewsRequestObject) (scheduleropenapi.GetScheduledTaskSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.savedViews.list)
}

func (s schedulerStrictServer) PostScheduledTaskSavedView(ctx context.Context, _ scheduleropenapi.PostScheduledTaskSavedViewRequestObject) (scheduleropenapi.PostScheduledTaskSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.savedViews.create)
}

func (s schedulerStrictServer) PutScheduledTaskSavedView(ctx context.Context, _ scheduleropenapi.PutScheduledTaskSavedViewRequestObject) (scheduleropenapi.PutScheduledTaskSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.savedViews.update)
}

func (s schedulerStrictServer) DeleteScheduledTaskSavedView(ctx context.Context, _ scheduleropenapi.DeleteScheduledTaskSavedViewRequestObject) (scheduleropenapi.DeleteScheduledTaskSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.savedViews.delete)
}

func (r schedulerRouteRuntime) handleListTasks(ginCtx *gin.Context) {
	params, ok := bindGeneratedTaskListParams(ginCtx, r.ctx)
	if !ok {
		return
	}

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	limit, offset := normalizedTaskListWindow(params)
	tasks, err := runtime.ListTasks(ginCtx.Request.Context(), schedulercore.TaskListQuery{
		Limit:   limit,
		Offset:  offset,
		Keyword: optionalSchedulerQueryValue(params.Keyword),
		JobKey:  optionalSchedulerQueryValue(params.JobKey),
		Status:  scheduledTaskStatusValue(params.Status),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskListResponse(tasks, limit, offset))
}

func scheduledTaskStatusValue(status *scheduleropenapi.GetScheduledTasksParamsStatus) string {
	if status == nil {
		return ""
	}
	return string(*status)
}

func (r schedulerRouteRuntime) handleListJobDefinitions(ginCtx *gin.Context) {

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	definitions, err := runtime.ListJobDefinitions(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskJobDefinitionListResponse(definitions))
}

func (r schedulerRouteRuntime) handleGetJobDefinition(ginCtx *gin.Context) {
	jobKey, ok := readScheduledTaskJobKey(ginCtx, r.ctx)
	if !ok {
		return
	}

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	definition, err := runtime.GetJobDefinition(ginCtx.Request.Context(), jobKey)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskJobDefinitionItem(definition))
}

func (r schedulerRouteRuntime) handleGetTask(ginCtx *gin.Context) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	task, err := runtime.GetTask(ginCtx.Request.Context(), key)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskItem(task))
}

func (r schedulerRouteRuntime) handleCreateTask(ginCtx *gin.Context, request scheduleropenapi.PostScheduledTaskJSONRequestBody) {
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
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskItem(task))
}

func (r schedulerRouteRuntime) handleUpdateTask(ginCtx *gin.Context, request scheduleropenapi.PutScheduledTaskJSONRequestBody) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}

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
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskItem(task))
}

func (r schedulerRouteRuntime) handleDeleteTask(ginCtx *gin.Context) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	if err := runtime.DeleteTask(ginCtx.Request.Context(), key); err != nil {
		r.writeRouteError(ginCtx, err)
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
	_ = key

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	task, err := runtime.SetTaskEnabled(ginCtx.Request.Context(), key, enabled)
	if err != nil {
		r.writeRouteError(ginCtx, err)
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
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskRunListResponse(result, limit, offset))
}

func (r schedulerRouteRuntime) handleGetRun(ginCtx *gin.Context) {
	runID, ok := readScheduledTaskRunID(ginCtx, r.ctx)
	if !ok {
		return
	}

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	run, err := runtime.GetRun(ginCtx.Request.Context(), runID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskRunItem(run))
}

func (r schedulerRouteRuntime) handleRunOnce(ginCtx *gin.Context) {
	key, ok := readScheduledTaskRunKey(ginCtx, r.ctx)
	if !ok {
		return
	}

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
		r.writeRouteError(ginCtx, err)
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

func (r schedulerRouteRuntime) handleRunAction(ginCtx *gin.Context, request scheduleropenapi.PostScheduledTaskActionJSONRequestBody) {
	key, ok := readScheduledTaskKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	actionKey, ok := readScheduledTaskActionKey(ginCtx, r.ctx)
	if !ok {
		return
	}
	requestConfig, ok := bindScheduledTaskActionConfigBody(ginCtx, r.ctx, request)
	if !ok {
		return
	}

	runtime, ok := r.resolveRuntime(ginCtx)
	if !ok {
		return
	}
	result, err := runtime.RunAction(ginCtx.Request.Context(), key, actionKey, requestConfig)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toScheduledTaskActionResult(result))
}

func (r schedulerRouteRuntime) resolveRuntime(ginCtx *gin.Context) (schedulercore.Runtime, bool) {
	if r.runtime == nil {
		r.writeRouteError(ginCtx, errors.New("scheduler runtime resolver is unavailable"))
		return nil, false
	}
	runtime, err := r.runtime()
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return nil, false
	}
	return runtime, true
}

func (r schedulerRouteRuntime) writeRouteError(ginCtx *gin.Context, err error) {
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
		httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, err)
	}
}
