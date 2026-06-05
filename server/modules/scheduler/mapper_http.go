package scheduler

import (
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	schedulercore "graft/server/internal/scheduler"
)

func toScheduledTaskListResponse(tasks []schedulercore.TaskSnapshot) generated.ScheduledTaskListResponse {
	items := make([]generated.ScheduledTaskItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, toScheduledTaskItem(task))
	}

	return generated.ScheduledTaskListResponse{
		Items: items,
		Total: len(items),
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
		TaskType:       generated.ScheduledTaskItemTaskType(task.Type),
		ScheduleType:   generated.ScheduledTaskItemScheduleType(task.Type),
		DisplayNameKey: strings.TrimSpace(task.DisplayMessageKey),
		DescriptionKey: strings.TrimSpace(task.DescriptionMessageKey),
		Owner:          strings.TrimSpace(task.Owner),
		Module:         strings.TrimSpace(task.Module),
		Enabled:        task.DefaultEnabled,
		Schedule:       strings.TrimSpace(task.Schedule),
		LastRun:        lastRun,
		Status:         status,
		Running:        task.Running,
	}
}

func toScheduledTaskLastRun(run schedulercore.TaskRun) generated.ScheduledTaskLastRun {
	return generated.ScheduledTaskLastRun{
		Id:           run.ID,
		TriggerType:  generated.ScheduledTaskLastRunTriggerType(run.TriggerType),
		Status:       generated.ScheduledTaskLastRunStatus(run.Status),
		StartedAt:    run.StartedAt,
		FinishedAt:   run.FinishedAt,
		DurationMs:   run.DurationMS,
		ErrorSummary: strings.TrimSpace(run.Error),
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
		Id:           run.ID,
		TaskKey:      strings.TrimSpace(run.TaskKey),
		TaskName:     strings.TrimSpace(run.TaskName),
		Owner:        strings.TrimSpace(run.Owner),
		Module:       strings.TrimSpace(run.Module),
		TaskType:     generated.ScheduledTaskRunItemTaskType(run.TaskType),
		TriggerType:  generated.ScheduledTaskRunItemTriggerType(run.TriggerType),
		Status:       generated.ScheduledTaskRunItemStatus(run.Status),
		ErrorSummary: strings.TrimSpace(run.Error),
		StartedAt:    run.StartedAt,
		FinishedAt:   run.FinishedAt,
		DurationMs:   run.DurationMS,
		CreatedAt:    run.CreatedAt,
	}
}
