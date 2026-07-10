package task

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	taskstore "graft/server/modules/task/store"
)

// registerTaskRoutes registers task endpoints and applies request identification and permission middleware.
// It returns an error if the required authentication or authorization services cannot be resolved.
func registerTaskRoutes(ctx *module.Context, runtime *Runtime, publisher httpx.SecurityAuditPublisher) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	auth, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return err
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return err
	}
	g := ctx.Router.Group("/tasks")
	g.Use(httpx.RequestIDMiddleware(), httpx.RequirePermission(ctx.I18n, auth, authorizer, "", publisher))
	r := taskRoutes{runtime: runtime, ctx: ctx}
	g.GET("", r.list)
	g.GET("/:id", r.detail)
	g.GET("/:id/stages", r.stages)
	g.GET("/:id/events", r.events)
	g.GET("/:id/logs", r.logs)
	g.POST("/:id/cancel", r.cancel)
	g.POST("/:id/stages/:stageID/retry", r.retry)
	return nil
}

type taskRoutes struct {
	runtime *Runtime
	ctx     *module.Context
}

const (
	defaultTaskListLimit  = 20
	maxTaskListLimit      = 100
	defaultTaskEventLimit = 100
	maxTaskEventLimit     = 500
	defaultTaskLogLimit   = 200
	maxTaskLogLimit       = 1000
)

func (r taskRoutes) get(c *gin.Context, action moduleapi.TaskOwnerAction) (moduleapi.TaskView, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		r.writeError(c, http.StatusBadRequest, errTaskInvalidArgument)
		return moduleapi.TaskView{}, false
	}
	task, err := r.runtime.GetTask(c.Request.Context(), id)
	if err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return moduleapi.TaskView{}, false
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(c.Request.Context())
	if !ok || auth.User == nil || r.runtime.AuthorizeOwner(c.Request.Context(), auth.User, action, task.Owner) != nil {
		r.writeError(c, http.StatusNotFound, taskstore.ErrNotFound)
		return moduleapi.TaskView{}, false
	}
	return task, true
}

func (r taskRoutes) detail(c *gin.Context) {
	task, ok := r.get(c, moduleapi.TaskOwnerActionView)
	if !ok {
		return
	}
	r.writeDetail(c, http.StatusOK, task)
}

func (r taskRoutes) stages(c *gin.Context) {
	task, ok := r.get(c, moduleapi.TaskOwnerActionView)
	if !ok {
		return
	}
	stages, err := r.runtime.ListTaskStages(c.Request.Context(), task.ID)
	if err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, map[string]any{"items": taskStageResponses(stages)})
}

func (r taskRoutes) events(c *gin.Context) {
	r.replay(c, taskReplayEvents, defaultTaskEventLimit, maxTaskEventLimit)
}

func (r taskRoutes) logs(c *gin.Context) {
	r.replay(c, taskReplayLogs, defaultTaskLogLimit, maxTaskLogLimit)
}

type taskReplayKind uint8

const (
	taskReplayEvents taskReplayKind = iota
	taskReplayLogs
)

func (r taskRoutes) replay(c *gin.Context, kind taskReplayKind, defaultLimit, maxLimit int) {
	task, ok := r.get(c, moduleapi.TaskOwnerActionView)
	if !ok {
		return
	}
	after, limit := taskSequencePage(c, defaultLimit, maxLimit)
	items, next, err := r.replayItems(c, task.ID, kind, after, limit)
	if err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, map[string]any{"items": items, "next_after_sequence": next})
}

func (r taskRoutes) replayItems(c *gin.Context, taskID uint64, kind taskReplayKind, after int64, limit int) (any, int64, error) {
	if kind == taskReplayEvents {
		items, err := r.runtime.ListTaskEvents(c.Request.Context(), taskID, after, limit)
		if len(items) > 0 {
			after = items[len(items)-1].Sequence
		}
		return taskEventResponses(items), after, err
	}
	items, err := r.runtime.ListTaskLogs(c.Request.Context(), taskID, after, limit)
	if len(items) > 0 {
		after = items[len(items)-1].Sequence
	}
	return taskLogResponses(items), after, err
}

func (r taskRoutes) cancel(c *gin.Context) {
	task, ok := r.get(c, moduleapi.TaskOwnerActionCancel)
	if !ok {
		return
	}
	if err := r.runtime.Cancel(c.Request.Context(), task.ID); err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return
	}
	updated, err := r.runtime.GetTask(c.Request.Context(), task.ID)
	if err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return
	}
	r.writeDetail(c, http.StatusOK, updated)
}

