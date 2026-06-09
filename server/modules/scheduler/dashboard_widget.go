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
	schedulerTaskQuickLinkID          = "scheduler.scheduled-tasks"
	schedulerTaskQuickLinkOrder       = 110
)

func registerSchedulerDashboardWidget(ctx *module.Context, instance *Module) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}

	if err := ctx.DashboardRegistry.RegisterQuickLink(schedulerTaskQuickLink()); err != nil {
		return fmt.Errorf("register scheduler dashboard quick link: %w", err)
	}

	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             schedulerTaskAttentionWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       "dashboard.widget.schedulerTaskAttention.title",
		Title:          "Scheduled Task Attention",
		DescriptionKey: "dashboard.widget.schedulerTaskAttention.description",
		Description:    "Failed, running, and disabled scheduled tasks that need operator attention.",
		Type:           dashboard.WidgetTypeStatGroup,
		Size:           dashboard.WidgetSizeMedium,
		Category:       dashboard.WidgetCategoryOperation,
		Priority:       dashboard.WidgetPriorityWarning,
		Order:          schedulerTaskAttentionWidgetOrder,
		RouteLocation:  schedulercontract.ScheduledTaskMenuPath,
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.actions.details",
			Label:    "View details",
			Route:    schedulercontract.ScheduledTaskMenuPath,
		},
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

func schedulerTaskQuickLink() dashboard.QuickLinkDefinition {
	return dashboard.QuickLinkDefinition{
		ID:                  schedulerTaskQuickLinkID,
		ModuleKey:           moduleID,
		TitleKey:            schedulercontract.ScheduledTaskMenuTitle.String(),
		Title:               "Scheduled Tasks",
		Icon:                "time",
		RouteLocation:       schedulercontract.ScheduledTaskMenuPath,
		RequiredPermissions: []string{schedulercontract.ScheduledTaskReadPermission.String()},
		Order:               schedulerTaskQuickLinkOrder,
	}
}

func loadSchedulerTaskAttentionWidget(ctx context.Context, runtime schedulercore.Runtime) (dashboard.WidgetPayload, error) {
	counts, err := schedulerAttentionCounts(ctx, runtime)
	if err != nil {
		return nil, err
	}

	visible := counts.failed > 0 || counts.running > 0 || counts.disabled > 0
	state, priority := schedulerAttentionState(counts)

	return dashboard.WidgetPayload{
		"items": []map[string]any{
			schedulerAttentionStat("failed", "Failed tasks", counts.failed, "error", "Last run failed."),
			schedulerAttentionStat("running", "Running tasks", counts.running, "info", "Currently executing tasks."),
			schedulerAttentionStat("disabled", "Disabled tasks", counts.disabled, "warning", "Tasks disabled from execution."),
		},
		"visible":      visible,
		"state":        string(state),
		"priority":     string(priority),
		"failed_tasks": counts.failed,
	}, nil
}

type schedulerAttentionCounters struct {
	failed   int
	running  int
	disabled int
}

func schedulerAttentionCounts(ctx context.Context, runtime schedulercore.Runtime) (schedulerAttentionCounters, error) {
	counts := schedulerAttentionCounters{}
	offset := 0
	for {
		tasks, err := runtime.ListTasks(ctx, schedulercore.TaskListQuery{
			Limit:  schedulerTaskAttentionListLimit,
			Offset: offset,
		})
		if err != nil {
			return schedulerAttentionCounters{}, err
		}

		for _, task := range tasks.Items {
			countSchedulerAttentionTask(&counts, task)
		}

		offset += len(tasks.Items)
		if len(tasks.Items) == 0 || offset >= tasks.Total {
			break
		}
	}
	return counts, nil
}

func countSchedulerAttentionTask(counts *schedulerAttentionCounters, task schedulercore.TaskSnapshot) {
	if counts == nil {
		return
	}
	if !task.Enabled {
		counts.disabled++
	}
	if task.Running {
		counts.running++
	}
	if task.LastRun != nil && task.LastRun.Status == schedulercore.RunStatusFailed {
		counts.failed++
	}
}

func schedulerAttentionState(counts schedulerAttentionCounters) (dashboard.WidgetState, dashboard.WidgetPriority) {
	state := dashboard.WidgetStateHidden
	priority := dashboard.WidgetPriorityNormal
	if counts.failed > 0 || counts.running > 0 || counts.disabled > 0 {
		state = dashboard.WidgetStateWarning
		priority = dashboard.WidgetPriorityWarning
	}
	if counts.failed > 0 {
		state = dashboard.WidgetStateCritical
		priority = dashboard.WidgetPriorityCritical
	}
	return state, priority
}

func schedulerAttentionStat(key string, label string, value int, tone string, description string) map[string]any {
	return map[string]any{
		"key":             key,
		"label_key":       "dashboard.widget.schedulerTaskAttention." + key,
		"label":           label,
		"value":           strconv.Itoa(value),
		"tone":            tone,
		"description_key": "dashboard.widget.schedulerTaskAttention." + key + "Description",
		"description":     description,
		"route_location":  schedulercontract.ScheduledTaskMenuPath,
	}
}
