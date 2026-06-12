// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"encoding/json"
	"math"
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/cronx"
	schedulercore "graft/server/internal/scheduler"
)

type scheduledTaskJobActionItem = struct {
	AffectedResource    *string                                               `json:"affected_resource,omitempty"`
	AffectedResourceKey *string                                               `json:"affected_resource_key,omitempty"`
	Behavior            *string                                               `json:"behavior,omitempty"`
	BehaviorKey         *string                                               `json:"behavior_key,omitempty"`
	BehaviorSummary     *string                                               `json:"behavior_summary,omitempty"`
	BehaviorSummaryKey  *string                                               `json:"behavior_summary_key,omitempty"`
	ConfirmRequired     *bool                                                 `json:"confirm_required,omitempty"`
	Description         *string                                               `json:"description,omitempty"`
	DescriptionKey      *string                                               `json:"description_key,omitempty"`
	Key                 string                                                `json:"key"`
	Theme               *generated.ScheduledTaskJobDefinitionItemActionsTheme `json:"theme,omitempty"`
	Title               *string                                               `json:"title,omitempty"`
	TitleKey            *string                                               `json:"title_key,omitempty"`
}

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
	item := generated.ScheduledTaskJobDefinitionItem{
		JobKey:         strings.TrimSpace(definition.JobKey),
		ModuleKey:      strings.TrimSpace(definition.ModuleKey),
		Category:       generated.ScheduledTaskJobDefinitionItemCategory(definition.Category),
		CategoryKey:    jobCategoryKey(definition.Category),
		TitleKey:       stringPointer(definition.TitleKey),
		Title:          strings.TrimSpace(definition.Title),
		ShortTitleKey:  stringPointer(definition.ShortTitleKey),
		ShortTitle:     strings.TrimSpace(definition.ShortTitle),
		DescriptionKey: stringPointer(definition.DescriptionKey),
		Description:    stringPointer(definition.Description),
		ConfigSchema:   defaultJSONObject(definition.ConfigSchema),
		DefaultConfig:  defaultJSONObject(definition.DefaultConfig),
		DefaultCron:    strings.TrimSpace(definition.DefaultCron),
		DefaultEnabled: definition.DefaultEnabled,
		Enabled:        definition.Enabled,
		Actions:        make([]scheduledTaskJobActionItem, 0, len(definition.Actions)),
	}
	for _, action := range definition.Actions {
		item.Actions = append(item.Actions, scheduledTaskJobActionItem{
			Key:            strings.TrimSpace(action.Key),
			TitleKey:       stringPointer(action.TitleKey),
			Title:          stringPointer(action.Title),
			DescriptionKey: stringPointer(action.DescriptionKey),
			Description:    stringPointer(action.Description),
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

	item := generated.ScheduledTaskItem{
		Id:              scheduledTaskRunID(task.ID),
		TaskKey:         strings.TrimSpace(task.Key),
		JobKey:          strings.TrimSpace(task.JobKey),
		TitleKey:        stringPointer(task.TitleKey),
		Title:           strings.TrimSpace(task.Title),
		DescriptionKey:  stringPointer(task.DescriptionKey),
		Description:     stringPointer(task.Description),
		CronExpression:  strings.TrimSpace(task.Schedule),
		Enabled:         task.Enabled,
		Builtin:         task.Builtin,
		ConfigJson:      defaultJSONObject(task.ConfigJSON),
		ConfigSource:    generated.ScheduledTaskItemConfigSource(task.ConfigSource),
		EffectiveConfig: defaultJSONObject(task.EffectiveConfig),
		LastRun:         lastRun,
		NextRunAt:       task.NextRunAt,
		Status:          status,
		Running:         task.Running,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
	if task.JobDefinition != nil {
		item.Job = toScheduledTaskJobDefinitionSummary(*task.JobDefinition)
	}
	return item
}

func toScheduledTaskJobDefinitionSummary(definition schedulercore.JobDefinitionSnapshot) *generated.ScheduledTaskJobDefinitionSummary {
	return &generated.ScheduledTaskJobDefinitionSummary{
		JobKey:         strings.TrimSpace(definition.JobKey),
		ModuleKey:      strings.TrimSpace(definition.ModuleKey),
		Category:       generated.ScheduledTaskJobDefinitionSummaryCategory(definition.Category),
		CategoryKey:    jobCategoryKey(definition.Category),
		TitleKey:       stringPointer(definition.TitleKey),
		Title:          strings.TrimSpace(definition.Title),
		ShortTitleKey:  stringPointer(definition.ShortTitleKey),
		ShortTitle:     strings.TrimSpace(definition.ShortTitle),
		DescriptionKey: stringPointer(definition.DescriptionKey),
		Description:    stringPointer(definition.Description),
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
		ErrorMessage:  stringPointer(run.ErrorMessage),
		ResultJson:    stringPointer(run.ResultJSON),
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
		Id:               scheduledTaskRunID(run.ID),
		TaskKey:          strings.TrimSpace(run.TaskKey),
		JobKey:           strings.TrimSpace(run.JobKey),
		TaskTitleKey:     stringPointer(run.TaskTitleKey),
		TaskTitle:        strings.TrimSpace(run.TaskTitle),
		JobTitleKey:      stringPointer(run.JobTitleKey),
		JobTitle:         strings.TrimSpace(run.JobTitle),
		JobShortTitleKey: stringPointer(run.JobShortTitleKey),
		JobShortTitle:    strings.TrimSpace(run.JobShortTitle),
		JobCategory:      generated.ScheduledTaskRunItemJobCategory(run.JobCategory),
		ModuleKey:        strings.TrimSpace(run.ModuleKey),
		TaskBuiltin:      run.TaskBuiltin,
		TriggerType:      generated.ScheduledTaskRunItemTriggerType(run.TriggerType),
		Status:           generated.ScheduledTaskRunItemStatus(run.Status),
		ErrorMessage:     stringPointer(run.ErrorMessage),
		ResultSummary:    stringPointer(run.Result),
		ResultJson:       stringPointer(run.ResultJSON),
		EffectiveConfig:  stringPointer(run.EffectiveConfig),
		StartedAt:        run.StartedAt,
		FinishedAt:       run.FinishedAt,
		DurationMs:       run.DurationMS,
		CreatedAt:        run.CreatedAt,
	}
}

func jobCategoryKey(category cronx.JobCategory) string {
	normalized := strings.TrimSpace(string(category))
	if normalized == "" {
		normalized = string(cronx.JobCategoryCustom)
	}
	return "scheduler.job.category." + normalized
}

func toScheduledTaskActionResult(result schedulercore.JobActionResult) generated.ScheduledTaskActionResult {
	safeResult := result.Result
	resultJSON, err := json.Marshal(safeResult)
	if err != nil {
		safeResult = cronx.JobRunResult{
			Summary:  "job action result serialization failed",
			Stage:    "failed",
			Warnings: []string{"job action result serialization failed"},
		}
		resultJSON, _ = json.Marshal(safeResult)
	}
	return generated.ScheduledTaskActionResult{
		ActionKey:       strings.TrimSpace(result.ActionKey),
		TaskKey:         strings.TrimSpace(result.TaskKey),
		JobKey:          strings.TrimSpace(result.JobKey),
		ResultJson:      string(resultJSON),
		Result:          toJobRunResult(safeResult),
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
