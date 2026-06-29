package scheduler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	httpheader "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	scheduleropenapi "graft/server/internal/contract/openapi/scheduler"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	schedulercore "graft/server/internal/scheduler"
	schedulercontract "graft/server/modules/scheduler/contract"
)

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