func (r taskRoutes) retry(c *gin.Context) {
	task, ok := r.get(c, moduleapi.TaskOwnerActionRetry)
	if !ok {
		return
	}
	stageID, err := strconv.ParseUint(c.Param("stageID"), 10, 64)
	if err != nil || stageID == 0 {
		r.writeError(c, http.StatusBadRequest, errTaskInvalidArgument)
		return
	}
	if err := r.runtime.RetryStage(c.Request.Context(), task.ID, stageID); err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return
	}
	updated, err := r.runtime.GetTask(c.Request.Context(), task.ID)
	if err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return
	}
	r.writeDetail(c, http.StatusAccepted, updated)
}

func (r taskRoutes) list(c *gin.Context) {
	limit, offset := taskListPage(c)
	auth, ok := moduleapi.RequestAuthContextFromContext(c.Request.Context())
	if !ok || auth.User == nil {
		httpx.WriteSuccess(c, http.StatusOK, map[string]any{"items": []map[string]any{}, "total": 0, "limit": limit, "offset": offset})
		return
	}
	owner := moduleapi.TaskOwner{Type: c.Query("owner_type"), ID: c.Query("owner_id")}
	if owner.Type == "" || owner.ID == "" {
		r.writeError(c, http.StatusBadRequest, errTaskInvalidArgument)
		return
	}
	if err := r.runtime.AuthorizeOwner(c.Request.Context(), auth.User, moduleapi.TaskOwnerActionView, owner); err != nil {
		r.writeError(c, http.StatusNotFound, taskstore.ErrNotFound)
		return
	}
	filter, err := taskListFilter(c, owner)
	if err != nil {
		r.writeError(c, http.StatusBadRequest, err)
		return
	}
	tasks, total, err := r.runtime.ListTasks(c.Request.Context(), filter, limit, offset)
	if err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, map[string]any{"items": taskSummaryResponses(tasks), "total": total, "limit": limit, "offset": offset})
}

func (r taskRoutes) writeDetail(c *gin.Context, status int, task moduleapi.TaskView) {
	stages, err := r.runtime.ListTaskStages(c.Request.Context(), task.ID)
	if err != nil {
		r.writeError(c, taskHTTPStatus(err), err)
		return
	}
	auth, _ := moduleapi.RequestAuthContextFromContext(c.Request.Context())
	response := taskSummaryResponse(task)
	response["capabilities"] = taskCapabilities(c.Request.Context(), r.runtime, auth.User, task, stages)
	response["stages"] = taskStageResponses(stages)
	httpx.WriteSuccess(c, status, response)
}

// taskSummaryResponse converts a task view into its canonical OpenAPI response object.
func taskSummaryResponse(task moduleapi.TaskView) map[string]any {
	return map[string]any{
		"id":                task.ID,
		"type":              task.Type,
		"owner_type":        task.Owner.Type,
		"owner_id":          task.Owner.ID,
		"status":            task.Status,
		"current_stage_key": task.CurrentStageKey,
		"created_by":        task.CreatedBy,
		"created_at":        task.CreatedAt,
		"started_at":        task.StartedAt,
		"finished_at":       task.FinishedAt,
		"duration_ms":       task.DurationMS,
		"failure_code":      task.FailureCode,
		"failure_message":   task.FailureMessage,
	}
}

// taskSummaryResponses 将任务视图列表转换为任务摘要响应对象列表。
func taskSummaryResponses(tasks []moduleapi.TaskView) []map[string]any {
	items := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, taskSummaryResponse(task))
	}
	return items
}

// taskStageResponse 将任务阶段视图转换为 API 响应对象。
// 返回包含阶段标识、执行信息、状态、时间信息及失败详情的字段映射。
func taskStageResponse(stage moduleapi.TaskStageView) map[string]any {
	return map[string]any{
		"id":              stage.ID,
		"key":             stage.Key,
		"sequence":        stage.Sequence,
		"executor_type":   stage.ExecutorType,
		"status":          stage.Status,
		"attempt":         stage.Attempt,
		"max_attempts":    stage.MaxAttempts,
		"recovery_policy": stage.RecoveryPolicy,
		"started_at":      stage.StartedAt,
		"finished_at":     stage.FinishedAt,
		"duration_ms":     stage.DurationMS,
		"failure_code":    stage.FailureCode,
		"failure_message": stage.FailureMessage,
	}
}

// taskStageResponses 将任务阶段视图转换为阶段响应对象列表。
func taskStageResponses(stages []moduleapi.TaskStageView) []map[string]any {
	items := make([]map[string]any, 0, len(stages))
	for _, stage := range stages {
		items = append(items, taskStageResponse(stage))
	}
	return items
}

// taskEventResponse 将任务事件视图转换为 API 响应对象。
// 返回包含事件标识、序列号、类型、载荷和创建时间的响应映射。
func taskEventResponse(event moduleapi.TaskEventView) map[string]any {
	return map[string]any{
		"id":         event.ID,
		"sequence":   event.Sequence,
		"type":       event.Type,
		"payload":    event.Payload,
		"created_at": event.CreatedAt,
	}
}

