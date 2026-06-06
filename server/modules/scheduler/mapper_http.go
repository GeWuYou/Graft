package scheduler

import (
	"math"
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	schedulercore "graft/server/internal/scheduler"
)

func toScheduledTaskListResponse(
	result schedulercore.TaskListResult,
	limit int,
	offset int,
) generated.ScheduledTaskListResponse {
	items := make([]generated.ScheduledTaskItem, 0, len(result.Items))
	for _, task := range result.Items {
		items = append(items, toScheduledTaskItem(task))
	}

	return generated.ScheduledTaskListResponse{
		Items:  items,
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	}
}

func toScheduledTaskJobDefinitionListResponse(definitions []schedulercore.JobDefinitionSnapshot) generated.ScheduledTaskJobDefinitionListResponse {
	items := make([]generated.ScheduledTaskJobDefinitionItem, 0, len(definitions))
	for _, definition := range definitions {
		items = append(items, toScheduledTaskJobDefinitionItem(definition))
	}

	return generated.ScheduledTaskJobDefinitionListResponse{
		Items: items,
		Total: len(items),
	}
}

func toScheduledTaskJobDefinitionItem(definition schedulercore.JobDefinitionSnapshot) generated.ScheduledTaskJobDefinitionItem {
	moduleKey := strings.TrimSpace(definition.ModuleKey)
	return generated.ScheduledTaskJobDefinitionItem{
		Key:                   strings.TrimSpace(definition.JobKey),
		Owner:                 moduleKey,
		Module:                moduleKey,
		DisplayNameKey:        strings.TrimSpace(definition.TitleKey),
		DescriptionKey:        strings.TrimSpace(definition.DescriptionKey),
		Title:                 stringPointer(definition.Title),
		Description:           stringPointer(definition.Description),
		ParamsSchemaJson:      defaultJSONObject(definition.ParamsSchema),
		DefaultParamsJson:     defaultJSONObject(definition.DefaultParams),
		DefaultCronExpression: strings.TrimSpace(definition.DefaultCron),
		DefaultEnabled:        definition.Enabled,
	}
}

func toScheduledTaskItem(task schedulercore.TaskSnapshot) generated.ScheduledTaskItem {
	status := generated.ScheduledTaskItemStatusIdle
	var lastRun *generated.ScheduledTaskLastRun
	if task.LastRun != nil {
		status = generated.ScheduledTaskItemStatus(task.LastRun.Status)
		mapped := toScheduledTaskLastRun(*task.LastRun)
		lastRun = &mapped
	}
	if task.Running {
		status = generated.ScheduledTaskItemStatusRunning
	}
	if !status.Valid() {
		status = generated.ScheduledTaskItemStatusUnknown
	}

	return generated.ScheduledTaskItem{
		Key:            strings.TrimSpace(task.Key),
		JobKey:         strings.TrimSpace(task.JobKey),
		ScheduleType:   generated.ScheduledTaskItemScheduleTypeCron,
		DisplayNameKey: strings.TrimSpace(task.DisplayMessageKey),
		DescriptionKey: strings.TrimSpace(task.DescriptionMessageKey),
		Owner:          strings.TrimSpace(task.ModuleKey),
		Module:         strings.TrimSpace(task.ModuleKey),
		Enabled:        task.Enabled,
		Builtin:        boolPointer(task.Builtin),
		Title:          stringPointer(task.Title),
		Description:    stringPointer(task.Description),
		ParamsJson:     stringPointer(task.ParamsJSON),
		Schedule:       strings.TrimSpace(task.Schedule),
		LastRun:        lastRun,
		Status:         status,
		Running:        task.Running,
	}
}

func toScheduledTaskLastRun(run schedulercore.TaskRun) generated.ScheduledTaskLastRun {
	return generated.ScheduledTaskLastRun{
		Id:            scheduledTaskRunID(run.ID),
		TriggerType:   generated.ScheduledTaskLastRunTriggerType(run.TriggerType),
		Status:        generated.ScheduledTaskLastRunStatus(run.Status),
		StartedAt:     run.StartedAt,
		FinishedAt:    run.FinishedAt,
		DurationMs:    run.DurationMS,
		ErrorSummary:  stringPointer(run.Error),
		ResultSummary: stringPointer(run.Result),
	}
}

func toScheduledTaskRunListResponse(
	result schedulercore.RunListResult,
	limit int,
	offset int,
) generated.ScheduledTaskRunListResponse {
	items := make([]generated.ScheduledTaskRunItem, 0, len(result.Items))
	for _, run := range result.Items {
		items = append(items, toScheduledTaskRunItem(run))
	}

	return generated.ScheduledTaskRunListResponse{
		Items:  items,
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	}
}

func toScheduledTaskRunItem(run schedulercore.TaskRun) generated.ScheduledTaskRunItem {
	return generated.ScheduledTaskRunItem{
		Id:            scheduledTaskRunID(run.ID),
		TaskKey:       strings.TrimSpace(run.TaskKey),
		TaskName:      strings.TrimSpace(run.TaskName),
		Owner:         strings.TrimSpace(run.Owner),
		Module:        strings.TrimSpace(run.Module),
		JobKey:        strings.TrimSpace(run.JobKey),
		TriggerType:   generated.ScheduledTaskRunItemTriggerType(run.TriggerType),
		Status:        generated.ScheduledTaskRunItemStatus(run.Status),
		ErrorSummary:  stringPointer(run.Error),
		ResultSummary: stringPointer(run.Result),
		StartedAt:     run.StartedAt,
		FinishedAt:    run.FinishedAt,
		DurationMs:    run.DurationMS,
		CreatedAt:     run.CreatedAt,
	}
}

func scheduledTaskRunID(id uint64) int64 {
	if id > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(id)
}

func boolPointer(value bool) *bool {
	return &value
}

func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func defaultJSONObject(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}
