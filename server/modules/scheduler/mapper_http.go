package scheduler

import (
	"encoding/json"
	"math"
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/cronx"
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
	item := generated.ScheduledTaskJobDefinitionItem{
		Key:                   strings.TrimSpace(definition.JobKey),
		Owner:                 moduleKey,
		Module:                moduleKey,
		DisplayNameKey:        strings.TrimSpace(definition.TitleKey),
		DescriptionKey:        strings.TrimSpace(definition.DescriptionKey),
		Title:                 stringPointer(definition.Title),
		Description:           stringPointer(definition.Description),
		ConfigSchemaJson:      defaultJSONObject(definition.ConfigSchema),
		DefaultConfigJson:     defaultJSONObject(definition.DefaultConfig),
		DefaultCronExpression: strings.TrimSpace(definition.DefaultCron),
		DefaultEnabled:        definition.Enabled,
		Actions: make([]struct {
			AffectedResource    *string                                               `json:"affected_resource,omitempty"`
			AffectedResourceKey *string                                               `json:"affected_resource_key,omitempty"`
			Behavior            *string                                               `json:"behavior,omitempty"`
			BehaviorKey         *string                                               `json:"behavior_key,omitempty"`
			BehaviorSummary     *string                                               `json:"behavior_summary,omitempty"`
			BehaviorSummaryKey  *string                                               `json:"behavior_summary_key,omitempty"`
			ConfigOverrides     *string                                               `json:"config_overrides,omitempty"`
			ConfirmRequired     *bool                                                 `json:"confirm_required,omitempty"`
			Description         *string                                               `json:"description,omitempty"`
			DescriptionKey      *string                                               `json:"description_key,omitempty"`
			DisplayNameKey      *string                                               `json:"display_name_key,omitempty"`
			Key                 string                                                `json:"key"`
			Theme               *generated.ScheduledTaskJobDefinitionItemActionsTheme `json:"theme,omitempty"`
			Title               *string                                               `json:"title,omitempty"`
		}, 0, len(definition.Actions)),
	}
	for _, action := range definition.Actions {
		item.Actions = append(item.Actions, struct {
			AffectedResource    *string                                               `json:"affected_resource,omitempty"`
			AffectedResourceKey *string                                               `json:"affected_resource_key,omitempty"`
			Behavior            *string                                               `json:"behavior,omitempty"`
			BehaviorKey         *string                                               `json:"behavior_key,omitempty"`
			BehaviorSummary     *string                                               `json:"behavior_summary,omitempty"`
			BehaviorSummaryKey  *string                                               `json:"behavior_summary_key,omitempty"`
			ConfigOverrides     *string                                               `json:"config_overrides,omitempty"`
			ConfirmRequired     *bool                                                 `json:"confirm_required,omitempty"`
			Description         *string                                               `json:"description,omitempty"`
			DescriptionKey      *string                                               `json:"description_key,omitempty"`
			DisplayNameKey      *string                                               `json:"display_name_key,omitempty"`
			Key                 string                                                `json:"key"`
			Theme               *generated.ScheduledTaskJobDefinitionItemActionsTheme `json:"theme,omitempty"`
			Title               *string                                               `json:"title,omitempty"`
		}{
			Key:             strings.TrimSpace(action.Key),
			Title:           stringPointer(action.Title),
			Description:     stringPointer(action.Description),
			ConfigOverrides: stringPointer(defaultJSONObject(action.ConfigOverrides)),
		})
	}
	return item
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
		Key:             strings.TrimSpace(task.Key),
		JobKey:          strings.TrimSpace(task.JobKey),
		ScheduleType:    generated.ScheduledTaskItemScheduleTypeCron,
		DisplayNameKey:  strings.TrimSpace(task.DisplayMessageKey),
		DescriptionKey:  strings.TrimSpace(task.DescriptionMessageKey),
		Owner:           strings.TrimSpace(task.ModuleKey),
		Module:          strings.TrimSpace(task.ModuleKey),
		Enabled:         task.Enabled,
		Builtin:         boolPointer(task.Builtin),
		Title:           stringPointer(task.Title),
		Description:     stringPointer(task.Description),
		ConfigJson:      stringPointer(task.ConfigJSON),
		EffectiveConfig: stringPointer(task.EffectiveConfig),
		Schedule:        strings.TrimSpace(task.Schedule),
		LastRun:         lastRun,
		Status:          status,
		Running:         task.Running,
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
		Id:              scheduledTaskRunID(run.ID),
		TaskKey:         strings.TrimSpace(run.TaskKey),
		TaskName:        strings.TrimSpace(run.TaskName),
		Owner:           strings.TrimSpace(run.Owner),
		Module:          strings.TrimSpace(run.Module),
		JobKey:          strings.TrimSpace(run.JobKey),
		TriggerType:     generated.ScheduledTaskRunItemTriggerType(run.TriggerType),
		Status:          generated.ScheduledTaskRunItemStatus(run.Status),
		ErrorSummary:    stringPointer(run.Error),
		ResultSummary:   stringPointer(run.Result),
		ResultJson:      stringPointer(run.ResultJSON),
		EffectiveConfig: stringPointer(run.EffectiveConfig),
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		DurationMs:      run.DurationMS,
		CreatedAt:       run.CreatedAt,
	}
}

func toScheduledTaskActionResult(result schedulercore.JobActionResult) generated.ScheduledTaskActionResult {
	resultJSON, _ := json.Marshal(result.Result)
	return generated.ScheduledTaskActionResult{
		ActionKey:       strings.TrimSpace(result.ActionKey),
		TaskKey:         strings.TrimSpace(result.TaskKey),
		JobKey:          strings.TrimSpace(result.JobKey),
		ResultJson:      string(resultJSON),
		Result:          toJobRunResult(result.Result),
		EffectiveConfig: stringPointer(result.EffectiveConfig),
	}
}

func toJobRunResult(result cronx.JobRunResult) generated.JobRunResult {
	return generated.JobRunResult{
		Summary:          stringPointer(result.Summary),
		Stage:            stringPointer(result.Stage),
		AffectedResource: stringPointer(result.AffectedResource),
		Metrics:          mapPointer(result.Metrics),
		Details:          mapPointer(result.Details),
		Warnings:         slicePointer(result.Warnings),
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

func mapPointer(value map[string]any) *map[string]any {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func slicePointer(value []string) *[]string {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func defaultJSONObject(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}