// taskEventResponses converts task event views into response objects.
func taskEventResponses(events []moduleapi.TaskEventView) []map[string]any {
	items := make([]map[string]any, 0, len(events))
	for _, event := range events {
		items = append(items, taskEventResponse(event))
	}
	return items
}

// taskLogResponse 将任务日志视图转换为 API 响应对象。
func taskLogResponse(log moduleapi.TaskLogView) map[string]any {
	return map[string]any{
		"id":          log.ID,
		"sequence":    log.Sequence,
		"stage_id":    log.StageID,
		"stream":      log.Stream,
		"level":       log.Level,
		"line":        log.Line,
		"occurred_at": log.OccurredAt,
	}
}

// taskLogResponses converts task log views into response objects.
func taskLogResponses(logs []moduleapi.TaskLogView) []map[string]any {
	items := make([]map[string]any, 0, len(logs))
	for _, log := range logs {
		items = append(items, taskLogResponse(log))
	}
	return items
}

func (r taskRoutes) writeError(c *gin.Context, status int, err error) {
	key := messagecontract.CommonInternalError.String()
	if status == http.StatusBadRequest {
		key = messagecontract.CommonInvalidArgument.String()
	}
	httpx.WriteLocalizedError(c, r.ctx.I18n, status, key, map[string]any{"error": err.Error()})
	c.Abort()
}

var errTaskInvalidArgument = errors.New("invalid task argument")

func taskListFilter(c *gin.Context, owner moduleapi.TaskOwner) (moduleapi.TaskListFilter, error) {
	filter := moduleapi.TaskListFilter{Owner: owner}
	if taskType := c.Query("type"); taskType != "" {
		value := moduleapi.TaskType(taskType)
		filter.Type = &value
	}
	if status := c.Query("status"); status != "" {
		value := moduleapi.TaskStatus(status)
		if !isTaskStatus(value) {
			return moduleapi.TaskListFilter{}, errTaskInvalidArgument
		}
		filter.Status = &value
	}
	return filter, nil
}

func isTaskStatus(status moduleapi.TaskStatus) bool {
	switch status {
	case moduleapi.TaskStatusPending, moduleapi.TaskStatusScheduled, moduleapi.TaskStatusRunning,
		moduleapi.TaskStatusSuccess, moduleapi.TaskStatusFailed, moduleapi.TaskStatusCancelled, moduleapi.TaskStatusNeedsAttention:
		return true
	default:
		return false
	}
}

// taskHTTPStatus maps task store errors to their corresponding HTTP status codes.
func taskHTTPStatus(err error) int {
	switch {
	case errors.Is(err, taskstore.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, taskstore.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, taskstore.ErrStateConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// taskSequencePage parses and bounds the sequence cursor and page size from the request query.
func taskSequencePage(c *gin.Context, defaultLimit, maxLimit int) (int64, int) {
	after, _ := strconv.ParseInt(c.DefaultQuery("after_sequence", "0"), 10, 64)
	if after < 0 {
		after = 0
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return after, limit
}

// taskListPage parses and bounds the task list pagination parameters.
// It returns the requested item limit and offset.
func taskListPage(c *gin.Context) (int, int) {
	_, limit := taskSequencePage(c, defaultTaskListLimit, maxTaskListLimit)
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// taskCapabilities 根据任务及阶段状态计算可用操作能力。
// 返回包含 cancel、retry 和 download_log 能力标记的映射。
func taskCapabilities(ctx context.Context, runtime *Runtime, actor *moduleapi.CurrentUser, task moduleapi.TaskView, stages []moduleapi.TaskStageView) map[string]bool {
	return map[string]bool{
		"cancel":       taskSupportsCancellation(task) && taskActionAllowed(ctx, runtime, actor, moduleapi.TaskOwnerActionCancel, task.Owner),
		"retry":        taskHasRetryableStage(stages) && taskSupportsRetry(task) && taskActionAllowed(ctx, runtime, actor, moduleapi.TaskOwnerActionRetry, task.Owner),
		"download_log": true,
	}
}

func taskSupportsCancellation(task moduleapi.TaskView) bool {
	switch task.Status {
	case moduleapi.TaskStatusPending, moduleapi.TaskStatusScheduled, moduleapi.TaskStatusRunning, moduleapi.TaskStatusNeedsAttention:
		return true
	default:
		return false
	}
}

func taskHasRetryableStage(stages []moduleapi.TaskStageView) bool {
	for _, stage := range stages {
		if stage.Status == moduleapi.StageStatusFailed || stage.Status == moduleapi.StageStatusUnknown {
			return true
		}
	}
	return false
}

func taskSupportsRetry(task moduleapi.TaskView) bool {
	return task.Status == moduleapi.TaskStatusFailed || task.Status == moduleapi.TaskStatusNeedsAttention
}

func taskActionAllowed(ctx context.Context, runtime *Runtime, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) bool {
	return actor != nil && runtime != nil && runtime.AuthorizeOwner(ctx, actor, action, owner) == nil
}
