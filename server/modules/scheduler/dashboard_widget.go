package scheduler

import (
	"context"
	"fmt"
	"strconv"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	schedulercore "graft/server/internal/scheduler"
	schedulercontract "graft/server/modules/scheduler/contract"
)

const (
	schedulerTaskAttentionWidgetID    = "scheduler.task-attention"
	schedulerTaskAttentionWidgetOrder = 120
	schedulerTaskAttentionListLimit   = 100
)

func registerSchedulerDashboardWidget(ctx *module.Context, instance *Module) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}

	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:                  schedulerTaskAttentionWidgetID,
		ModuleKey:           moduleID,
		TitleKey:            "dashboard.widget.schedulerTaskAttention.title",
		Title:               "Scheduled Task Attention",
		DescriptionKey:      "dashboard.widget.schedulerTaskAttention.description",
		Description:         "Failed, running, and disabled scheduled tasks that need operator attention.",
		Type:                dashboard.WidgetTypeStatGroup,
		Size:                dashboard.WidgetSizeMedium,
		Order:               schedulerTaskAttentionWidgetOrder,
		RouteLocation:       schedulercontract.ScheduledTaskMenuPath,
		RequiredPermissions: []string{schedulercontract.ScheduledTaskReadPermission.String()},
		Loader: dashboard.WidgetLoaderFunc(func(loadCtx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			runtime, err := instance.resolveRuntime(ctx)
			if err != nil {
				return nil, err
			}
			return loadSchedulerTaskAttentionWidget(loadCtx, runtime)
		}),
	}); err != nil {
		return fmt.Errorf("register scheduler dashboard widget: %w", err)
	}

	return nil
}

func loadSchedulerTaskAttentionWidget(ctx context.Context, runtime schedulercore.Runtime) (dashboard.WidgetPayload, error) {
	tasks, err := runtime.ListTasks(ctx, schedulercore.TaskListQuery{Limit: schedulerTaskAttentionListLimit})
	if err != nil {
		return nil, err
	}

	failed := 0
	running := 0
	disabled := 0
	for _, task := range tasks.Items {
		if !task.Enabled {
			disabled++
		}
		if task.Running {
			running++
		}
		if task.LastRun != nil && task.LastRun.Status == schedulercore.RunStatusFailed {
			failed++
		}
	}

	return dashboard.WidgetPayload{
		"items": []map[string]any{
			schedulerAttentionStat("failed", "Failed tasks", failed, "error", "Last run failed."),
			schedulerAttentionStat("running", "Running tasks", running, "info", "Currently executing tasks."),
			schedulerAttentionStat("disabled", "Disabled tasks", disabled, "warning", "Tasks disabled from execution."),
		},
	}, nil
}

func schedulerAttentionStat(key string, label string, value int, tone string, description string) map[string]any {
	return map[string]any{
		"key":            key,
		"label_key":      "dashboard.widget.schedulerTaskAttention." + key,
		"label":          label,
		"value":          strconv.Itoa(value),
		"tone":           tone,
		"description":    description,
		"route_location": schedulercontract.ScheduledTaskMenuPath,
	}
}
