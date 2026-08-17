package runtimetarget

import (
	"context"
	"fmt"
	"strconv"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	contract "graft/server/modules/runtime-target/contract"
	store "graft/server/modules/runtime-target/store"
)

const (
	runtimeTargetSummaryWidgetID    = "runtime-target.fleet-summary"
	runtimeTargetSummaryWidgetOrder = 100
)

// registerRuntimeTargetDashboardWidget 注册运行目标目录摘要；该摘要不触发远端运行时探测。
func registerRuntimeTargetDashboardWidget(ctx *module.Context, repository *store.SQLRepository) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}
	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             runtimeTargetSummaryWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       "dashboard.widget.runtimeTargetSummary.title",
		DescriptionKey: "dashboard.widget.runtimeTargetSummary.description",
		Type:           dashboard.WidgetTypeStatGroup,
		Size:           dashboard.WidgetSizeSmall,
		Category:       dashboard.WidgetCategorySystem,
		Priority:       dashboard.WidgetPriorityNormal,
		Order:          runtimeTargetSummaryWidgetOrder,
		RouteLocation:  contract.MenuPath,
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.widget.runtimeTargetSummary.action",
			Route:    contract.MenuPath,
		},
		RequiredPermissions: []string{contract.ViewPermission},
		Loader: dashboard.WidgetLoaderFunc(func(loadCtx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			return loadRuntimeTargetSummaryWidget(loadCtx, repository)
		}),
	}); err != nil {
		return fmt.Errorf("register runtime target dashboard widget: %w", err)
	}
	return nil
}

// loadRuntimeTargetSummaryWidget 读取模块自有的单次 SQL 聚合，不连接 Docker 或其他远端 provider。
func loadRuntimeTargetSummaryWidget(ctx context.Context, repository *store.SQLRepository) (dashboard.WidgetPayload, error) {
	summary, err := repository.ReadSummary(ctx)
	if err != nil {
		return nil, err
	}
	state := dashboard.WidgetStateNormal
	priority := dashboard.WidgetPriorityNormal
	if summary.Unavailable > 0 {
		state = dashboard.WidgetStateWarning
		priority = dashboard.WidgetPriorityWarning
	}
	return dashboard.WidgetPayload{
		"items": []map[string]any{
			runtimeTargetSummaryStat("total", "Runtime targets", summary.Total, "info"),
			runtimeTargetSummaryStat("healthy", "Healthy targets", summary.Healthy, "success"),
			runtimeTargetSummaryStat("unavailable", "Unavailable targets", summary.Unavailable, runtimeTargetUnavailableTone(summary.Unavailable)),
		},
		"state":    string(state),
		"priority": string(priority),
	}, nil
}

func runtimeTargetSummaryStat(key, label string, value int64, tone string) map[string]any {
	return map[string]any{
		"key":             key,
		"label_key":       "dashboard.widget.runtimeTargetSummary." + key,
		"label":           label,
		"value":           strconv.FormatInt(value, 10),
		"tone":            tone,
		"description_key": "dashboard.widget.runtimeTargetSummary." + key + "Description",
		"route_location":  contract.MenuPath,
	}
}

func runtimeTargetUnavailableTone(count int64) string {
	if count > 0 {
		return "error"
	}
	return "success"
}
